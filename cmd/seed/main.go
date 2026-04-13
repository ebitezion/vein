package main

import (
	"database/sql"
	"fmt"
	"os"

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

	script, err := os.ReadFile(scriptPath)
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

	if _, err := db.Exec(string(script)); err != nil {
		fmt.Printf("unable to apply seed script: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("seed applied successfully")
}
