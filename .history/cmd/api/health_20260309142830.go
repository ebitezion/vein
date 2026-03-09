package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
    
	//write output

	state:=map[string]string{
		AppName:app.config.appName,
		Env:app.config.env,
		Version:app.
	}

  app.writeJSON(w,http.StatusOK, state, http.Header{})
}
