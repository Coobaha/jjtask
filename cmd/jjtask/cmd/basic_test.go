package cmd_test

import (
	"strings"
	"testing"
)

func TestCreateTask(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task", "Test description")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Test task") {
		t.Error("task not found in output")
	}

}

func TestCreateDraft(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "--draft", "@", "Draft task")

	output := repo.Run("jjtask", "find", "--status", "all")
	if !strings.Contains(output, "[task:draft]") {
		t.Error("draft flag not found")
	}

}

func TestFlagUpdatesStatus(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task")
	taskID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "wip", "--rev", taskID)

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "[task:wip]") {
		t.Error("wip flag not found")
	}

}

func TestFindShowsTasks(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task A")
	repo.Run("jjtask", "create", "Task B")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Task A") || !strings.Contains(output, "Task B") {
		t.Error("tasks not found in output")
	}

}

func TestFindRevsetFiltersTasks(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "My task")
	repo.Run("jj", "new", "-m", "Regular commit")

	output := repo.Run("jjtask", "find", "-r", "all()")
	if !strings.Contains(output, "My task") {
		t.Error("task not found in output")
	}
	if strings.Contains(output, "Regular commit") {
		t.Error("regular commit should not appear in task find")
	}

}

func TestShowDesc(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test title", "Test body content")
	taskID := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskID)

	output := repo.Run("jjtask", "show-desc")
	if !strings.Contains(output, "Test title") || !strings.Contains(output, "Test body content") {
		t.Error("description content not found")
	}

}

func TestParallelCreatesSiblings(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "parallel", "Task A", "Task B", "Task C")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Task A") ||
		!strings.Contains(output, "Task B") ||
		!strings.Contains(output, "Task C") {
		t.Error("parallel tasks not found")
	}

}

func TestPrime(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	output := repo.Run("jjtask", "prime")
	if output == "" {
		t.Error("prime produced no output")
	}

}

func TestCheckpoint(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	output := repo.Run("jjtask", "checkpoint", "-m", "test-checkpoint")
	if output == "" {
		t.Error("checkpoint produced no output")
	}

}

func TestDescTransform(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Original title", "## Context\nSome context here")
	taskID := repo.GetTaskID("todo")
	repo.Run("jjtask", "desc-transform", "--rev", taskID, "sed", "s/Original/Modified/")

	output := repo.Run("jjtask", "show-desc", "--rev", taskID)
	if !strings.Contains(output, "Modified title") {
		t.Error("transform not applied")
	}

}

func TestDescTransformError(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test title")
	taskID := repo.GetTaskID("todo")

	output := repo.RunExpectFail("jjtask", "desc-transform", "--rev", taskID, "nonexistent-cmd-xyz")
	if output == "" {
		t.Error("expected error output")
	}

}

func TestConfigTaskLogDiffStats(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("testfile.txt", "test content")
	repo.Run("jj", "describe", "-m", "Test commit with changes")

	output := repo.Run("jj", "log", "-r", "@", "--no-graph", "-T", "task_log")
	if output == "" {
		t.Error("task_log template produced no output")
	}

}

func TestConfigTaskLogShortDesc(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Short title", "## Context\nThis is a longer description\nwith multiple lines")
	taskID := repo.GetTaskID("todo")

	output := repo.Run("jj", "log", "-r", taskID, "--no-graph", "-T", "task_log")
	if !strings.Contains(output, "Short title") {
		t.Error("title not found in task_log output")
	}

}
