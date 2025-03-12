# repo-opener

[![golangci-lint](https://github.com/jtprogru/repo-opener/actions/workflows/lint.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/lint.yaml)
[![testing](https://github.com/jtprogru/repo-opener/actions/workflows/tests.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/tests.yaml)
[![goreleaser](https://github.com/jtprogru/repo-opener/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/goreleaser.yaml)
[![bearer](https://github.com/jtprogru/repo-opener/actions/workflows/bearer.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/bearer.yaml)
[![GitHub stars](https://img.shields.io/github/stars/jtprogru/repo-opener?style=plastic&color=5BB359)](https://github.com/jtprogru/repo-opener/stargazers)

Simple utility for fast open current repository in browser.

## Installation

For installation you need to load latest version from [Release](https://github.com/jtprogru/repo-opener/releases) page and download version for you platform.

Another way is usage `go install`:

```sh
# Get latest version from CLI
VERSION=`curl -sSL https://api.github.com/repos/jtprogru/repo-opener/releases/latest -s | jq .name -r`
go install github.com/jtprogru/repo-opener@$VERSION
```

## Usage

To use `repo-opener`, simply run the following command in your terminal:

```sh
repo-opener
```

This will open the current Git repository's remote URL in your default web browser.

To make life easier, you can add an alias like this:

```sh
echo "alias rop=$(which repo-opener)" >> $HOME/.aliases
```

After which you will have the command `rop` for running `repo-opener`.

### Options

- `-version`: Print version and build information and exit.

## License

[LICENSE](LICENSE)
