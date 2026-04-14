package data

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

const dbTimeout = 3 * time.Second

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

func (u UserModel) Insert(user User) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `INSERT INTO users(first_name, last_name, email, phone, password_hash, role, status, email_verified)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			 RETURNING id, created_at
	`
	args := []interface{}{
		user.FirstName,
		user.LastName,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.Role,
		user.Status,
		user.EmailVerified,
	}

	return u.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.ID, &user.CreatedAt)
}

func (u UserModel) Get(id int64) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if id < 1 {
		return nil, ErrRecordNotFound
	}
	stmt := `SELECT id, first_name, last_name, email, phone, role, status, email_verified, created_at 
			 FROM users
			 WHERE id = $1
	`

	var user User

	err := u.DB.QueryRowContext(ctx, stmt, id).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Phone, &user.Role, &user.Status, &user.EmailVerified, &user.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}

	}

	return &user, nil
}

func (u UserModel) Update(user User, id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `UPDATE users
			SET first_name = $1, last_name = $2, email = $3, phone = $4, password_hash = $5, role = $6, status = $7, email_verified = $8 
			WHERE id = $9
			RETURNING id
	     `
	args := []interface{}{
		user.FirstName,
		user.LastName,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.Role,
		user.Status,
		user.EmailVerified,
		id,
	}
	return u.DB.QueryRowContext(ctx, stmt, args...).Scan(&user.ID)
}

func (u UserModel) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, ErrRecordNotFound
	}

	stmt := `SELECT id, first_name, last_name, email, phone, password_hash, role, status, email_verified, created_at, updated_at
		FROM users
		WHERE lower(email) = $1`

	var user User
	err := u.DB.QueryRowContext(ctx, stmt, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (u UserModel) List(filters Filters) ([]User, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query, err := userListQueryForSort(filters.Sort)
	if err != nil {
		return nil, Metadata{}, err
	}

	args := []interface{}{filters.PageSize, (filters.Page - 1) * filters.PageSize}

	rows, err := u.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	users := []User{}

	for rows.Next() {
		var user User
		if err := rows.Scan(
			&totalRecords,
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.Phone,
			&user.Role,
			&user.Status,
			&user.EmailVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, Metadata{}, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return users, metadata, nil
}

func userListQueryForSort(sort string) (string, error) {
	switch sort {
	case "created_at":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY created_at ASC, id ASC
			LIMIT $1 OFFSET $2`, nil
	case "-created_at":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY created_at DESC, id ASC
			LIMIT $1 OFFSET $2`, nil
	case "email":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY email ASC, id ASC
			LIMIT $1 OFFSET $2`, nil
	case "-email":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY email DESC, id ASC
			LIMIT $1 OFFSET $2`, nil
	case "first_name":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY first_name ASC, id ASC
			LIMIT $1 OFFSET $2`, nil
	case "-first_name":
		return `SELECT count(*) OVER(), id, first_name, last_name, email, phone, role, status, email_verified, created_at, updated_at
			FROM users
			ORDER BY first_name DESC, id ASC
			LIMIT $1 OFFSET $2`, nil
	default:
		return "", ErrRecordNotFound
	}
}
