package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "create-app":
		if len(os.Args) < 3 {
			fmt.Println("usage: veincli create-app <name>")
			os.Exit(1)
		}
		if err := createApp(os.Args[2]); err != nil {
			fmt.Printf("create-app error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("app scaffold created")
	case "generate":
		if len(os.Args) < 4 {
			fmt.Println("usage: veincli generate <module|endpoint> <name>")
			os.Exit(1)
		}
		kind := os.Args[2]
		name := os.Args[3]
		if err := generate(kind, name); err != nil {
			fmt.Printf("generate error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("artifact generated")
	default:
		usage()
	}
}

func usage() {
	fmt.Println("veincli commands:")
	fmt.Println("  create-app <name>")
	fmt.Println("  generate module <name>")
	fmt.Println("  generate endpoint <name>")
}

func createApp(name string) error {
	root, err := sanitizeRelativePath(name)
	if err != nil {
		return fmt.Errorf("invalid app name")
	}

	dirs := []string{
		filepath.Join(root, "cmd", "api"),
		filepath.Join(root, "internal", "data"),
		filepath.Join(root, "internal", "validator"),
		filepath.Join(root, "migrations"),
	}
	for _, dir := range dirs {
		// #nosec G703 -- `dir` is built from sanitized relative `root`.
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}

	env := strings.Join([]string{
		"APP_NAME=" + name,
		"APP_VERSION=1.0.0",
		"PORT=4000",
		"MY_ENV=development",
		"DB_DSN=postgres://postgres:postgres@localhost:5432/" + root + "?sslmode=disable",
		"TOKEN_SECRET=change-me",
		"CORS_TRUSTED_ORIGINS=http://localhost:3000",
		"RATE_LIMIT_RPS=5",
		"RATE_LIMIT_BURST=10",
	}, "\n") + "\n"

	// #nosec G703 -- destination path is constrained to sanitized relative `root`.
	return os.WriteFile(filepath.Join(root, ".env.example"), []byte(env), 0o600)
}

func generate(kind, name string) error {
	if name == "" {
		return fmt.Errorf("name must be provided")
	}

	safeName := strings.ToLower(strings.TrimSpace(name))
	safeName = strings.ReplaceAll(safeName, " ", "_")
	safeName, err := sanitizeName(safeName)
	if err != nil {
		return err
	}

	switch kind {
	case "module":
		path := filepath.Join("internal", safeName, safeName+".go")
		content := "package " + safeName + "\n\n// TODO: implement module logic\n"
		return writeIfMissing(path, content)
	case "endpoint":
		path := filepath.Join("cmd", "api", safeName+".go")
		content := "package main\n\nimport \"net/http\"\n\nfunc (app *application) " + safeName + "Handler(w http.ResponseWriter, r *http.Request) {\n\t_ = app.writeJSON(w, http.StatusOK, envelope{\"message\": \"" + safeName + " endpoint\"}, nil)\n}\n"
		return writeIfMissing(path, content)
	default:
		return fmt.Errorf("unsupported generator kind: %s", kind)
	}
}

func writeIfMissing(path, content string) error {
	cleanPath, err := sanitizeRelativePath(path)
	if err != nil {
		return err
	}

	// #nosec G703 -- `cleanPath` is validated by sanitizeRelativePath.
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil {
		return err
	}
	// #nosec G703 -- `cleanPath` is validated by sanitizeRelativePath.
	if _, err := os.Stat(cleanPath); err == nil {
		return fmt.Errorf("file already exists: %s", cleanPath)
	}
	// #nosec G703 -- `cleanPath` is validated by sanitizeRelativePath.
	return os.WriteFile(cleanPath, []byte(content), 0o600)
}

func sanitizeName(value string) (string, error) {
	clean := strings.TrimSpace(value)
	if clean == "" || !safeNamePattern.MatchString(clean) {
		return "", fmt.Errorf("name can only contain letters, digits, underscore, and hyphen")
	}
	return clean, nil
}

func sanitizeRelativePath(path string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must be a safe relative path")
	}

	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("path traversal is not allowed")
		}
	}

	return clean, nil
}
