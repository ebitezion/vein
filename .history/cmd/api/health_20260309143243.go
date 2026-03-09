package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {

	//write output

	data := map[string]string{
		AppName: app.config.appName,
		Version: app.config.version,
		ENV:     app.config.env,
	}

	app.writeJSON(w, http.StatusOK, , nil)
}
