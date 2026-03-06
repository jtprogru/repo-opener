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
- `-print`: Print URL to stdout and exit (without opening browser).
- `-dry-run`: Show what URL would be opened (without opening browser).
- `-remote <name>`: Specify remote name to use (default: `origin`).

### Examples

```sh
# Open current repository in browser
repo-opener

# Print URL without opening browser
repo-opener -print
# Output: https://github.com/user/repo

# Show what would be opened (dry run)
repo-opener -dry-run
# Output: Would open: https://github.com/user/repo

# Use different remote (e.g., upstream)
repo-opener -remote upstream

# Combine flags
repo-opener -print -remote origin
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
- On headless systems, consider using `-print` flag (if available)

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
