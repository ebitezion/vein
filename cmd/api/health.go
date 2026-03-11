package main

import "net/http"

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {

	//write output
	input := map[string]string{
		"AppName": app.config.appName,
		"Version": app.config.version,
		"MY_ENV":  app.config.env,
	}

	data := envelope{
		"healthcheck": input,
	}

	app.writeJSON(w, http.StatusOK, data, nil)
}
