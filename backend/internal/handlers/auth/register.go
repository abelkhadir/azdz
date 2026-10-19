package auth

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"social-network/internal/handlers/auth/avatar"
	"social-network/internal/handlers/auth/validation"
	"social-network/internal/models"
	"social-network/internal/pkg/response"
	authservice "social-network/internal/services/auth"
)

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.checkExistingSession(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, formDataSize*2) //10 MB
	if err := r.ParseMultipartForm(formDataSize); err != nil {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			response.WriteError(w, "Registration form must be at most 10 MB", http.StatusRequestEntityTooLarge)
			return
		}
		response.WriteError(w, "Some registration fields are invalid", http.StatusBadRequest)
		log.Println("Error parsing form data:", err)
		return
	}
	defer r.MultipartForm.RemoveAll()
	req := models.RegisterRequest{
		Email:       strings.ToLower(strings.TrimSpace(r.FormValue("email"))),
		Password:    r.FormValue("password"),
		FirstName:   strings.TrimSpace(r.FormValue("firstName")),
		LastName:    strings.TrimSpace(r.FormValue("lastName")),
		DateOfBirth: strings.TrimSpace(r.FormValue("dateOfBirth")),
		Nickname:    strings.TrimSpace(r.FormValue("nickname")),
		AboutMe:     strings.TrimSpace(r.FormValue("aboutMe")),
	}
	if req.Nickname == "" {
		req.Nickname = req.FirstName + " " + req.LastName
	}
	if errors := validation.Register(req); len(errors) > 0 {
		validation.WriteError(w, errors)
		return
	}
	passwordHash, err := authservice.HashPassword(req.Password)
	if err != nil {
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error hashing password:", err)
		return
	}
	avatarURL, ok := avatar.Save(w, r)
	if !ok {
		return
	}
	user := models.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  req.DateOfBirth,
		Nickname:     req.Nickname,
		AboutMe:      req.AboutMe,
		Avatar:       avatarURL,
	}
	if avatarURL != "" {
		defer func() {
			if user.ID == "" {
				if err := avatar.Remove(avatarURL); err != nil {
					log.Println("Error removing avatar:", err)
				}
			}
		}()
	}
	if err := authservice.CreateUserWithDB(h.DB, &user); err != nil {
		if errors.Is(err, authservice.ErrEmailExists) {
			response.WriteError(w, "Email is already registered", http.StatusConflict)
			return
		}
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error creating user:", err)
		return
	}
	cookie, err := createSession(h.DB, user.ID)
	if err != nil {
		if _, cleanupErr := h.DB.Exec(`DELETE FROM users WHERE id = ?`, user.ID); cleanupErr != nil {
			log.Println("Error removing user after session failure:", cleanupErr)
		} else {
			user.ID = ""
		}
		log.Println("Error creating registration session:", err)
		response.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
	response.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully",
		"userId":  user.ID,
	})
}
