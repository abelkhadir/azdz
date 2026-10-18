package validation

import (
	"social-network/internal/models"
)

func Login(req models.LoginRequest) map[string]string {
	errs := make(map[string]string)

	// email
	switch {
	case req.Email == "":
		errs["email"] = "Email is required"
	case len(req.Email) > 254:
		errs["email"] = "Invalid credentials"
	}

	// password
	switch {
	case req.Password == "":
		errs["password"] = "Password is required"
	case len(req.Password) < 8 || len(req.Password) > 64:
		errs["password"] = "Invalid credentials"
	}

	return errs
}
