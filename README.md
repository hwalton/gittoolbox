# gittoolbox

A Go library for extracting version metadata and performing common Git repository checks.  
Useful for embedding Git commit info into builds, CI pipelines, or custom tooling.

## Features

- **GetVersionMetadata**:  
  Retrieve the latest commit date (with optional suffix for multiple commits per day) and commit hash for a set of files, directories, or globs.
- **AssertBranchIsCleanAndSynced**:  
  Ensure the current branch is up-to-date with its remote and has no uncommitted changes.

## Installation

```
go get github.com/hwalton/gittoolbox
```

## Usage

```go
import "github.com/hwalton/gittoolbox"

targets := []gittoolbox.PathTarget{
    {Path: "main.go", IncludeSubdirs: false},
    {Path: "cmd/", IncludeSubdirs: true},
}

commitDate, commitHash, err := gittoolbox.GetVersionMetadata(targets)
if err != nil {
    // handle error
}
fmt.Println("Commit Date:", commitDate)
fmt.Println("Commit Hash:", commitHash)

// Check if branch is clean and synced
if err := gittoolbox.AssertBranchIsCleanAndSynced(); err != nil {
    fmt.Println("Repo not clean/synced:", err)
}
```

## API

### type PathTarget

```go
type PathTarget struct {
    Path           string // File, directory, or glob pattern
    IncludeSubdirs bool   // If true, include subdirectories (for directories)
}
```

### func GetVersionMetadata

```go
func GetVersionMetadata(targets []PathTarget) (commitDate string, commitHash string, err error)
```

- `commitDate`: `YYYY-MM-DD` (with optional suffix, e.g. `2025-09-11-b` for multiple commits on the same day)
- `commitHash`: Short commit hash

### func AssertBranchIsCleanAndSynced

```go
func AssertBranchIsCleanAndSynced() error
```

- Returns error if the current branch is behind/ahead of remote or has uncommitted changes.

## License

Apache 2.0. See [LICENSE](LICENSE).