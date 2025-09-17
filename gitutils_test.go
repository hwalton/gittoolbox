package gittoolbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// helpers

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func runGitWithEnv(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v with env %v failed: %v\n%s", args, env, err, string(out))
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "user.email", "test@example.com")
}

// write file, add and commit using the provided date/time string (no timezone required)
func writeAndCommit(t *testing.T, dir, fname, content, date string) {
	t.Helper()
	path := filepath.Join(dir, fname)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", fname)
	env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	runGitWithEnv(t, dir, env, "commit", "-m", "commit "+fname)
}

// Tests

func TestGetVersionMetadata_SingleFile(t *testing.T) {
	td := t.TempDir()
	initRepo(t, td)

	// single commit on 2025-09-10 (use date/time string without timezone)
	writeAndCommit(t, td, "a.txt", "hello", "2025-09-10 12:00:00")

	commitDate, commitHash, err := GetVersionMetadata([]PathTarget{{Path: filepath.Join(td, "a.txt"), IncludeSubdirs: false}})
	if err != nil {
		t.Fatalf("GetVersionMetadata failed: %v", err)
	}
	if commitHash == "" {
		t.Fatalf("expected non-empty CommitHash")
	}
	// CommitDate should be YYYY-MM-DD (no timezone) and equal to date component
	if commitDate != "2025-09-10" {
		t.Fatalf("expected CommitDate 2025-09-10, got %q", commitDate)
	}
}

func TestGetVersionMetadata_MultipleCommits_Suffix(t *testing.T) {
	td := t.TempDir()
	initRepo(t, td)

	// two commits on same date -> expect suffix like "-b" for second commit
	date := "2025-09-11 09:00:00"
	writeAndCommit(t, td, "b.txt", "first", date)
	// second commit same date
	writeAndCommit(t, td, "b.txt", "second", date)

	commitDate, commitHash, err := GetVersionMetadata([]PathTarget{{Path: filepath.Join(td, "b.txt"), IncludeSubdirs: false}})
	if err != nil {
		t.Fatalf("GetVersionMetadata failed: %v", err)
	}
	if commitHash == "" {
		t.Fatalf("expected non-empty CommitHash")
	}
	if !strings.HasPrefix(commitDate, "2025-09-11") {
		t.Fatalf("expected CommitDate starting with 2025-09-11; got %q", commitDate)
	}
	// suffix should be present and end with a letter (e.g. -b)
	if !strings.Contains(commitDate, "-") || !strings.HasSuffix(commitDate, "b") {
		t.Fatalf("expected CommitDate to include suffix ending with 'b', got %q", commitDate)
	}
}

func TestGetVersionMetadata_GlobAndDirIncludeSubdirs(t *testing.T) {
	td := t.TempDir()
	initRepo(t, td)

	// create nested files
	if err := os.MkdirAll(filepath.Join(td, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeAndCommit(t, td, "root.txt", "root", "2025-09-12 08:00:00")
	writeAndCommit(t, td, filepath.ToSlash(filepath.Join("sub", "inner.txt")), "inner", "2025-09-12 09:00:00")

	// glob should pick up root.txt
	commitDate, commitHash, err := GetVersionMetadata([]PathTarget{{Path: filepath.Join(td, "*.txt"), IncludeSubdirs: false}})
	if err != nil {
		t.Fatalf("GetVersionMetadata with glob failed: %v", err)
	}
	if commitHash == "" || commitDate == "" {
		t.Fatalf("unexpected meta from glob: commitDate=%q commitHash=%q", commitDate, commitHash)
	}

	// directory with IncludeSubdirs true should see nested file commits
	commitDate2, commitHash2, err := GetVersionMetadata([]PathTarget{{Path: td, IncludeSubdirs: true}})
	if err != nil {
		t.Fatalf("GetVersionMetadata with dir IncludeSubdirs failed: %v", err)
	}
	if commitHash2 == "" || commitDate2 == "" {
		t.Fatalf("unexpected meta from dir includeSubdirs: commitDate=%q commitHash=%q", commitDate2, commitHash2)
	}
}

func TestGetVersionMetadata_NoMatchesGlob(t *testing.T) {
	td := t.TempDir()
	initRepo(t, td)

	// glob that matches nothing should return an error
	_, _, err := GetVersionMetadata([]PathTarget{{Path: filepath.Join(td, "no-such-*.txt"), IncludeSubdirs: false}})
	if err == nil {
		t.Fatalf("expected error for glob with no matches")
	}
}
