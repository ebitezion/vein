package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
	root := filepath.Clean(name)
	if root == "." || root == "" {
		return fmt.Errorf("invalid app name")
	}

	dirs := []string{
		filepath.Join(root, "cmd", "api"),
		filepath.Join(root, "internal", "data"),
		filepath.Join(root, "internal", "validator"),
		filepath.Join(root, "migrations"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	env := strings.Join([]string{
		"APP_NAME=" + name,
		"APP_VERSION=1.0.0",
		"PORT=4000",
		"MY_ENV=development",
		"DB_DSN=postgres://postgres:postgres@localhost:5432/" + name + "?sslmode=disable",
		"TOKEN_SECRET=change-me",
		"CORS_TRUSTED_ORIGINS=http://localhost:3000",
		"RATE_LIMIT_RPS=5",
		"RATE_LIMIT_BURST=10",
	}, "\n") + "\n"

	return os.WriteFile(filepath.Join(root, ".env.example"), []byte(env), 0o644)
}

func generate(kind, name string) error {
	if name == "" {
		return fmt.Errorf("name must be provided")
	}

	safeName := strings.ToLower(strings.TrimSpace(name))
	safeName = strings.ReplaceAll(safeName, " ", "_")

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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
