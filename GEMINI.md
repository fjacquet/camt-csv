# camt-csv Development Guidelines

Last updated: 2025-10-12

## Active Technologies
- Go 1.22+ + spf13/cobra, spf13/viper, github.com/sirupsen/logrus, gopkg.in/yaml.v3 (001-review-the-codebase)
- Files (for constitution definitions and generated reports) (001-review-the-codebase)
- Go 1.22+ + `spf13/cobra`, `spf13/viper`, `github.com/sirupsen/logrus`, `gopkg.in/yaml.v3`, `go/ast`, `go/parser`, `go/token`, `github.com/stretchr/testify`, `github.com/shopspring/decimal`, `github.com/gocarina/gocsv`, `github.com/joho/godotenv` (002-enahance-the-codebase)
- Files (constitution definitions, generated reports), YAML files (categories, creditors, debitors) (002-enahance-the-codebase)

- Go 1.22+ + `go/ast`, `go/parser`, `go/token` (for Go code analysis); `gopkg.in/yaml.v3` (for parsing constitution/config); `github.com/sirupsen/logrus` (for logging); `spf13/cobra`, `spf13/viper` (for CLI and config).

## Project Structure

```
src/
tests/
```

## Commands

# Add commands for Go 1.22+

## Code Style

Go 1.22+: Follow standard conventions

## Go Development Guidelines

This document outlines best practices for writing secure and robust Go code, based on security analysis results.

### 1. Error Handling (G104)

Never ignore errors returned by functions. Inadequate error handling can mask bugs and security vulnerabilities.

**Bad:**

```go
file, _ := os.Open("a_file.txt")
defer file.Close()
```

**Good:**

```go
file, err := os.Open("a_file.txt")
if err != nil {
    log.Fatalf("failed to open file: %s", err)
}
defer file.Close()
```

### 2. Filesystem Security

#### Directory Permissions (G301)

When creating directories, use restrictive permissions to prevent unauthorized access. It is recommended to use `0750` or less.

**Bad:**

```go
os.MkdirAll(path, 0777) // Overly permissive
os.MkdirAll(path, os.ModePerm) // Equivalent to 0777
```

**Good:**

```go
// Owner can read/write/execute, group can read/execute.
if err := os.MkdirAll(path, 0750); err != nil {
    // Handle error
}
```

#### File Permissions (G306)

Similarly, when creating files, ensure permissions are as restrictive as possible. `0600` is a good default for files that should only be accessible by the owner.

**Bad:**

```go
os.WriteFile(filepath, data, 0644) // Group and others can read the file.
```

**Good:**

```go
// Owner can read/write only.
err := os.WriteFile(filepath, data, 0600)
if err != nil {
    // Handle error
}
```

#### Path Traversal Prevention (G304)

Never trust file paths from external sources (user input, configuration, etc.). Always validate and sanitize paths to ensure they do not allow access to files outside the intended directory.

**Bad:**

```go
fileName := r.URL.Query().Get("file")
data, err := os.ReadFile("/var/www/data/" + fileName) // Vulnerable to ../../etc/passwd
```

**Good:**

```go
import (
    "path/filepath"
    "strings"
)

baseDir := "/var/www/data/"
fileName := r.URL.Query().Get("file")
absPath, err := filepath.Abs(filepath.Join(baseDir, fileName))
if err != nil {
    // Handle error
}

// Verify that the final path is prefixed by the base directory.
if !strings.HasPrefix(absPath, baseDir) {
    // Handle error: path traversal attempt
}

data, err := os.ReadFile(absPath)
```

### 3. Command Injection Prevention (G204)

Launching external processes with user input is risky. If you must do so, never construct command strings by concatenation. Pass arguments separately.

**Bad:**

```go
out, err := exec.Command("bash", "-c", "pdftotext " + userInput).Output()
```

**Good:**

```go
// Arguments are passed separately and not interpreted by the shell.
cmd := exec.Command("pdftotext", "-layout", "-raw", pdfFile, tempFile)
err := cmd.Run()
```

### 4. Avoid Hardcoded Credentials (G101)

Never include identifiers, passwords, API keys, or other secrets directly in the source code. Use environment variables, secure configuration files, or secret management services.

**Bad:**

```go
password := "mysecretpassword"
db, err := sql.Open("mysql", "user:"+password+"@/dbname")
```

**Good:**

```go
password := os.Getenv("DB_PASSWORD")
if password == "" {
    log.Fatal("DB_PASSWORD environment variable not set")
}
db, err := sql.Open("mysql", "user:"+password+"@/dbname")
```

_Note: In the case of `internal/xmlutils/constants.go`, the G101 alert is likely a false positive, as the strings
resemble XPath queries and not credentials. You can ignore them with a `//gosec:G101` comment._


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
