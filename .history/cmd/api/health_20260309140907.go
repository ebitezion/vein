package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	inputs := struct {
		appName string `json`
		version string `version`
	}{}
    
	//write output

	map[string]string{
		
	}
}
