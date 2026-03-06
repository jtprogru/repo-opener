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
		require.Error(t, err)
		assert.ErrorContains(t, err, `remote "nonexistent" not found`)
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

	// Test case: invalid SSH URL.
	invalidSSHURL := "git@github.com:user"
	_, err = parseRemoteURL(invalidSSHURL)
	require.Error(t, err, "Expected error for invalid SSH URL")

	// Test case: unsupported URL format.
	invalidURL := "ftp://example.com/repo.git"
	_, err = parseRemoteURL(invalidURL)
	require.Error(t, err, "Expected error for unsupported URL format")
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
	require.Error(t, err, "Expected error for empty repository path")

	// Test case: ssh scheme with invalid user.
	u, _ = url.Parse("ssh://user@github.com/jtprogru/repo-opener.git") //nolint:errcheck // Ignore error from url.Parse
	_, err = parseStructuredURL(u)
	require.Error(t, err, "Expected error for unsupported SSH username")

	// Test case: unsupported scheme.
	u, _ = url.Parse("ftp://example.com/repo.git") //nolint:errcheck // Ignore error from url.Parse
	_, err = parseStructuredURL(u)
	require.Error(t, err, "Expected error for unsupported scheme")
}

func TestParseSSHURL(t *testing.T) {
	t.Parallel()

	// Test case: valid SSH URL.
	sshURL := "git@github.com:jtprogru/repo-opener.git"
	webURL, err := parseSSHURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: invalid SSH URL.
	invalidSSHURL := "git@github.com:user"
	_, err = parseSSHURL(invalidSSHURL)
	require.Error(t, err, "Expected error for invalid SSH URL")

	// Test case: empty path.
	emptyPathURL := "git@github.com:/"
	_, err = parseSSHURL(emptyPathURL)
	assert.ErrorContains(t, err, ErrEmptyRepositoryPath)
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
