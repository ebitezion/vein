package main

import (
	"net/http"
	"strings"

	"github.com/ebitezion/vein/internal/data"
	"github.com/ebitezion/vein/internal/validator"
)

const (
	challengeEmail    = "admin@vein.dev"
	challengePassword = "VeinPass#2026!"
)

func (app *application) issueToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(input.Email != "", "email", "must be provided")
	v.Check(validator.Matches(input.Email, validator.EmailRX), "email", "must be a valid email address")
	v.Check(input.Password != "", "password", "must be provided")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if !app.dbEnabled {
		if strings.ToLower(strings.TrimSpace(input.Email)) != challengeEmail || input.Password != challengePassword {
			app.unauthorizedResponse(w, r)
			return
		}

		token, err := app.generateToken("user-1", "admin", app.config.security.tokenTTL)
		if err != nil {
			app.serverErrorResponse(w, r)
			return
		}
		_ = app.writeJSON(w, http.StatusCreated, envelope{"auth": envelope{
			"token":      token,
			"expires_in": int(app.config.security.tokenTTL.Seconds()),
			"role":       "admin",
		}}, nil)
		return
	}

	user, err := app.model.Users.GetByEmail(strings.ToLower(input.Email))
	if err != nil || !verifyPasswordHash(input.Password, user.PasswordHash) {
		app.unauthorizedResponse(w, r)
		return
	}

	if user.Status != "active" {
		app.forbiddenResponse(w, r)
		return
	}

	token, err := app.generateToken(user.ID, user.Role, app.config.security.tokenTTL)
	if err != nil {
		app.serverErrorResponse(w, r)
		return
	}
	_ = app.writeJSON(w, http.StatusCreated, envelope{"auth": envelope{
		"token":      token,
		"expires_in": int(app.config.security.tokenTTL.Seconds()),
		"role":       user.Role,
	}}, nil)
}

func (app *application) listUsers(w http.ResponseWriter, r *http.Request) {
	if !app.dbEnabled {
		app.errorResponse(w, r, http.StatusServiceUnavailable, "users endpoint requires DB setup")
		return
	}

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
