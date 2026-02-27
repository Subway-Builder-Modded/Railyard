package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

func TestEnsureLocalRepoReturnsNilForHealthyRepo(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "registry")
	initLocalRepoWithCommit(t, repoPath)

	r := &Registry{repoPath: repoPath}
	require.NoError(t, r.ensureLocalRepo())
}

func initLocalRepoWithCommit(t *testing.T, repoPath string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(repoPath, 0o755))
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("test"), 0o644))

	wt, err := repo.Worktree()
	require.NoError(t, err)

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)
}
