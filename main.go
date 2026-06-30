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
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
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

	// Сегменты пути к ветке в веб-интерфейсе разных Git-платформ.
	branchPathGitHub    = "/tree/"       // GitHub, GitHub Enterprise.
	branchPathGitLab    = "/-/tree/"     // GitLab (gitlab.com и self-hosted).
	branchPathBitbucket = "/src/"        // Bitbucket Cloud.
	branchPathGitea     = "/src/branch/" // Gitea, Forgejo, Codeberg.
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
		versionFlag  bool
		openFlag     bool
		noBranchFlag bool
		remoteName   string
	)

	fs := flag.NewFlagSet("repo-opener", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&versionFlag, "version", false, "Print version information and exit")
	fs.BoolVar(&openFlag, "open", false, "Open the remote URL in the default browser")
	fs.BoolVar(&openFlag, "o", false, "Open the remote URL in the default browser (shorthand)")
	fs.BoolVar(&noBranchFlag, "no-branch", false, "Always print the repository root URL, ignoring the current branch")
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

	// На кастомной (не дефолтной) ветке добавляем к URL ссылку на эту ветку.
	// Детект веток best-effort и тихо деградирует к корневому URL при ошибках,
	// поэтому он не влияет на основную функциональность.
	if !noBranchFlag {
		if branch, ok := customBranch(".", remoteName); ok {
			webURL, err = appendBranchPath(webURL, branch)
			if err != nil {
				return err
			}
		}
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

	store := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	cfg, err := store.Config()
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

// openStorer открывает слой хранилища go-git для репозитория в dir, повторяя
// двухпутёвую логику resolveRemoteURL: обычный git.PlainOpen, а при включённых
// несовместимых расширениях (extensions.worktreeConfig и т.п.) — прямое чтение
// .git через filesystem.Storage. Возвращённый storage.Storer реализует
// storer.ReferenceStorer, поэтому чтение ссылок одинаково для обоих путей.
func openStorer(dir string) (storage.Storer, error) {
	repo, err := git.PlainOpen(dir)
	if err == nil {
		return repo.Storer, nil
	}

	if errors.Is(err, git.ErrUnsupportedExtensionRepositoryFormatVersion) ||
		errors.Is(err, git.ErrUnknownExtension) {
		dotGit := osfs.New(filepath.Join(dir, gitDirName))
		if _, statErr := dotGit.Stat(""); statErr != nil {
			return nil, fmt.Errorf("not a git repository: %w", statErr)
		}

		return filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault()), nil
	}

	return nil, fmt.Errorf("not a git repository: %w", err)
}

// customBranch сообщает имя текущей ветки и true, если она не является дефолтной
// (то есть для неё нужно собрать URL со ссылкой на ветку). Детект best-effort:
// при любой ошибке, detached HEAD или дефолтной ветке возвращается ("", false),
// и вызывающая сторона печатает корневой URL.
func customBranch(dir, remote string) (string, bool) {
	store, err := openStorer(dir)
	if err != nil {
		return "", false
	}

	cur := headBranch(store)
	if cur == "" {
		return "", false
	}

	def := defaultBranch(store, remote)
	if def != "" {
		return cur, cur != def
	}

	// origin/HEAD недоступен (частый случай локальных клонов) — считаем
	// дефолтными общепринятые main/master.
	switch cur {
	case "main", "master":
		return cur, false
	default:
		return cur, true
	}
}

// headBranch возвращает короткое имя текущей ветки, читая символьную ссылку HEAD
// напрямую (без резолва в коммит — работает и в репозитории без коммитов).
// Для detached HEAD возвращается пустая строка.
func headBranch(refs storer.ReferenceStorer) string {
	head, err := refs.Reference(plumbing.HEAD)
	if err != nil || head.Type() != plumbing.SymbolicReference {
		return ""
	}

	target := head.Target()
	if !target.IsBranch() {
		return ""
	}

	return target.Short()
}

// defaultBranch возвращает имя дефолтной ветки удалёнки remote по символьной
// ссылке refs/remotes/<remote>/HEAD. Если ссылка отсутствует или не символьная,
// возвращается пустая строка.
func defaultBranch(refs storer.ReferenceStorer, remote string) string {
	ref, err := refs.Reference(plumbing.NewRemoteHEADReferenceName(remote))
	if err != nil || ref.Type() != plumbing.SymbolicReference {
		return ""
	}

	// Target вида refs/remotes/origin/main → Short() = "origin/main".
	return strings.TrimPrefix(ref.Target().Short(), remote+"/")
}

// appendBranchPath добавляет к корневому web-URL ссылку на ветку с учётом
// платформенного формата пути (см. branchPathSegment).
func appendBranchPath(webURL, branch string) (string, error) {
	u, err := url.Parse(webURL)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidResultURL, err)
	}

	return webURL + branchPathSegment(u.Hostname()) + encodeBranch(branch), nil
}

// branchPathSegment подбирает сегмент пути к ветке по имени хоста. Публичные
// платформы определяются точно, self-hosted инсталляции — эвристически по
// подстроке в hostname. Для неизвестных хостов используется GitHub-стиль /tree/
// как наиболее распространённый (его понимает в том числе GitHub Enterprise).
func branchPathSegment(rawHost string) string {
	host := strings.ToLower(rawHost)

	switch {
	case host == "gitlab.com" || strings.Contains(host, "gitlab"):
		return branchPathGitLab
	case host == "bitbucket.org" || strings.Contains(host, "bitbucket"):
		return branchPathBitbucket
	case host == "codeberg.org" || strings.Contains(host, "gitea") || strings.Contains(host, "forgejo"):
		return branchPathGitea
	default:
		return branchPathGitHub
	}
}

// encodeBranch экранирует имя ветки для подстановки в путь URL, сохраняя слэши
// как разделители сегментов (feat/x → feat/x), но экранируя прочие спецсимволы.
func encodeBranch(branch string) string {
	segments := strings.Split(branch, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}

	return strings.Join(segments, "/")
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1) //nolint:revive // Exit with error
}
