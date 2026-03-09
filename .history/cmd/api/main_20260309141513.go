package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
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
	flag.IntVar(&cfg.port, "port", port, "This is a port flag. -port:4000")
	flag.StringVar(&cfg.appName, "appName", os.Getenv("APP_NAME"), "This is the application Name")
	flag.StringVar(&cfg.env, "env", os.Getenv("ENV"), "This is the working Environment. staging|development|production")

	//Initialize application struct
	log := log.New(os.Stdout, "[Vien Framework]", log.Ldate|log.Ltime|log.Lshortfile)

	app := application{
		config: cfg,
		log:    log,
	}

	//Intialize routes
	srv:= &http.ServeMux{
		
	}

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
