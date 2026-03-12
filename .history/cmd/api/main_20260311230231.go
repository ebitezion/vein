package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	AppName = os.Getenv("APP_NAME")
	Version = os.Getenv("APP_VERSION")
)

type response struct {
	Greet string
}

// application type allows for application dependency injection
type application struct {
	config config
	log    *log.Logger
}

// config type allows for system configuration
type config struct {
	port    int
	env     string
	appName string
	version string
	db      struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	//set config command-line flags
	cfg := config{}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		fmt.Println("[Main] Error from port conversion", err)
	}
	cfg.version = Version
	cfg.db.dsn = os.Getenv("DB_DSN")
	flag.IntVar(&cfg.port, "port", port, "This is a port flag. -port:4000")
	flag.StringVar(&cfg.appName, "appName", os.Getenv("APP_NAME"), "This is the application Name")
	flag.StringVar(&cfg.env, "env", os.Getenv("MY_ENV"), "This is the working Environment. staging|development|production")

	// Read the connection pool settings from command-line flags into the config struct.
	// Notice the default values that we're using?
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.Parse()

	//Set DB
	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer db.Close()
	log.Printf("database connection pool established")
	//Initialize application struct
	log := log.New(os.Stdout, "[Vien Framework]", log.Ldate|log.Ltime|log.Lshortfile)

	app := application{
		config: cfg,
		log:    log,
	}

	//Intialize routes
	srv := &http.Server{
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		Addr:         fmt.Sprintf(":%v", cfg.port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.log.Println(" ---------------------------------------------------------------")
	app.log.Printf("  Starting Server on PORT %d and Env as %s", cfg.port, cfg.env)
	app.log.Println(" ---------------------------------------------------------------")
	err = srv.ListenAndServe()
	if err != nil {
		app.log.Printf("[MAIN|SERVER]%v", err)
	}
}

// openDB() function returns a sql.DB connection pool.
func openDB(cfg config) (*sql.DB, error) {

	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	// Return the sql.DB connection pool.
	return db, nil
}

func run(input interface{}) *response {
	response := &response{}
	switch v := input.(type) {
	case string:
		response.Greet = strings.ToLower(v)
	case nil:
		response.Greet = strings.ToLower(AppName)
	default:
		response.Greet = "unknown input type"
	}
	return response
}
