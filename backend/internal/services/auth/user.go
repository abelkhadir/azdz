package auth

import (
	"database/sql"
	"errors"
	"strconv"

	"social-network/internal/models"

	"github.com/mattn/go-sqlite3"
)

var database *sql.DB

func SetDB(db *sql.DB) {
	database = db
}

var ErrEmailExists = errors.New("email already registered")

func CreateUser(user models.User) error {
	return CreateUserWithDB(database, &user)
}

func CreateUserWithDB(db *sql.DB, user *models.User) error {
	if db == nil {
		return errors.New("auth database is not initialized")
	}
	result, err := db.Exec(
		`INSERT INTO users (
			email,
			password_hash,
			first_name,
			last_name,
			date_of_birth,
			avatar_path,
			nickname,
			about_me
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.DateOfBirth,
		user.Avatar,
		user.Nickname,
		user.AboutMe,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return ErrEmailExists
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = strconv.FormatInt(id, 10)
	return nil
}
func GetUserByEmail(email string) *models.User {
	if database == nil {
		return nil
	}

	var user models.User
	err := database.QueryRow(
		`SELECT
			id,
			email,
			password_hash,
			first_name,
			last_name,
			date_of_birth,
			COALESCE(avatar_path, ''),
			COALESCE(nickname, ''),
			COALESCE(about_me, '')
		FROM users
		WHERE email = ?`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.Avatar,
		&user.Nickname,
		&user.AboutMe,
	)
	if err != nil {
		return nil
	}
	return &user
}
