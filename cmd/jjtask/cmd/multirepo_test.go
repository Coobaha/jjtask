package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MultiRepoTestEnv provides a multi-repo test environment
type MultiRepoTestEnv struct {
	t       *testing.T
	rootDir string
	repos   map[string]*TestRepo
}

// SetupMultiRepo creates a multi-repo test environment with .jj-workspaces.yaml
func SetupMultiRepo(t *testing.T) *MultiRepoTestEnv {
	t.Helper()

	rootDir, err := os.MkdirTemp("", "jjtask-multi-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	env := &MultiRepoTestEnv{
		t:       t,
		rootDir: rootDir,
		repos:   make(map[string]*TestRepo),
	}

	t.Cleanup(func() {
		env.autoSnapshot(t)
		_ = os.RemoveAll(rootDir)
	})

	// Create .jj-workspaces.yaml
	workspacesYAML := `repos:
  - path: frontend
    name: frontend
  - path: backend
    name: backend
  - path: .
    name: root
`
	if err := os.WriteFile(filepath.Join(rootDir, ".jj-workspaces.yaml"), []byte(workspacesYAML), 0o644); err != nil {
		t.Fatalf("failed to write workspaces config: %v", err)
	}

	// Create repos
	for _, name := range []string{"frontend", "backend", "."} {
		repoPath := filepath.Join(rootDir, name)
		if name != "." {
			if err := os.MkdirAll(repoPath, 0o755); err != nil {
				t.Fatalf("failed to create repo dir: %v", err)
			}
		}

		repo := env.createRepoAt(repoPath)
		if name == "." {
			env.repos["root"] = repo
		} else {
			env.repos[name] = repo
		}
	}

	return env
}

func (env *MultiRepoTestEnv) createRepoAt(dir string) *TestRepo {
	env.t.Helper()

	repo := &TestRepo{
		t:          env.t,
		dir:        dir,
		log:        &bytes.Buffer{},
		cmdCounter: 0,
	}

	repo.baseEnv = makeBaseEnv(repo.findProjectRoot(), env.rootDir)

	repo.runSilent("jj", "git", "init", "--colocate")
	return repo
}

// RunInRoot runs a command from the root directory
func (env *MultiRepoTestEnv) RunInRoot(name string, args ...string) string {
	env.t.Helper()
	return env.repos["root"].Run(name, args...)
}

// RunIn runs a command in a specific repo
func (env *MultiRepoTestEnv) RunIn(repoName, cmdName string, args ...string) string {
	env.t.Helper()
	repo, ok := env.repos[repoName]
	if !ok {
		env.t.Fatalf("unknown repo: %s", repoName)
	}
	return repo.Run(cmdName, args...)
}

// autoSnapshot combines logs from all repos and saves snapshot
func (env *MultiRepoTestEnv) autoSnapshot(t *testing.T) {
	var combined bytes.Buffer
	for _, name := range []string{"root", "frontend", "backend"} {
		repo := env.repos[name]
		if repo != nil && repo.log.Len() > 0 {
			combined.WriteString("=== " + name + " ===\n")
			trace := repo.normalizeTrace(repo.log.String())
			// Also normalize the multi-repo root dir
			trace = strings.ReplaceAll(trace, env.rootDir, "$ROOT")
			combined.WriteString(trace)
			combined.WriteString("\n")
		}
	}

	snapshotName := testNameToSnakeCase(t.Name())
	snapshotDir := filepath.Join(env.repos["root"].findProjectRoot(), "test", "snapshots_go")
	snapshotFile := filepath.Join(snapshotDir, snapshotName+".txt")
	trace := combined.String()

	if os.Getenv("SNAPSHOT_UPDATE") != "" {
		_ = os.MkdirAll(snapshotDir, 0o755)
		if err := os.WriteFile(snapshotFile, []byte(trace), 0o644); err != nil {
			t.Fatalf("failed to write snapshot: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatalf("snapshot not found: %s\nRun with SNAPSHOT_UPDATE=1 to create\n\nActual output:\n%s", snapshotFile, trace)
	}

	if string(expected) != trace {
		t.Errorf("snapshot mismatch: %s\n\nExpected:\n%s\n\nActual:\n%s", snapshotName, expected, trace)
	}
}

func TestMultiRepoFind(t *testing.T) {
	t.Parallel()
	env := SetupMultiRepo(t)

	env.RunIn("frontend", "jjtask", "create", "Frontend task")
	env.RunIn("backend", "jjtask", "create", "Backend task")

	output := env.RunInRoot("jjtask", "find")

	if !strings.Contains(output, "Frontend task") {
		t.Error("frontend task not found in output")
	}
	if !strings.Contains(output, "Backend task") {
		t.Error("backend task not found in output")
	}
	// Should show repo grouping
	if !strings.Contains(output, "frontend") && !strings.Contains(output, "backend") {
		t.Error("expected repo names in output")
	}
}

func TestMultiRepoAllLog(t *testing.T) {
	t.Parallel()
	env := SetupMultiRepo(t)

	env.repos["frontend"].WriteFile("file.txt", "test")
	env.RunIn("frontend", "jj", "describe", "-m", "Frontend commit")

	env.repos["backend"].WriteFile("file.txt", "test")
	env.RunIn("backend", "jj", "describe", "-m", "Backend commit")

	output := env.RunInRoot("jjtask", "all", "log", "-r", "@")

	if !strings.Contains(output, "Frontend commit") {
		t.Error("frontend commit not found in all log")
	}
	if !strings.Contains(output, "Backend commit") {
		t.Error("backend commit not found in all log")
	}
}

func TestMultiRepoWorkspaceHint(t *testing.T) {
	t.Parallel()
	env := SetupMultiRepo(t)

	env.RunIn("frontend", "jjtask", "create", "Frontend task")

	// Create subdirectory and run from there
	subdir := filepath.Join(env.repos["frontend"].dir, "src")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Run jjtask find from subdirectory
	repo := env.repos["frontend"]
	origDir := repo.dir
	repo.dir = subdir
	output := repo.Run("jjtask", "find")
	repo.dir = origDir

	if !strings.Contains(output, "Frontend task") {
		t.Error("task not found when running from subdirectory")
	}
}

func TestMultiRepoConfigMigration(t *testing.T) {
	t.Parallel()
	env := SetupMultiRepo(t)

	yamlPath := filepath.Join(env.rootDir, ".jj-workspaces.yaml")
	tomlPath := filepath.Join(env.rootDir, ".jjtask.toml")

	// Verify YAML exists before migration
	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("expected .jj-workspaces.yaml to exist: %v", err)
	}

	// Run any jjtask command to trigger migration
	env.RunInRoot("jjtask", "find")

	// Verify TOML was created
	if _, err := os.Stat(tomlPath); err != nil {
		t.Fatalf("expected .jjtask.toml to be created: %v", err)
	}

	// Verify YAML was removed
	if _, err := os.Stat(yamlPath); err == nil {
		t.Error("expected .jj-workspaces.yaml to be removed after migration")
	}

	// Verify TOML content
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read migrated config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[workspaces]") {
		t.Error("migrated config missing [workspaces] section")
	}
	if !strings.Contains(content, "frontend") || !strings.Contains(content, "backend") {
		t.Error("migrated config missing repo entries")
	}
}

func TestMultiRepoComplex(t *testing.T) {
	t.Parallel()
	env := SetupMultiRepo(t)

	// Create tasks in root
	env.RunIn("root", "jjtask", "create", "ROOT: CI/CD pipeline")
	env.RunIn("root", "jjtask", "create", "--draft", "@", "ROOT: Terraform modules")
	env.RunIn("root", "jjtask", "create", "ROOT: Integration tests")

	// Create tasks in frontend
	env.RunIn("frontend", "jjtask", "create", "FE: Auth login page")
	env.RunIn("frontend", "jjtask", "create", "--draft", "@", "FE: Dark mode toggle")
	env.RunIn("frontend", "jjtask", "create", "FE: Error boundaries")

	// Create tasks in backend
	env.RunIn("backend", "jjtask", "create", "BE: User API endpoints")
	env.RunIn("backend", "jjtask", "create", "--draft", "@", "BE: GraphQL schema")
	env.RunIn("backend", "jjtask", "create", "BE: Background jobs")

	output := env.RunInRoot("jjtask", "find", "--status", "all")

	// Verify all tasks exist
	expected := []string{
		"ROOT: CI/CD pipeline",
		"ROOT: Integration tests",
		"FE: Auth login page",
		"FE: Error boundaries",
		"BE: User API endpoints",
		"BE: Background jobs",
	}
	for _, task := range expected {
		if !strings.Contains(output, task) {
			t.Errorf("expected task %q not found in output", task)
		}
	}

	// Verify draft tasks exist when showing all
	drafts := []string{
		"ROOT: Terraform modules",
		"FE: Dark mode toggle",
		"BE: GraphQL schema",
	}
	for _, task := range drafts {
		if !strings.Contains(output, task) {
			t.Errorf("expected draft task %q not found in output", task)
		}
	}
}
