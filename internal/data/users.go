package data

import (
	"database/sql"
	"time"

	"github.com/ebitezion/vein/internal/validator"
)

// Define a MovieModel struct type which wraps a sql.DB connection pool.
type UserModel struct {
	DB *sql.DB
}
type User struct {
	ID            string    `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	PasswordHash  string    `json:"-"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ValidateUsers(v *validator.Validator, user *User) {

	// First Name
	v.Check(user.FirstName != "", "first_name", "must be provided")
	v.Check(len(user.FirstName) <= 50, "first_name", "must not be more than 50 bytes long")

	// Last Name
	v.Check(user.LastName != "", "last_name", "must be provided")
	v.Check(len(user.LastName) <= 50, "last_name", "must not be more than 50 bytes long")

	// Email
	v.Check(user.Email != "", "email", "must be provided")
	v.Check(len(user.Email) <= 255, "email", "must not be more than 255 bytes long")
	v.Check(validator.Matches(user.Email, validator.EmailRX), "email", "must be a valid email address")

	// Phone (optional but must be reasonable length if provided)
	if user.Phone != "" {
		v.Check(len(user.Phone) <= 30, "phone", "must not be more than 30 bytes long")
	}

	// Password Hash
	v.Check(user.PasswordHash != "", "password", "must be provided")

	// Role validation
	v.Check(user.Role != "", "role", "must be provided")
	v.Check(validator.In(user.Role, "user", "admin", "manager"), "role", "must be a valid role")

	// Status validation
	v.Check(user.Status != "", "status", "must be provided")
	v.Check(validator.In(user.Status, "active", "disabled", "suspended"), "status", "must be a valid status")
}
