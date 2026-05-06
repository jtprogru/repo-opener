# repo-opener

[![golangci-lint](https://github.com/jtprogru/repo-opener/actions/workflows/lint.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/lint.yaml)
[![testing](https://github.com/jtprogru/repo-opener/actions/workflows/tests.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/tests.yaml)
[![goreleaser](https://github.com/jtprogru/repo-opener/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/goreleaser.yaml)
[![bearer](https://github.com/jtprogru/repo-opener/actions/workflows/bearer.yaml/badge.svg)](https://github.com/jtprogru/repo-opener/actions/workflows/bearer.yaml)
[![GitHub stars](https://img.shields.io/github/stars/jtprogru/repo-opener?style=plastic&color=5BB359)](https://github.com/jtprogru/repo-opener/stargazers)

Simple utility that prints the current Git repository's remote HTTP URL — works for public hosts (GitHub/GitLab/Bitbucket) and self-hosted installations alike. Optionally opens the URL in your default browser.

## Installation

### Homebrew (macOS / Linux)

```sh
brew install jtprogru/tap/repo-opener
```

### Pre-built binaries

Download the archive for your platform from the [Releases](https://github.com/jtprogru/repo-opener/releases) page.

### `go install`

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

This prints the current Git repository's remote HTTP URL to stdout. To also open it in your default browser, pass `-o`/`--open`.

To make life easier, you can add an alias like this:

```sh
echo "alias rop=$(which repo-opener)" >> $HOME/.aliases
```

After which you will have the command `rop` for running `repo-opener`.

### Options

- `-version`: Print version and build information and exit.
- `-o`, `--open`: Print the URL and also open it in the default browser.
- `-remote <name>`: Specify remote name to use (default: `origin`).

### Examples

```sh
# Print current repository URL (default behavior)
repo-opener
# Output: https://github.com/user/repo

# Print and open the URL in the default browser
repo-opener -o
repo-opener --open

# Use a different remote (e.g., upstream)
repo-opener -remote upstream

# Combine flags
repo-opener --open -remote upstream
```

## Supported Git Hosts

### Public Platforms

| Host | SSH | HTTPS |
|------|-----|-------|
| GitHub | ✅ | ✅ |
| GitLab | ✅ | ✅ |
| Bitbucket | ✅ | ✅ |

### Private Installations

| Platform | Example |
|----------|---------|
| GitLab CE/EE | `gitlab.internal.company` |
| GitHub Enterprise | `github.internal.company` |
| Gitea | `gitea.example.com` |
| Forgejo | `forgejo.example.com` |
| Custom | `git.jtprog.ru` |

> **Note:** Private installations are **not normalized** — the host is preserved as-is.

## Troubleshooting

### Error: origin remote not found

**Cause:** The repository doesn't have a remote named `origin`.

**Solution:**
```sh
# Check existing remotes
git remote -v

# Add origin remote
git remote add origin <repository-url>
```

### Error: not a git repository

**Cause:** You're not in a Git repository directory.

**Solution:**
```sh
# Check if you're in a git repository
git status

# Navigate to a repository directory
cd /path/to/your/repo

# Or initialize a new repository
git init
```

### Error: failed to open browser

**Cause:** The system cannot open the default browser.

**Solution:**
- Ensure you have a default browser configured
- On headless systems, simply omit `-o`/`--open` — the default behavior only prints the URL

### Private GitLab/GitHub Enterprise not working

**Cause:** Custom Git hosts should work automatically. If issues occur:

**Solution:**
```sh
# Verify remote URL
git remote -v

# Ensure the URL is accessible in browser
# SSH format: git@host.com:org/repo.git
# HTTPS format: https://host.com/org/repo.git
```

## License

[LICENSE](LICENSE)
