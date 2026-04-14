package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Println("DB_DSN must be set")
		os.Exit(1)
	}

	scriptPath := os.Getenv("SEED_FILE")
	if scriptPath == "" {
		scriptPath = "seeds/000001_seed_users.sql"
	}

	script, err := readSeedScript(scriptPath)
	if err != nil {
		fmt.Printf("unable to read seed file: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("unable to open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// #nosec G201,G701 -- this command intentionally executes trusted seed SQL loaded from repository-scoped files.
	if _, err := db.Exec(string(script)); err != nil {
		fmt.Printf("unable to apply seed script: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("seed applied successfully")
}

func readSeedScript(path string) ([]byte, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid seed path")
	}

	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	defer root.Close()

	file, err := root.Open(clean)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
