package data

import (
	"database/sql"
	"errors"
)

// Define a custom ErrRecordNotFound error.
var (
	ErrRecordNotFound = errors.New("record not found")
)

// Create a Models struct which wraps the UserModel and others
type Models struct {
	Users UserModel
	Tx    TxManager
}

// New() method returns a Models struct containing.
func NewModels(db *sql.DB) Models {
	return Models{
		Users: UserModel{DB: db},
		Tx:    TxManager{DB: db},
	}
}
