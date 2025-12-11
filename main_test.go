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

func TestCheckGitRepo(t *testing.T) {
	t.Run("valid repo", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		err := checkGitRepo()
		require.NoError(t, err)
	})

	t.Run("not a repo", func(t *testing.T) {
		tmp := t.TempDir()
		withChdir(t, tmp)

		err := checkGitRepo()
		require.Error(t, err)
	})
}

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

		remoteURL, err := getRemoteURL()
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/org/repo.git", remoteURL)
	})

	t.Run("origin missing", func(t *testing.T) {
		repoPath := initTempRepo(t)
		withChdir(t, repoPath)

		repo, err := git.PlainOpen(repoPath)
		require.NoError(t, err)

		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "upstream",
			URLs: []string{"https://example.com/org/repo.git"},
		})
		require.NoError(t, err)

		_, err = getRemoteURL()
		require.Error(t, err)
		assert.ErrorContains(t, err, ErrOriginRemoteNotFound)
	})
}

func TestParseRemoteURL(t *testing.T) {
	t.Parallel()

	// Test case: valid HTTP URL.
	httpURL := "https://github.com/jtprogru/repo-opener.git"
	webURL, err := parseRemoteURL(httpURL)
	require.NoError(t, err, "Expected no error for valid HTTP URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

	// Test case: valid SSH URL.
	sshURL := "git@github.com:jtprogru/repo-opener.git"
	webURL, err = parseRemoteURL(sshURL)
	require.NoError(t, err, "Expected no error for valid SSH URL")
	assert.Equal(t, "https://github.com/jtprogru/repo-opener", webURL)

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
		{
			name: "github host normalized",
			host: "github.example.com",
			path: "jtprogru/repo-opener.git",
			want: "https://github.com/jtprogru/repo-opener",
		},
		{
			name: "custom host preserved",
			host: "gitlab.com",
			path: "/org/repo/",
			want: "https://gitlab.com/org/repo",
		},
		{
			name:    "empty path",
			host:    "github.com",
			path:    "",
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
