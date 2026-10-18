package models

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	DateOfBirth  string `json:"dateOfBirth"`
	Avatar       string `json:"avatar"`
	Nickname     string `json:"nickname"`
	AboutMe      string `json:"aboutMe"`
}

type RegisterRequest struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	DateOfBirth string
	Nickname    string
	AboutMe     string
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
