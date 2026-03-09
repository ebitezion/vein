package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {

	//write output
	data := map[string]string{
		"AppName": app.config.appName,
		"Version": app.config.version,
		"MY_ENV":  app,
	}

	app.writeJSON(w, http.StatusOK, data, nil)
}
