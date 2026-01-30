package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jjtask/internal/config"
)

// Repo is an alias for config.Repo for backwards compatibility
type Repo = config.Repo

// Config is an alias for backwards compatibility
type Config = config.Config

// FindConfig delegates to config package
func FindConfig() (string, error) {
	path, _, err := config.FindConfig()
	return path, err
}

// Load delegates to config package
func Load() (cfg *Config, root string, err error) {
	return config.Load()
}

// IsMultiRepo delegates to config package
func IsMultiRepo() bool {
	return config.IsMultiRepo()
}

// GetRepos delegates to config package
func GetRepos() (repos []Repo, root string, err error) {
	return config.GetRepos()
}

// ResolveRepoPath resolves a repo path relative to workspace root
func ResolveRepoPath(repo Repo, workspaceRoot string) string {
	if repo.Path == "." {
		return workspaceRoot
	}
	if filepath.IsAbs(repo.Path) {
		return repo.Path
	}
	return filepath.Join(workspaceRoot, repo.Path)
}

// DisplayName returns the display name for a repo
func DisplayName(repo Repo) string {
	if repo.Name != "" {
		return repo.Name
	}
	if repo.Path == "." {
		return "workspace"
	}
	return repo.Path
}

// RelativePath computes relative path from cwd to target
func RelativePath(target string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return target
	}
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return target
	}
	if rel == "." {
		return "."
	}
	if !strings.HasPrefix(rel, "..") {
		return "./" + rel
	}
	return rel
}

// ContextHint returns context hint for multi-repo or subdirectory usage
func ContextHint() string {
	cfg, workspaceRoot, err := config.Load()
	if err != nil || cfg == nil {
		return ""
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Resolve symlinks for comparison
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		realCwd = cwd
	}
	realRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		realRoot = workspaceRoot
	}

	repos := cfg.Workspaces.Repos
	isMulti := len(repos) > 1
	inSubdir := realCwd != realRoot

	if !isMulti && !inSubdir {
		return ""
	}

	// Find which repo we're in
	var currentRepo string
	for _, repo := range repos {
		repoPath := ResolveRepoPath(repo, workspaceRoot)
		realRepo, err := filepath.EvalSymlinks(repoPath)
		if err != nil {
			realRepo = repoPath
		}
		if strings.HasPrefix(realCwd, realRepo) {
			currentRepo = DisplayName(repo)
			break
		}
	}

	cwdRel := "."
	if realCwd != realRoot {
		rel, err := filepath.Rel(realRoot, realCwd)
		if err == nil {
			cwdRel = rel
		}
	}

	if cwdRel == "." {
		return fmt.Sprintf("cwd: . | repo: %s", currentRepo)
	}

	// Compute relative path to workspace root
	depth := strings.Count(cwdRel, string(filepath.Separator))
	rootRel := ".."
	for range depth {
		rootRel = "../" + rootRel
	}

	return fmt.Sprintf("cwd: %s | repo: %s | workspace: %s", cwdRel, currentRepo, rootRel)
}
