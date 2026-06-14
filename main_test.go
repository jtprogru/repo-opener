package main

import (
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

	// Test case: ssh scheme with invalid user.
	u, _ = url.Parse("ssh://user@github.com/jtprogru/repo-opener.git") //nolint:errcheck // Ignore error from url.Parse
	_, err = parseStructuredURL(u)
	require.ErrorIs(t, err, ErrUnsupportedSSHUser)

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
