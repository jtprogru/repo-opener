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
	GitHubHost    = "github.com"
	GitLabHost    = "gitlab.com"
	BitbucketHost = "bitbucket.org"

	// minPathSegments — минимальное число сегментов пути (owner/repo).
	minPathSegments = 2
	// scpURLParts — число частей при разборе scp-URL по первому двоеточию.
	scpURLParts = 2
)

// Sentinel-ошибки для программной проверки через errors.Is.
var (
	ErrRemoteNotFound        = errors.New("remote not found")
	ErrEmptyRepositoryPath   = errors.New("empty repository path")
	ErrInvalidRepositoryPath = errors.New("invalid repository path")
	ErrUnsupportedURLFormat  = errors.New("unsupported URL format")
	ErrInvalidSSHURL         = errors.New("invalid SSH URL format")
	ErrUnsupportedScheme     = errors.New("unsupported scheme")
	ErrUnsupportedSSHUser    = errors.New("unsupported SSH username")
	ErrInvalidResultURL      = errors.New("invalid resulting URL")
)

var (
	Version     = "dev"     //nolint:gochecknoglobals // This is normal
	Commit      = "none"    //nolint:gochecknoglobals // This is normal
	Date        = "unknown" //nolint:gochecknoglobals // This is normal
	BuiltBy     = "unknown" //nolint:gochecknoglobals // This is normal
	versionFlag bool        //nolint:gochecknoglobals // This is normal
	remoteName  string      //nolint:gochecknoglobals // This is normal
	openFlag    bool        //nolint:gochecknoglobals // This is normal
)

func main() {
	flag.BoolVar(&versionFlag, "version", false, "Print version information and exit")
	flag.BoolVar(&openFlag, "open", false, "Open the remote URL in the default browser")
	flag.BoolVar(&openFlag, "o", false, "Open the remote URL in the default browser (shorthand)")
	flag.StringVar(&remoteName, "remote", "origin", "Remote name to use (default: origin)")

	flag.Parse()

	if versionFlag {
		fmt.Println("Version:", Version)
		fmt.Printf("Build info: commit %s, built at %s, by %s\n", Commit, Date, BuiltBy)
		return
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		exitWithError(fmt.Errorf("not a git repository: %w", err))
	}

	remoteURL, err := getRemoteURL(repo, remoteName)
	if err != nil {
		exitWithError(err)
	}

	webURL, err := parseRemoteURL(remoteURL)
	if err != nil {
		exitWithError(err)
	}

	fmt.Println(webURL)

	if openFlag {
		if err := browser.OpenURL(webURL); err != nil {
			exitWithError(fmt.Errorf("failed to open browser: %w", err))
		}
	}
}

func getRemoteURL(repo *git.Repository, name string) (string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		cfg := remote.Config()
		if cfg.Name == name && len(cfg.URLs) > 0 {
			return cfg.URLs[0], nil
		}
	}

	return "", fmt.Errorf("%w: %q", ErrRemoteNotFound, name)
}

func parseRemoteURL(remoteURL string) (string, error) {
	// scp-подобный формат [user@]host:path разбираем до url.Parse, потому что
	// "host:path" парсер ошибочно принимает за схему ("host").
	if isSCPLikeURL(remoteURL) {
		return parseSCPURL(remoteURL)
	}

	// Стандартный URL со схемой (http(s), ssh, git).
	if u, err := url.Parse(remoteURL); err == nil && u.Scheme != "" {
		return parseStructuredURL(u)
	}

	return "", fmt.Errorf("%w: %s", ErrUnsupportedURLFormat, remoteURL)
}

// isSCPLikeURL сообщает, что строка имеет scp-подобный вид [user@]host:path:
// содержит двоеточие раньше первого слэша и не содержит "://".
func isSCPLikeURL(remoteURL string) bool {
	if strings.Contains(remoteURL, "://") {
		return false
	}

	colon := strings.IndexByte(remoteURL, ':')
	if colon < 0 {
		return false
	}

	slash := strings.IndexByte(remoteURL, '/')

	return slash < 0 || colon < slash
}

func parseStructuredURL(u *url.URL) (string, error) {
	var host, path string

	switch u.Scheme {
	case "http", "https":
		host = u.Host
		path = u.Path
	case "ssh", "git":
		// Для ssh/git порт относится к транспорту, а не к веб-интерфейсу,
		// поэтому отбрасываем его через Hostname().
		host = u.Hostname()
		path = u.Path
		if u.Scheme == "ssh" && u.User != nil && u.User.Username() != "git" {
			return "", fmt.Errorf("%w: %s", ErrUnsupportedSSHUser, u.User.Username())
		}
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedScheme, u.Scheme)
	}

	if path == "" {
		return "", ErrEmptyRepositoryPath
	}

	return buildWebURL(host, path)
}

func parseSCPURL(remoteURL string) (string, error) {
	parts := strings.SplitN(remoteURL, ":", scpURLParts)
	if len(parts) != scpURLParts {
		return "", ErrInvalidSSHURL
	}

	// Отбрасываем необязательный [user@] — для веб-URL имя пользователя не нужно.
	host := parts[0]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}

	if host == "" {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedURLFormat, remoteURL)
	}

	return buildWebURL(host, strings.Trim(parts[1], "/"))
}

func buildWebURL(hostOrigin, path string) (string, error) {
	// Создаем копию hostOrigin, чтобы не изменять оригинальную строку.
	host := hostOrigin

	// Удаляем .git в конце пути и ведущие/конечные слэши.
	cleanPath := strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	if cleanPath == "" {
		return "", ErrEmptyRepositoryPath
	}

	// Путь к репозиторию всегда состоит как минимум из owner/repo.
	if len(strings.Split(cleanPath, "/")) < minPathSegments {
		return "", fmt.Errorf("%w: %s", ErrInvalidRepositoryPath, cleanPath)
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

	webURL := fmt.Sprintf("%s%s/%s", "https://", host, cleanPath)

	// Defense-in-depth: убеждаемся, что собранная строка — корректный https-URL
	// с непустым хостом, прежде чем выводить её или открывать в браузере.
	parsed, err := url.Parse(webURL)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidResultURL, err)
	}

	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return "", fmt.Errorf("%w: %s", ErrInvalidResultURL, webURL)
	}

	return webURL, nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1) //nolint:revive // Exit with error
}
