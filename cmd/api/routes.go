package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	routes := httprouter.New()

	routes.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFoundErrorResponse(w, r)
	})
	routes.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.methodNotAllowedErrorResponse(w, r)
	})

	routes.HandlerFunc(http.MethodGet, "/healthcheck", app.healthcheck)
	routes.HandlerFunc(http.MethodGet, "/liveness", app.liveness)
	routes.HandlerFunc(http.MethodGet, "/readiness", app.readiness)
	routes.HandlerFunc(http.MethodGet, "/metrics", app.metricsHandler)

	routes.HandlerFunc(http.MethodPost, "/v1/auth/token", app.issueToken)
	routes.Handler(http.MethodGet, "/v1/users", app.authenticate(app.requireRoles("admin", "manager")(http.HandlerFunc(app.listUsers))))
	routes.HandlerFunc(http.MethodPost, "/v1/jobs/audit", app.enqueueAuditJob)

	return app.chain(routes)
}
