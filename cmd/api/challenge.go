package main

import "net/http"

// challengeDowngradeRole simulates permission drift for the active estate.
func (app *application) challengeDowngradeRole(w http.ResponseWriter, r *http.Request) {
	userID := app.userIDFromContext(r.Context())
	estateID := app.estateIDFromContext(r.Context())
	if userID == "" || estateID == "" {
		app.forbiddenResponse(w, r)
		return
	}

	app.estateRoles.SetRole(userID, estateID, "resident")
	_ = app.writeJSON(w, http.StatusOK, envelope{"message": "role downgraded", "role": "resident"}, nil)
}
