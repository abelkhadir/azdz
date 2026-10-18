package validation

import (
	"net/mail"
	"regexp"
	"time"
	"unicode/utf8"

	"social-network/internal/models"
)

var nameRe = regexp.MustCompile(`^\p{L}{2,14}$`)

func Register(req models.RegisterRequest) map[string]string {
	errs := make(map[string]string)

	// email
	switch {
	case req.Email == "":
		errs["email"] = "Email is required"
	case len(req.Email) > 254:
		errs["email"] = "Email must be at most 254 characters long"
	default:
		addr, err := mail.ParseAddress(req.Email)
		if err != nil || addr.Address != req.Email || addr.Name != "" {
			errs["email"] = "Invalid email address"
		}
	}

	// password
	switch {
	case req.Password == "":
		errs["password"] = "Password is required"
	case len(req.Password) < 8 || len(req.Password) > 64:
		errs["password"] = "Password must be between 8 and 64 characters long"
	}

	// names
	if req.FirstName == "" {
		errs["firstName"] = "First name is required"
	} else if !nameRe.MatchString(req.FirstName) {
		errs["firstName"] = "First name must be 2 to 14 letters"
	}
	if req.LastName == "" {
		errs["lastName"] = "Last name is required"
	} else if !nameRe.MatchString(req.LastName) {
		errs["lastName"] = "Last name must be 2 to 14 letters"
	}

	// date of birth
	if req.DateOfBirth == "" {
		errs["dateOfBirth"] = "Date of birth is required"
	} else if date, err := time.Parse("2006-01-02", req.DateOfBirth); err != nil {
		errs["dateOfBirth"] = "Use format YYYY-MM-DD"
	} else if date.After(time.Now()) || date.Year() < 1900 {
		errs["dateOfBirth"] = "Invalid date of birth"
	} else {
		now := time.Now()
		age := now.Year() - date.Year()
		if now.Month() < date.Month() || (now.Month() == date.Month() && now.Day() < date.Day()) {
			age--
		}
		if age < 16 {
			errs["dateOfBirth"] = "You must be at least 16 years old to register"
		} else if age > 100 {
			errs["dateOfBirth"] = "Age must be 100 years or less"
		}
	}

	if utf8.RuneCountInString(req.Nickname) > 31 {
		errs["nickname"] = "Nickname must be at most 31 characters long"
	}
	if utf8.RuneCountInString(req.AboutMe) > 500 {
		errs["aboutMe"] = "About me must be at most 500 characters long"
	}

	return errs
}
