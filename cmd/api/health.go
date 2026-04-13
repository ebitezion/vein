package main

import (
	"context"
	"net/http"
	"time"
)

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

	_ = app.writeJSON(w, http.StatusOK, data, nil)
}

func (app *application) liveness(w http.ResponseWriter, r *http.Request) {
	_ = app.writeJSON(w, http.StatusOK, envelope{
		"liveness": envelope{
			"status":    "alive",
			"startedAt": app.startTime.Format(time.RFC3339),
		},
	}, nil)
}

func (app *application) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := app.model.Users.DB.PingContext(ctx); err != nil {
		app.errorResponse(w, r, http.StatusServiceUnavailable, "database not ready")
		return
	}

	_ = app.writeJSON(w, http.StatusOK, envelope{
		"readiness": envelope{
			"status": "ready",
		},
	}, nil)
}
