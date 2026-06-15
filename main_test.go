package main

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRemoteURL(t *testing.T) {
	t.Run("returns origin remote", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{"https://example.com/org/repo.git"},
		})
		require.NoError(t, err)

		remoteURL, err := getRemoteURL(repo, "origin")
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/org/repo.git", remoteURL)
	})

	t.Run("returns upstream remote", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "upstream",
			URLs: []string{"https://github.com/org/repo.git"},
		})
		require.NoError(t, err)

		remoteURL, err := getRemoteURL(repo, "upstream")
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/org/repo.git", remoteURL)
	})

	t.Run("remote not found", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{"https://example.com/org/repo.git"},
		})
		require.NoError(t, err)

		_, err = getRemoteURL(repo, "nonexistent")
		require.ErrorIs(t, err, ErrRemoteNotFound)
		assert.ErrorContains(t, err, "nonexistent")
	})
}

func TestParseRemoteURL(t *testing.T) {
	t.Parallel()

	// Test case: valid HTTP URL (GitHub).
	httpURL := "https://github.com/jtprogru/repo-opener.git"
	webURL, err := parseRemoteURL(httpURL)
	require.NoError(t, err, "Expected no error for valid HTTP URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: valid HTTP URL (GitLab).
	httpURL = "https://gitlab.com/org/repo.git"
	webURL, err = parseRemoteURL(httpURL)
	require.NoError(t, err, "Expected no error for valid HTTP URL")
	assert.Equal(t, "https://gitlab.com/org/repo", webURL)

	// Test case: valid HTTP URL (Bitbucket).
	httpURL = "https://bitbucket.org/org/repo.git"
	webURL, err = parseRemoteURL(httpURL)
	require.NoError(t, err, "Expected no error for valid HTTP URL")
	assert.Equal(t, "https://bitbucket.org/org/repo", webURL)

	// Test case: valid SSH URL (GitHub).
	sshURL := "git@github.com:jtprogru/repo-opener.git"
	webURL, err = parseRemoteURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: valid SSH URL (GitLab).
	sshURL = "git@gitlab.com:org/repo.git"
	webURL, err = parseRemoteURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://gitlab.com/org/repo", webURL)

	// Test case: valid SSH URL (Bitbucket).
	sshURL = "git@bitbucket.org:org/repo.git"
	webURL, err = parseRemoteURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://bitbucket.org/org/repo", webURL)

	// Test case: ssh:// scheme with SSH port is stripped.
	webURL, err = parseRemoteURL("ssh://git@github.com:22/org/repo.git")
	require.NoError(t, err, "Expected no error for ssh:// URL with port")
	assert.Equal(t, "https://github.com/org/repo", webURL)

	// Test case: git:// protocol is supported.
	webURL, err = parseRemoteURL("git://github.com/org/repo.git")
	require.NoError(t, err, "Expected no error for git:// URL")
	assert.Equal(t, "https://github.com/org/repo", webURL)

	// Test case: scp-like URL without git@ user.
	webURL, err = parseRemoteURL("github.com:org/repo.git")
	require.NoError(t, err, "Expected no error for scp-like URL without user")
	assert.Equal(t, "https://github.com/org/repo", webURL)

	// Test case: scp-like URL with a custom user.
	webURL, err = parseRemoteURL("org-ssh@github.com:org/repo.git")
	require.NoError(t, err, "Expected no error for scp-like URL with custom user")
	assert.Equal(t, "https://github.com/org/repo", webURL)

	// Test case: single-segment path is rejected (http).
	_, err = parseRemoteURL("https://github.com/onlyuser")
	require.ErrorIs(t, err, ErrInvalidRepositoryPath)

	// Test case: invalid SSH URL.
	invalidSSHURL := "git@github.com:user"
	_, err = parseRemoteURL(invalidSSHURL)
	require.Error(t, err, "Expected error for invalid SSH URL")

	// Test case: unsupported URL format.
	invalidURL := "ftp://example.com/repo.git"
	_, err = parseRemoteURL(invalidURL)
	require.ErrorIs(t, err, ErrUnsupportedScheme)
}

func TestParseStructuredURL(t *testing.T) {
	t.Parallel()

	// Test case: valid HTTP URL.
	u, _ := url.Parse("https://github.com/jtprogru/repo-opener.git") //nolint:errcheck // Ignore error from url.Parse
	webURL, err := parseStructuredURL(u)
	require.NoError(t, err, "Expected no error for valid structured URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: missing path.
	u, _ = url.Parse("https://github.com") //nolint:errcheck // Ignore error from url.Parse
	_, err = parseStructuredURL(u)
	require.ErrorIs(t, err, ErrEmptyRepositoryPath)

	// Test case: ssh scheme with port is stripped from the web host.
	u, _ = url.Parse("ssh://git@github.com:22/jtprogru/repo-opener.git") //nolint:errcheck // Ignore error from url.Parse
	webURL, err = parseStructuredURL(u)
	require.NoError(t, err, "Expected no error for ssh URL with port")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: git scheme is supported.
	u, _ = url.Parse("git://github.com/jtprogru/repo-opener.git") //nolint:errcheck // Ignore error from url.Parse
	webURL, err = parseStructuredURL(u)
	require.NoError(t, err, "Expected no error for git scheme")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: ssh scheme with a non-git user (self-hosted) is accepted;
	// the username is irrelevant to the resulting web URL.
	u, _ = url.Parse("ssh://gitlab@git.company.com/group/repo.git") //nolint:errcheck // Ignore error from url.Parse
	webURL, err = parseStructuredURL(u)
	require.NoError(t, err, "Expected no error for non-git ssh user")
	assert.Equal(t, "https://git.company.com/group/repo", webURL)

	// Test case: ssh scheme without a user (anonymous) is accepted.
	u, _ = url.Parse("ssh://github.com/org/repo.git") //nolint:errcheck // Ignore error from url.Parse
	webURL, err = parseStructuredURL(u)
	require.NoError(t, err, "Expected no error for anonymous ssh")
	assert.Equal(t, "https://github.com/org/repo", webURL)

	// Test case: unsupported scheme.
	u, _ = url.Parse("ftp://example.com/repo.git") //nolint:errcheck // Ignore error from url.Parse
	_, err = parseStructuredURL(u)
	require.ErrorIs(t, err, ErrUnsupportedScheme)
}

func TestParseSCPURL(t *testing.T) {
	t.Parallel()

	// Test case: valid SSH URL.
	sshURL := "git@github.com:jtprogru/repo-opener.git"
	webURL, err := parseSCPURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: custom user is discarded.
	webURL, err = parseSCPURL("org-ssh@gitlab.com:org/repo.git")
	require.NoError(t, err, "Expected no error for custom user")
	assert.Equal(t, "https://gitlab.com/org/repo", webURL)

	// Test case: single-segment path is rejected.
	_, err = parseSCPURL("git@github.com:user")
	require.ErrorIs(t, err, ErrInvalidRepositoryPath)

	// Test case: empty path.
	_, err = parseSCPURL("git@github.com:/")
	require.ErrorIs(t, err, ErrEmptyRepositoryPath)

	// Test case: missing host.
	_, err = parseSCPURL(":org/repo.git")
	require.ErrorIs(t, err, ErrUnsupportedURLFormat)

	// Test case: no colon at all (defensive; unreachable via parseRemoteURL).
	_, err = parseSCPURL("github.com")
	require.ErrorIs(t, err, ErrInvalidSSHURL)
}

func TestBuildWebURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		path    string
		want    string
		wantErr bool
	}{
		// Публичные платформы.
		{
			name: "github.com normalized",
			host: "github.com",
			path: "org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "www.github.com normalized",
			host: "www.github.com",
			path: "org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "gitlab.com normalized",
			host: "gitlab.com",
			path: "org/repo.git",
			want: "https://gitlab.com/org/repo",
		},
		{
			name: "www.gitlab.com normalized",
			host: "www.gitlab.com",
			path: "org/repo.git",
			want: "https://gitlab.com/org/repo",
		},
		{
			name: "bitbucket.org normalized",
			host: "bitbucket.org",
			path: "org/repo.git",
			want: "https://bitbucket.org/org/repo",
		},
		{
			name: "www.bitbucket.org normalized",
			host: "www.bitbucket.org",
			path: "org/repo.git",
			want: "https://bitbucket.org/org/repo",
		},
		// GitLab-подгруппы и глубокая вложенность путей сохраняются.
		{
			name: "gitlab subgroup preserved",
			host: "gitlab.com",
			path: "group/subgroup/repo.git",
			want: "https://gitlab.com/group/subgroup/repo",
		},
		{
			name: "deeply nested subgroups preserved",
			host: "gitlab.company.com",
			path: "team/group/subgroup/repo.git",
			want: "https://gitlab.company.com/team/group/subgroup/repo",
		},
		// Приватные инсталляции (не изменяются).
		{
			name: "private gitlab preserved",
			host: "gitlab.internal.company",
			path: "org/repo.git",
			want: "https://gitlab.internal.company/org/repo",
		},
		{
			name: "private git host preserved",
			host: "git.jtprog.ru",
			path: "org/repo.git",
			want: "https://git.jtprog.ru/org/repo",
		},
		{
			name: "private github enterprise preserved",
			host: "github.internal.company",
			path: "org/repo.git",
			want: "https://github.internal.company/org/repo",
		},
		{
			name: "gitea instance preserved",
			host: "gitea.example.com",
			path: "org/repo",
			want: "https://gitea.example.com/org/repo",
		},
		{
			name: "forgejo instance preserved",
			host: "forgejo.example.com",
			path: "org/repo",
			want: "https://forgejo.example.com/org/repo",
		},
		{
			name: "gitlab under url subpath preserved",
			host: "example.com",
			path: "gitlab/group/repo.git",
			want: "https://example.com/gitlab/group/repo",
		},
		{
			name: "host with web port preserved",
			host: "gitlab.company.com:8443",
			path: "group/repo.git",
			want: "https://gitlab.company.com:8443/group/repo",
		},
		// Ошибки.
		{
			name:    "empty path",
			host:    "github.com",
			path:    "",
			wantErr: true,
		},
		{
			name:    "only .git in path",
			host:    "github.com",
			path:    ".git",
			wantErr: true,
		},
		{
			name:    "single segment path",
			host:    "github.com",
			path:    "onlyuser",
			wantErr: true,
		},
		{
			name:    "invalid host produces invalid url",
			host:    "bad host",
			path:    "org/repo",
			wantErr: true,
		},
		{
			name:    "empty host produces invalid url",
			host:    "",
			path:    "org/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildWebURL(tt.host, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSCPLikeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		remoteURL string
		want      bool
	}{
		{name: "git@ scp form", remoteURL: "git@github.com:org/repo.git", want: true},
		{name: "scp without user", remoteURL: "github.com:org/repo.git", want: true},
		{name: "custom user scp", remoteURL: "org-ssh@github.com:org/repo.git", want: true},
		{name: "https url", remoteURL: "https://github.com/org/repo.git", want: false},
		{name: "ssh url with scheme", remoteURL: "ssh://git@github.com/org/repo.git", want: false},
		{name: "git url with scheme", remoteURL: "git://github.com/org/repo.git", want: false},
		{name: "no colon", remoteURL: "github.com/org/repo", want: false},
		{name: "colon after slash", remoteURL: "/path/to:repo", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isSCPLikeURL(tt.remoteURL))
		})
	}
}

func TestParseRemoteURLSelfHosted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		remote  string
		want    string
		wantErr bool
	}{
		// HTTPS self-hosted: веб-порт сохраняется.
		{
			name:   "https self-hosted gitlab with web port",
			remote: "https://gitlab.company.com:8443/group/repo.git",
			want:   "https://gitlab.company.com:8443/group/repo",
		},
		// HTTPS со встроенными кредами — креды отбрасываются.
		{ //nolint:gosec // Test data with fake credentials, not a real secret.
			name:   "https with embedded credentials",
			remote: "https://user:token@git.internal.io/team/project.git",
			want:   "https://git.internal.io/team/project",
		},
		// HTTP (insecure) self-hosted — в вебе всё равно https.
		{
			name:   "http self-hosted upgraded to https",
			remote: "http://git.internal/team/repo.git",
			want:   "https://git.internal/team/repo",
		},
		// GitLab-подгруппы (глубокая вложенность).
		{
			name:   "https gitlab nested subgroups",
			remote: "https://gitlab.com/group/sub/repo.git",
			want:   "https://gitlab.com/group/sub/repo",
		},
		// GitLab под subpath (relative URL root).
		{
			name:   "https self-hosted gitlab under subpath",
			remote: "https://example.com/gitlab/group/repo.git",
			want:   "https://example.com/gitlab/group/repo",
		},
		// ssh-схема: нестандартный порт + не-git пользователь (self-hosted).
		{
			name:   "ssh custom port and non-git user",
			remote: "ssh://gitlab@git.company.com:2222/group/repo.git",
			want:   "https://git.company.com/group/repo",
		},
		// ssh-схема без пользователя (анонимный).
		{
			name:   "ssh anonymous self-hosted",
			remote: "ssh://code.example.org/org/repo.git",
			want:   "https://code.example.org/org/repo",
		},
		// git-протокол с нестандартным портом.
		{
			name:   "git protocol with port",
			remote: "git://git.example.com:9418/org/repo.git",
			want:   "https://git.example.com/org/repo",
		},
		// scp self-hosted Gitea с кастомным пользователем.
		{
			name:   "scp gitea custom user",
			remote: "forgejo@code.example.org:org/repo.git",
			want:   "https://code.example.org/org/repo",
		},
		// scp self-hosted без пользователя.
		{
			name:   "scp self-hosted without user",
			remote: "git.internal.io:team/project.git",
			want:   "https://git.internal.io/team/project",
		},
		// scp с глубокой вложенностью подгрупп.
		{
			name:   "scp gitlab nested subgroups",
			remote: "git@gitlab.company.com:team/group/sub/repo.git",
			want:   "https://gitlab.company.com/team/group/sub/repo",
		},
		// scp с завершающим слэшем и .git.
		{
			name:   "scp trailing slash and dot-git",
			remote: "git@gitea.example.com:org/repo.git/",
			want:   "https://gitea.example.com/org/repo",
		},
		// Ошибки: одиночный сегмент пути на self-hosted.
		{
			name:    "self-hosted single-segment path rejected",
			remote:  "https://git.internal/onlyrepo",
			wantErr: true,
		},
		// Ошибки: неподдерживаемая схема.
		{
			name:    "ftp scheme rejected",
			remote:  "ftp://git.internal/org/repo.git",
			wantErr: true,
		},
		// Ошибки: локальный путь без схемы и scp-двоеточия.
		{
			name:    "local path rejected",
			remote:  "/home/user/repo",
			wantErr: true,
		},
		// Ошибки: произвольная строка.
		{
			name:    "plain string rejected",
			remote:  "justastring",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRemoteURL(tt.remote)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRun(t *testing.T) {
	noopOpen := func(string) error { return nil }

	createOriginRepo := func(t *testing.T, remoteURL string) {
		t.Helper()

		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteURL},
		})
		require.NoError(t, err)
	}

	t.Run("prints web url for origin", func(t *testing.T) {
		createOriginRepo(t, "git@github.com:org/repo.git")

		var buf bytes.Buffer
		require.NoError(t, run(nil, &buf, noopOpen))
		assert.Equal(t, "https://github.com/org/repo\n", buf.String())
	})

	t.Run("open flag invokes opener with self-hosted url", func(t *testing.T) {
		createOriginRepo(t, "https://gitlab.company.com:8443/group/repo.git")

		var (
			opened string
			buf    bytes.Buffer
		)

		err := run([]string{"-o"}, &buf, func(u string) error {
			opened = u

			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, "https://gitlab.company.com:8443/group/repo", opened)
	})

	t.Run("custom remote flag", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "upstream",
			URLs: []string{"git@gitea.example.com:team/proj.git"},
		})
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, run([]string{"--remote", "upstream"}, &buf, noopOpen))
		assert.Equal(t, "https://gitea.example.com/team/proj\n", buf.String())
	})

	t.Run("version flag", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, run([]string{"--version"}, &buf, noopOpen))
		assert.Contains(t, buf.String(), "Version:")
		assert.Contains(t, buf.String(), "Build info:")
	})

	t.Run("not a git repository", func(t *testing.T) {
		withChdir(t, t.TempDir())

		var buf bytes.Buffer
		err := run(nil, &buf, noopOpen)
		require.Error(t, err)
		assert.ErrorContains(t, err, "not a git repository")
	})

	t.Run("remote not found", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		var buf bytes.Buffer
		require.ErrorIs(t, run(nil, &buf, noopOpen), ErrRemoteNotFound)
	})

	t.Run("unsupported remote url", func(t *testing.T) {
		createOriginRepo(t, "ftp://example.com/repo.git")

		var buf bytes.Buffer
		require.ErrorIs(t, run(nil, &buf, noopOpen), ErrUnsupportedScheme)
	})

	t.Run("open failure is propagated", func(t *testing.T) {
		createOriginRepo(t, "git@github.com:org/repo.git")

		failOpen := func(string) error { return errors.New("boom") }

		var buf bytes.Buffer
		err := run([]string{"-o"}, &buf, failOpen)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to open browser")
	})

	t.Run("invalid flag", func(t *testing.T) {
		var buf bytes.Buffer
		require.Error(t, run([]string{"--nonexistent"}, &buf, noopOpen))
	})
}

func TestResolveRemoteURL(t *testing.T) {
	t.Run("falls back to direct config read when worktreeConfig extension is set", func(t *testing.T) {
		repoPath := initTempRepo(t)
		setOriginRemote(t, repoPath, "git@github.com:org/repo.git")
		enableWorktreeConfig(t, repoPath)

		// go-git отказывается открыть такой репозиторий — проверяем это явно,
		// чтобы тест ловил именно срабатывание запасного пути.
		_, openErr := git.PlainOpen(repoPath)
		require.Error(t, openErr)

		withChdir(t, repoPath)

		remoteURL, err := resolveRemoteURL(".", "origin")
		require.NoError(t, err)
		assert.Equal(t, "git@github.com:org/repo.git", remoteURL)
	})

	t.Run("fallback reports missing remote, not a missing repository", func(t *testing.T) {
		repoPath := initTempRepo(t)
		setOriginRemote(t, repoPath, "git@github.com:org/repo.git")
		enableWorktreeConfig(t, repoPath)
		withChdir(t, repoPath)

		_, err := resolveRemoteURL(".", "upstream")
		require.ErrorIs(t, err, ErrRemoteNotFound)
		assert.NotContains(t, err.Error(), "not a git repository")
	})

	t.Run("end-to-end run resolves url with worktreeConfig extension", func(t *testing.T) {
		repoPath := initTempRepo(t)
		setOriginRemote(t, repoPath, "git@github.com:org/repo.git")
		enableWorktreeConfig(t, repoPath)
		withChdir(t, repoPath)

		var buf bytes.Buffer
		require.NoError(t, run(nil, &buf, func(string) error { return nil }))
		assert.Equal(t, "https://github.com/org/repo\n", buf.String())
	})

	t.Run("non-repository still reports not a git repository", func(t *testing.T) {
		dir := t.TempDir()

		_, err := resolveRemoteURL(dir, "origin")
		require.Error(t, err)
		assert.ErrorContains(t, err, "not a git repository")
	})
}

func setOriginRemote(t *testing.T, repoPath, remoteURL string) {
	t.Helper()

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	require.NoError(t, err)
}

// enableWorktreeConfig включает расширение extensions.worktreeConfig — то же,
// что выставляют VS Code и git worktree и что заставляет go-git отказаться
// открывать репозиторий через git.PlainOpen.
func enableWorktreeConfig(t *testing.T, repoPath string) {
	t.Helper()

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	cfg, err := repo.Config()
	require.NoError(t, err)

	cfg.Raw.Section("extensions").SetOption("worktreeConfig", "true")
	require.NoError(t, repo.Storer.SetConfig(cfg))
}

func initTempRepo(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()

	_, err := git.PlainInit(tmp, false)
	require.NoError(t, err)

	return tmp
}

func withChdir(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	})
}
