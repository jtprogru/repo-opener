package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
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

	// gitDirName — имя служебного каталога git внутри репозитория.
	gitDirName = ".git"
)

// Sentinel-ошибки для программной проверки через errors.Is.
var (
	ErrRemoteNotFound        = errors.New("remote not found")
	ErrEmptyRepositoryPath   = errors.New("empty repository path")
	ErrInvalidRepositoryPath = errors.New("invalid repository path")
	ErrUnsupportedURLFormat  = errors.New("unsupported URL format")
	ErrInvalidSSHURL         = errors.New("invalid SSH URL format")
	ErrUnsupportedScheme     = errors.New("unsupported scheme")
	ErrInvalidResultURL      = errors.New("invalid resulting URL")
)

var (
	Version = "dev"     //nolint:gochecknoglobals // Set via -ldflags at build time.
	Commit  = "none"    //nolint:gochecknoglobals // Set via -ldflags at build time.
	Date    = "unknown" //nolint:gochecknoglobals // Set via -ldflags at build time.
	BuiltBy = "unknown" //nolint:gochecknoglobals // Set via -ldflags at build time.
)

func main() {
	if err := run(os.Args[1:], os.Stdout, browser.OpenURL); err != nil {
		exitWithError(err)
	}
}

// run содержит всю логику CLI и вынесен из main для тестируемости:
// out принимает обычный вывод, openURL — функцию открытия в браузере.
func run(args []string, out io.Writer, openURL func(string) error) error {
	var (
		versionFlag bool
		openFlag    bool
		remoteName  string
	)

	fs := flag.NewFlagSet("repo-opener", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&versionFlag, "version", false, "Print version information and exit")
	fs.BoolVar(&openFlag, "open", false, "Open the remote URL in the default browser")
	fs.BoolVar(&openFlag, "o", false, "Open the remote URL in the default browser (shorthand)")
	fs.StringVar(&remoteName, "remote", "origin", "Remote name to use (default: origin)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if versionFlag {
		fmt.Fprintln(out, "Version:", Version)
		fmt.Fprintf(out, "Build info: commit %s, built at %s, by %s\n", Commit, Date, BuiltBy)

		return nil
	}

	remoteURL, err := resolveRemoteURL(".", remoteName)
	if err != nil {
		return err
	}

	webURL, err := parseRemoteURL(remoteURL)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, webURL)

	if openFlag {
		if err := openURL(webURL); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
		}
	}

	return nil
}

// resolveRemoteURL возвращает URL ремоута name для репозитория в dir.
//
// Сначала используется обычный путь go-git (git.PlainOpen). Если go-git
// отказывается открывать репозиторий из-за включённых git-расширений
// (например extensions.worktreeConfig, который выставляют VS Code и
// git worktree), выполняется запасной путь: конфиг читается напрямую, минуя
// проверку расширений, поскольку для построения веб-URL достаточно прочитать
// секцию remote.
func resolveRemoteURL(dir, name string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err == nil {
		return getRemoteURL(repo, name)
	}

	if errors.Is(err, git.ErrUnsupportedExtensionRepositoryFormatVersion) ||
		errors.Is(err, git.ErrUnknownExtension) {
		return remoteURLFromConfig(dir, name)
	}

	return "", fmt.Errorf("not a git repository: %w", err)
}

// remoteURLFromConfig возвращает первый URL ремоута name, читая конфигурацию
// через слой хранилища go-git напрямую (filesystem.Storage.Config), минуя
// git.Open и его проверку расширений. Файловый ввод-вывод выполняет go-git
// поверх go-billy — собственных обращений к os мы не делаем.
func remoteURLFromConfig(dir, name string) (string, error) {
	dotGit := osfs.New(filepath.Join(dir, gitDirName))
	if _, err := dotGit.Stat(""); err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	storer := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	cfg, err := storer.Config()
	if err != nil {
		return "", fmt.Errorf("failed to read git config: %w", err)
	}

	remote, ok := cfg.Remotes[name]
	if !ok || len(remote.URLs) == 0 {
		return "", fmt.Errorf("%w: %q", ErrRemoteNotFound, name)
	}

	return remote.URLs[0], nil
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
		// поэтому отбрасываем его через Hostname(). Имя пользователя для
		// веб-URL не нужно, поэтому любые пользователи допустимы (self-hosted
		// инсталляции нередко используют не git-пользователя).
		host = u.Hostname()
		path = u.Path
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
