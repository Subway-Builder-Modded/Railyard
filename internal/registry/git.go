package registry

import (
	"fmt"
	"strings"

	"railyard/internal/files"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// openOrClone opens an existing repo or force-clones if missing/corrupt.
func (r *Registry) openOrClone() error {
	repo, err := git.PlainOpen(r.repoPath)
	if err != nil {
		return r.forceClone()
	}

	if _, err := repo.Head(); err != nil {
		return r.forceClone()
	}

	return nil
}

// refreshRepo fetches and resets to origin/main, or force-clones on failure.
func (r *Registry) refreshRepo() error {
	repo, err := git.PlainOpen(r.repoPath)
	if err != nil {
		return r.forceClone()
	}

	if err := r.fetchAndReset(repo); err != nil {
		return r.forceClone()
	}

	return nil
}

// getCredentials uses the system's git credential helper to resolve
// credentials for the registry repo URL. Returns nil auth if no
// credentials are found (for public repos).
//func (r *Registry) getCredentials() *githttp.BasicAuth {
//	parsed, err := url.Parse(RegistryRepoURL)
//	if err != nil {
//		return nil
//	}
//
//	input := fmt.Sprintf("protocol=%s\nhost=%s\npath=%s\n\n", parsed.Scheme, parsed.Host, strings.TrimPrefix(parsed.Path, "/"))
//
//	cmd := exec.Command("git", "credential", "fill")
//	cmd.Stdin = strings.NewReader(input)
//	var out bytes.Buffer
//	cmd.Stdout = &out
//	if err := cmd.Run(); err != nil {
//		return nil
//	}
//
//	var username, password string
//	scanner := bufio.NewScanner(&out)
//	for scanner.Scan() {
//		line := scanner.Text()
//		if k, v, ok := strings.Cut(line, "="); ok {
//			switch k {
//			case "username":
//				username = v
//			case "password":
//				password = v
//			}
//		}
//	}
//
//	if username != "" && password != "" {
//		return &githttp.BasicAuth{
//			Username: username,
//			Password: password,
//		}
//	}
//	return nil
//}

// forceClone removes any existing directory and performs a fresh clone.
func (r *Registry) forceClone() error {
	cloneErrors := make([]string, 0, len(RegistryRepoURLs))
	for _, repoURL := range RegistryRepoURLs {
		err := files.WritePathsAtomically([]files.AtomicWrite{
			files.AtomicDirectoryWrite{
				Path:  r.repoPath,
				Label: "registry clone directory",
				Callback: func(stagingPath string) error {
					cloneOpts := &git.CloneOptions{
						URL:           repoURL,
						ReferenceName: plumbing.NewBranchReferenceName("main"),
						SingleBranch:  true,
						Depth:         1,
					}
					_, cloneErr := git.PlainClone(stagingPath, false, cloneOpts)
					return cloneErr
				},
			},
		})
		if err != nil {
			cloneErrors = append(cloneErrors, fmt.Sprintf("%s: %v", repoURL, err))
			r.logger.Warn("Failed to clone registry repository candidate", "repo_url", repoURL, "error", err)
			continue
		}
		if err := r.fetchFromDisk(); err != nil {
			cloneErrors = append(cloneErrors, fmt.Sprintf("%s: failed to load data after clone: %v", repoURL, err))
			r.logger.Warn("Failed to load registry data after clone candidate", "repo_url", repoURL, "error", err)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to clone registry repository from all candidates: %s", strings.Join(cloneErrors, "; "))
}

// fetchAndReset fetches from origin and hard-resets the working tree to
// origin/main.
func (r *Registry) fetchAndReset(repo *git.Repository) error {
	fetchOpts := &git.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"+refs/heads/main:refs/remotes/origin/main",
		},
		Force: true,
	}
	err := repo.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}

	ref, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), true)
	if err != nil {
		return fmt.Errorf("failed to resolve origin/main: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = wt.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to origin/main: %w", err)
	}

	return nil
}
