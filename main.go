package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/pkg/browser"
)

const (
	ErrOriginRemoteNotFound = "origin remote not found"
	ErrEmptyRepositoryPath  = "empty repository path"

	GitHubHost    = "github.com"
	GitLabHost    = "gitlab.com"
	BitbucketHost = "bitbucket.org"
)

var (
	Version     = "dev"     //nolint:gochecknoglobals // This is normal
	Commit      = "none"    //nolint:gochecknoglobals // This is normal
	Date        = "unknown" //nolint:gochecknoglobals // This is normal
	BuiltBy     = "unknown" //nolint:gochecknoglobals // This is normal
	versionFlag bool        //nolint:gochecknoglobals // This is normal
	remoteName  string      //nolint:gochecknoglobals // This is normal
	dryRunFlag  bool        //nolint:gochecknoglobals // This is normal
	printFlag   bool        //nolint:gochecknoglobals // This is normal
)

func main() {
	flag.BoolVar(&versionFlag, "version", false, "Print version information and exit")
	flag.BoolVar(&dryRunFlag, "dry-run", false, "Show URL without opening browser")
	flag.BoolVar(&printFlag, "print", false, "Print URL to stdout and exit")
	flag.StringVar(&remoteName, "remote", "origin", "Remote name to use (default: origin)")

	flag.Parse()

	if versionFlag {
		fmt.Println("Version:", Version)
		fmt.Printf("Build info: commit %s, built at %s, by %s\n", Commit, Date, BuiltBy)
		return
	}

	// Открываем Git-репозиторий.
	repo, err := git.PlainOpen(".")
	if err != nil {
		exitWithError(fmt.Errorf("not a git repository: %w", err))
	}

	// Получаем URL remote.
	remoteURL, err := getRemoteURL(repo, remoteName)
	if err != nil {
		exitWithError(err)
	}

	// Парсим URL и конвертируем в веб-формат.
	webURL, err := parseRemoteURL(remoteURL)
	if err != nil {
		exitWithError(err)
	}

	// Режим -print: только выводим URL.
	if printFlag {
		fmt.Println(webURL)
		return
	}

	// Режим -dry-run: показываем, что сделали бы.
	if dryRunFlag {
		fmt.Printf("Would open: %s\n", webURL)
		return
	}

	// Открываем URL в браузере.
	if err := browser.OpenURL(webURL); err != nil {
		exitWithError(fmt.Errorf("failed to open browser: %w", err))
	}

	fmt.Printf("Opened repository URL: %s\n", webURL)
}

func getRemoteURL(repo *git.Repository, name string) (string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		if remote.Config().Name == name {
			if len(remote.Config().URLs) > 0 {
				return remote.Config().URLs[0], nil
			}
		}
	}

	return "", fmt.Errorf("remote %q not found", name)
}

func parseRemoteURL(remoteURL string) (string, error) {
	// Пытаемся парсить как стандартный URL.
	if u, err := url.Parse(remoteURL); err == nil && u.Scheme != "" {
		return parseStructuredURL(u)
	}

	// Обрабатываем SSH-формат (git@host:path).
	if strings.HasPrefix(remoteURL, "git@") {
		return parseSSHURL(remoteURL)
	}

	return "", fmt.Errorf("unsupported URL format: %s", remoteURL)
}

func parseStructuredURL(u *url.URL) (string, error) {
	var host, path string

	switch u.Scheme {
	case "http", "https":
		host = u.Host
		path = u.Path
	case "ssh":
		host = u.Host
		path = u.Path
		if u.User != nil && u.User.Username() != "git" {
			return "", fmt.Errorf("unsupported SSH username: %s", u.User.Username())
		}
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	if path == "" {
		return "", errors.New("empty repository path")
	}

	return buildWebURL(host, path)
}

func parseSSHURL(remoteURL string) (string, error) {
	parts := strings.SplitN(remoteURL, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid SSH URL format")
	}

	host := strings.TrimPrefix(parts[0], "git@")
	path := strings.Trim(parts[1], "/")

	if path == "" {
		return "", errors.New(ErrEmptyRepositoryPath)
	}

	if len(strings.Split(path, "/")) < 2 {
		return "", errors.New("invalid SSH URL format")
	}

	return buildWebURL(host, path)
}

func buildWebURL(hostOrigin, path string) (string, error) {
	// Создаем копию hostOrigin, чтобы не изменять оригинальную строку.
	host := hostOrigin

	// Удаляем .git в конце пути и ведущие/конечные слэши.
	cleanPath := strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	if cleanPath == "" {
		return "", errors.New(ErrEmptyRepositoryPath)
	}

	// Нормализуем хост только для публичных Git-платформ.
	// Приватные инсталляции (gitlab.company.com, git.internal) не изменяем.
	switch host {
	case "github.com", "www.github.com":
		host = GitHubHost
	case "gitlab.com", "www.gitlab.com":
		host = GitLabHost
	case "bitbucket.org", "www.bitbucket.org":
		host = BitbucketHost
	default:
		// Приватные инсталляции сохраняем как есть.
	}

	return fmt.Sprintf("%s%s/%s", "https://", host, cleanPath), nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1) //nolint:revive // Exit with error
}
