package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {

	//write output

	state := map[string]string{
		AppName: app.config.appName,
		Env:     app.,
		Version: app.config.version,
	}

	app.writeJSON(w, http.StatusOK, state, nil)
}
