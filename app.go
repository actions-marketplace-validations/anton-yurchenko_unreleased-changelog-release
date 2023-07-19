package main

import (
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	changelog "github.com/anton-yurchenko/go-changelog"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/afero"
	"golang.org/x/mod/semver"
)

const dateFormat string = `2006-01-02`

type app struct {
	version   string
	refs      []string
	changelog changelogFile
	git       gitRepo
}

type changelogFile struct {
	filename string
	parser   *changelog.Parser
	content  *changelog.Changelog
}

type gitRepo struct {
	committer *object.Signature
	author    *object.Signature
	auth      *githttp.BasicAuth
	repo      *git.Repository
	worktree  *git.Worktree
}

func new() (*app, error) {
	v := os.Getenv("VERSION")
	if !semver.IsValid(v) {
		return nil, fmt.Errorf("invalid semantic version (make sure to add a 'v' prefix: vX.X.X)")
	}

	refs := []string{v}
	u, err := strconv.ParseBool(os.Getenv("UPDATE_TAGS"))
	if err != nil {
		return nil, wrap("error parsing UPDATE_TAGS environmental variable: %s", err)
	}
	if u {
		refs = append(refs, []string{
			semver.Major(v),
			semver.MajorMinor(v),
		}...)
	}

	f := "CHANGELOG.md"
	if x := os.Getenv("CHANGELOG_FILE"); x != "" {
		f = x
	}

	p, err := changelog.NewParser(f)
	if err != nil {
		return nil, wrap("error initializing changelog parser: %s", err)
	}

	c, err := p.Parse()
	if err != nil {
		return nil, wrap("error parsing changelog: %s", err)
	}

	g, err := newGitRepo()
	if err != nil {
		return nil, wrap("git configuration error: %s", err)
	}

	return &app{
		version: v,
		refs:    refs,
		changelog: changelogFile{
			filename: f,
			parser:   p,
			content:  c,
		},
		git: g,
	}, nil
}

func newGitRepo() (gitRepo, error) {
	o := gitRepo{}
	r, err := git.PlainOpen(".")
	if err != nil {
		return o, wrap("error opening repository: %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return o, wrap("error identifying worktree: %s", err)
	}

	tm := time.Now()
	committer := &object.Signature{
		Name:  "github-actions[bot]",
		Email: "github-actions[bot]@users.noreply.github.com",
		When:  tm,
	}

	author := &object.Signature{
		Name: os.Getenv("GITHUB_ACTOR"),
		When: tm,
	}

	if author.Name == "" {
		author = committer
	} else {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/users/%s", author.Name), nil)
		if err != nil {
			return o, wrap("error creating a api request to fetch author information: %s", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("GITHUB_TOKEN")))
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return o, wrap("error fetching author information: %s", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return o, wrap("error contacting github api: http code %v", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return o, wrap("error reading response body: %s", err)
		}

		s := struct {
			Email string `json:"email"`
		}{}

		if err := json.Unmarshal(body, &s); err != nil {
			return o, wrap("error decoding response body: %s", err)
		}

		if s.Email != "" {
			author.Email = s.Email
		}
	}

	return gitRepo{
		committer: committer,
		author:    author,
		auth: &githttp.BasicAuth{
			Username: committer.Name,
			Password: os.Getenv("GITHUB_TOKEN"),
		},
		repo:     r,
		worktree: w,
	}, nil
}

func (a *app) updateChangelog() error {
	url := fmt.Sprintf("https://github.com/%s", os.Getenv("GITHUB_REPOSITORY"))

	releaseURL := fmt.Sprintf("%s/releases/tag/%s", url, a.version)
	if len(a.changelog.content.Releases) > 0 {
		t := a.changelog.content.Releases
		sort.Sort(t)
		if t[len(t)-1].Version != nil {
			releaseURL = fmt.Sprintf("%s/compare/v%s...%s", url, *t[len(t)-1].Version, a.version)
		}
	}

	// TODO: get latest release
	if _, err := a.changelog.content.CreateReleaseFromUnreleasedWithURL(
		strings.TrimPrefix(a.version, "v"),
		time.Now().Format(dateFormat),
		releaseURL,
	); err != nil {
		return wrap("error creating release from an unreleased: %s", err)
	}

	if err := a.changelog.content.SetUnreleasedURL(fmt.Sprintf("%s/compare/%s...HEAD", url, a.version)); err != nil {
		return wrap("error updating unreleased url: %s", err)
	}

	if err := a.changelog.content.SaveToFile(afero.NewOsFs(), a.changelog.filename); err != nil {
		return wrap("error saving changelog to file: %s", err)
	}

	return nil
}

func (a *app) commit() (plumbing.Hash, error) {
	if _, err := a.git.worktree.Add(a.changelog.filename); err != nil {
		return plumbing.Hash{}, wrap("error staging %s file: %s", a.changelog.filename, err)
	}

	c, err := a.git.worktree.Commit(
		strings.TrimPrefix(a.version, "v"),
		&git.CommitOptions{
			Author:    a.git.author,
			Committer: a.git.committer,
		},
	)
	if err != nil {
		return plumbing.Hash{}, wrap("error committing changes: %s", err)
	}

	return c, nil
}

func (a *app) checkVersionTagExistence() (bool, error) {
	tags, err := a.git.repo.TagObjects()
	if err != nil {
		return false, wrap("error fetching tags: %s", err)
	}
	res := false
	err = tags.ForEach(func(t *object.Tag) error {
		if t.Name == a.version {
			res = true
			return git.ErrTagExists
		}
		return nil
	})
	if err != nil && err != git.ErrTagExists {
		return false, wrap("tags iterator error: %s", err)
	}
	return res, nil
}

func (a *app) tag(commit plumbing.Hash) error {
	exists, err := a.checkVersionTagExistence()
	if err != nil {
		return wrap("error validating tag existence: %s", err)
	}
	if exists {
		return git.ErrTagExists
	}

	h, err := a.git.repo.Head()
	if err != nil {
		return wrap("error identifying head reference: %s", err)
	}

	for _, v := range a.refs {
		_, err = a.git.repo.CreateTag(v, h.Hash(), &git.CreateTagOptions{
			Message: v,
			Tagger:  a.git.committer,
		})
		if err != nil {
			if v != a.version && err == git.ErrTagExists {
				if err := a.git.repo.DeleteTag(v); err != nil {
					return wrap("error deleting tag: %s", err)
				}

				if _, err = a.git.repo.CreateTag(v, h.Hash(), &git.CreateTagOptions{
					Message: v,
					Tagger:  a.git.committer,
				}); err != nil {
					return wrap("error tagging a commit (%s): %s", v, err)
				}

				continue
			}

			return wrap("error tagging a commit (%s): %s", a.version, err)
		}
	}

	return nil
}

func (a *app) push() error {
	if err := a.git.repo.Push(&git.PushOptions{Auth: a.git.auth}); err != nil {
		return wrap("error pushing the commit: %s", err)
	}

	for _, v := range a.refs {
		o := &git.PushOptions{
			RefSpecs: []config.RefSpec{config.RefSpec(
				fmt.Sprintf("refs/tags/%s:refs/tags/%s", v, v),
			)},
			Auth: a.git.auth,
		}

		if v != a.version {
			o.Force = true
		}

		err := a.git.repo.Push(o)
		if err != nil {
			return wrap("error pushing the tag (%s): %s", v, err)
		}
	}

	return nil
}
