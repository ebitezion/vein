package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	inputs := struct {
		appName string `json`
		version string `version`
	}{}
    
	//write output

	state:=map[string]string{
		AppName:app.config.appName,
		Env:app.config.env,
		Version:app.
	}

	
}
