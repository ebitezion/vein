package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)


func (app *application) routes() *httprouter.Router{

	routes := httprouter.New()

	routes.HandlerFunc(http.MethodGet, "/healthcheck",http.HandlerFunc(app.health) )
}