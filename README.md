# unreleased-changelog-release

[![Release](https://img.shields.io/github/v/release/anton-yurchenko/unreleased-changelog-release)](https://github.com/anton-yurchenko/unreleased-changelog-release/releases/latest)
[![Code Coverage](https://codecov.io/gh/anton-yurchenko/unreleased-changelog-release/branch/main/graph/badge.svg)](https://codecov.io/gh/anton-yurchenko/unreleased-changelog-release)
[![Go Report Card](https://goreportcard.com/badge/github.com/anton-yurchenko/unreleased-changelog-release)](https://goreportcard.com/report/github.com/anton-yurchenko/unreleased-changelog-release)
[![Release](https://github.com/anton-yurchenko/unreleased-changelog-release/actions/workflows/release.yml/badge.svg)](https://github.com/anton-yurchenko/unreleased-changelog-release/actions/workflows/release.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/antonyurchenko/unreleased-changelog-release)](https://hub.docker.com/r/antonyurchenko/unreleased-changelog-release)
[![License](https://img.shields.io/github/license/anton-yurchenko/unreleased-changelog-release)](LICENSE.md)

A **GitHub Action** for a new release creation on **changelog file** using an **Unreleased section**.  

![PIC](docs/images/demo.gif)

## Features

- Parse input to match [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/) Compliant
- Supports GitHub Enterprise
- Supports standard `v` prefix out of the box

## Manual

1. Add changes to the **Unreleased** section in `CHANGELOG.md` file in your feature branch. *For example:*

    ```markdown
    ## [Unreleased]

    ### Added
    - Glob pattern support
    - Unit Tests
    - Log version
    
    ### Fixed
    - Exception on margins larger than context of changelog
    - Nil pointer exception in 'release' package
    
    ### Changed
    - Refactor JavaScript wrapper
    ```

2. Review the changes and merge Pull Request
3. Manually start a GitHub Actions Worfklow providing a desired Semantic Version (**including `v` prefix**)
4. **Changelog-Release** will update the Changelog file, create Tags and push the changes :wink:  
    a. Optionally trigger another workflow using the newly arrived Version Tag

## Configuration

1. Change the workflow to be triggered manually with the required inputs:

    ```yaml
    on:
      workflow_dispatch:
        inputs:
          version:
            type: string
            description: "Semantic Version (X.X.X)"
            required: true
    ```

2. Add *Release* step to your workflow:

    ```yaml
        - name: Update Changelog
          uses: docker://antonyurchenko/unreleased-changelog-release:v1
          env:
            GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
            VERSION: ${{ github.event.inputs.version }}
            UPDATE_TAGS: "true"
    ```

3. Configure *Release* step:

    | Environmental Variable  | Allowed Values    | Default Value     | Description                                |
    |:-----------------------:|:-----------------:|:-----------------:|:------------------------------------------:|
    | `CHANGELOG_FILE`        | `*`               | `CHANGELOG.md`    | Changelog filename                         |
    | `UPDATE_TAGS`           | `true`/`false`    | `false`           | Update major and minor tags (`vX`, `vX.X`) |

## Remarks

- This action has multiple tags: `latest / v1 / v1.2 / v1.2.3`. You may lock to a certain version instead of using **latest**.  
(*Recommended to lock against a major version, for example* `v4`)
- Docker image is published both to [**Docker Hub**](https://hub.docker.com/r/antonyurchenko/unreleased-changelog-release) and [**GitHub Packages**](https://github.com/anton-yurchenko/unreleased-changelog-release/packages). If you don't want to rely on **Docker Hub** but still want to use the dockerized action, you may switch from `uses: docker://antonyurchenko/unreleased-changelog-release:latest` to `uses: docker://ghcr.io/anton-yurchenko/unreleased-changelog-release:latest`
- `unreleased-changelog-release` may crash when executed against a not supported changelog file format. Make sure your changelog file is compliant to one of the supported formats.

## License

[MIT](LICENSE.md) © 2023-present Anton Yurchenko
