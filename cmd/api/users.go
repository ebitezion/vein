package main

import (
	"net/http"

	"github.com/ebitezion/vein/internal/data"
	"github.com/ebitezion/vein/internal/validator"
)

func (app *application) issueToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Subject string `json:"subject"`
		Role    string `json:"role"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(input.Subject != "", "subject", "must be provided")
	v.Check(validator.In(input.Role, "user", "manager", "admin"), "role", "must be a valid role")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.generateToken(input.Subject, input.Role, app.config.security.tokenTTL)
	if err != nil {
		app.serverErrorResponse(w, r)
		return
	}
	_ = app.writeJSON(w, http.StatusCreated, envelope{"auth": envelope{"token": token}}, nil)
}

func (app *application) listUsers(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	v := validator.New()
	filters := data.Filters{
		Page:         app.readInt(qs, "page", 1, v),
		PageSize:     app.readInt(qs, "page_size", 20, v),
		Sort:         app.readString(qs, "sort", "created_at"),
		SortSafelist: []string{"created_at", "-created_at", "email", "-email", "first_name", "-first_name"},
	}

	data.ValidateFilters(v, filters)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	users, metadata, err := app.model.Users.List(filters)
	if err != nil {
		app.serverErrorResponse(w, r)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, envelope{
		"users":    users,
		"metadata": metadata,
	}, nil)
}

func (app *application) enqueueAuditJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Action == "" {
		input.Action = "unknown"
	}

	if err := app.queue.Publish(r.Context(), Job{Name: "audit.log", Payload: map[string]string{"action": input.Action}}); err != nil {
		app.serverErrorResponse(w, r)
		return
	}
	_ = app.writeJSON(w, http.StatusAccepted, envelope{"message": "job enqueued"}, nil)
}
