package main

import "net/http"

func (app *application) openGate(w http.ResponseWriter, r *http.Request) {
	_ = app.writeJSON(w, http.StatusOK, envelope{"message": "gate opened"}, nil)
}
