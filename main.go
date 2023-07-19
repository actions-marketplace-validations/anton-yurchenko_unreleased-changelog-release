package main

import (
	"fmt"
	"os"
	"os/exec"
)

const Version string = "0.1.0"

func init() {
	vars := []string{
		"GITHUB_REPOSITORY",
		"GITHUB_TOKEN",
		"GITHUB_ACTOR",
		"VERSION",
	}

	for i, v := range vars {
		if os.Getenv(v) == "" {
			fatal("missing required environmental variable: %s", vars[i])
		}
	}
}

func wrap(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

func fatal(format string, a ...any) {
	fmt.Println(wrap(format, a...))
	os.Exit(1)
}

func output(k, v string) error {
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("echo \"%s=%s\" >> $GITHUB_OUTPUT", k, v))
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func main() {
	fmt.Println("- initializing action")
	a, err := new()
	if err != nil {
		fatal("initialization error: %s", err)
	}

	fmt.Println("- updating changelog file")
	if err := a.updateChangelog(); err != nil {
		fatal("error updating changelog file: %s", err)
	}

	fmt.Println("- committing changes")
	commit, err := a.commit()
	if err != nil {
		fatal("error committing changes: %s", err)
	}

	fmt.Println("- creating tags")
	if err := a.tag(commit); err != nil {
		fatal("error creating tags: %s", err)
	}

	fmt.Println("- pushing changes")
	if err := a.push(); err != nil {
		fatal("error pushing changes: %s", err)
	}

	o := map[string]string{
		"hash": commit.String(),
		"tag":  a.version,
	}
	for k, v := range o {
		fmt.Printf("- creating output: %s=%s\n", k, v)
		if err := output(k, v); err != nil {
			fatal("error defining output (%s=%s): %s", k, v, err)
		}
	}
	fmt.Println("- success")
}
