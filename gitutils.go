package gittoolbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PathTarget struct {
	Path           string
	IncludeSubdirs bool
}

func GetVersionMetadata(targets []PathTarget) (string, string, error) {
	// default to current dir if nothing provided
	if len(targets) == 0 {
		targets = []PathTarget{{Path: "."}}
	}

	resolved := []string{}
	for _, t := range targets {
		p := filepath.Clean(t.Path)

		// If path looks like a glob, expand it
		if strings.ContainsAny(p, "*?[") {
			matches, err := filepath.Glob(p)
			if err != nil {
				return "", "", fmt.Errorf("invalid glob %s: %w", p, err)
			}
			if len(matches) == 0 {
				return "", "", fmt.Errorf("glob %s: no matches", p)
			}
			for _, m := range matches {
				// treat each match like a target with the same IncludeSubdirs flag
				info, err := os.Stat(m)
				if err != nil {
					return "", "", fmt.Errorf("stat %s: %w", m, err)
				}
				if info.IsDir() {
					if t.IncludeSubdirs {
						resolved = append(resolved, m)
					} else {
						entries, err := os.ReadDir(m)
						if err != nil {
							return "", "", fmt.Errorf("read dir %s: %w", m, err)
						}
						for _, e := range entries {
							if e.IsDir() {
								continue
							}
							fp := filepath.Join(m, e.Name())
							resolved = append(resolved, fp)
						}
					}
				} else {
					resolved = append(resolved, m)
				}
			}
			continue
		}

		// non-glob: existing behavior
		info, err := os.Stat(p)
		if err != nil {
			return "", "", fmt.Errorf("stat %s: %w", p, err)
		}

		if info.IsDir() {
			if t.IncludeSubdirs {
				// passing the directory to git includes its subtree
				resolved = append(resolved, p)
			} else {
				entries, err := os.ReadDir(p)
				if err != nil {
					return "", "", fmt.Errorf("read dir %s: %w", p, err)
				}
				for _, e := range entries {
					if e.IsDir() {
						// skip nested directories when includeSubdirs == false
						continue
					}
					fp := filepath.Join(p, e.Name())
					resolved = append(resolved, fp)
				}
			}
		} else {
			// file: include as-is
			resolved = append(resolved, p)
		}
	}

	if len(resolved) == 0 {
		return "", "", fmt.Errorf("no files to inspect")
	}

	// determine working directory for git commands: use directory of first resolved path
	workDir := ""
	if len(resolved) > 0 {
		if info, err := os.Stat(resolved[0]); err == nil && !info.IsDir() {
			workDir = filepath.Dir(resolved[0])
		} else {
			workDir = resolved[0]
		}
	}

	// Build git args: e.g. git log -n 1 --pretty=format:%h -- <paths...>
	baseArgsHash := []string{"log", "-n", "1", "--pretty=format:%h", "--"}
	argsHash := append(baseArgsHash, resolved...)
	commitHash, err := gitOutputIn(workDir, argsHash...)
	if err != nil {
		return "", "", fmt.Errorf("git log hash: %w", err)
	}

	// Get latest commit date (YYYY-MM-DD)
	baseArgsDate := []string{"log", "-n", "1", "--date=format:%Y-%m-%d", "--pretty=format:%cd", "--"}
	argsDate := append(baseArgsDate, resolved...)
	commitDateNoSuffix, err := gitOutputIn(workDir, argsDate...)
	if err != nil {
		return "", "", fmt.Errorf("git log date: %w", err)
	}

	// Count commits on that date for the same set of paths
	baseArgsCount := []string{"log", "--date=format:%Y-%m-%d", "--pretty=format:%cd", "--"}
	argsCount := append(baseArgsCount, resolved...)
	countStr, err := gitOutputIn(workDir, argsCount...)
	if err != nil {
		return "", "", fmt.Errorf("git log count: %w", err)
	}

	count := 0
	for _, line := range strings.Split(countStr, "\n") {
		if strings.TrimSpace(line) == commitDateNoSuffix {
			count++
		}
	}

	// Suffix: a for first, b for second, etc.
	suffix := ""
	if count > 1 {
		// a=1, b=2, c=3, ...
		suffix = string(rune('a' + count - 1))
	}

	var commitDate string
	if suffix == "" || suffix == "a" {
		commitDate = commitDateNoSuffix
	} else {
		commitDate = fmt.Sprintf("%s-%s", commitDateNoSuffix, suffix)
	}

	return commitDate, commitHash, nil
}

func AssertBranchIsCleanAndSynced() error {
	branch, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	branch = strings.TrimSpace(branch)

	if _, err := gitOutput("fetch", "origin", branch, "--quiet"); err != nil {
		return err
	}

	status, err := gitOutput("rev-list", "--left-right", "--count", "origin/"+branch+"..."+branch)
	if err != nil {
		return err
	}
	parts := strings.Fields(status)
	if len(parts) != 2 {
		return fmt.Errorf("unexpected output from git rev-list")
	}
	if parts[0] != "0" {
		return fmt.Errorf("your branch is behind origin/%s", branch)
	}
	if parts[1] != "0" {
		return fmt.Errorf("your branch is ahead of origin/%s", branch)
	}

	changes, err := gitOutput("status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(changes) != "" {
		return fmt.Errorf("you have uncommitted changes")
	}
	return nil
}

func gitOutput(args ...string) (string, error) {
	return gitOutputIn("", args...)
}

// gitOutputIn runs git with args; if dir is non-empty it sets cmd.Dir so git runs in that directory.
func gitOutputIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}
