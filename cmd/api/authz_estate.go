package main

import "net/http"

func (app *application) requireEstateRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := app.userIDFromContext(r.Context())
			estateID := app.estateIDFromContext(r.Context())
			if userID == "" || estateID == "" {
				app.forbiddenResponse(w, r)
				return
			}

			role, ok := app.estateRoles.GetRole(userID, estateID)
			if !ok || role != requiredRole {
				app.forbiddenResponse(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
