package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"os"
	"testing"

	"social-network/internal/models"
)

func TestSessionsRemainIsolatedAfterLogout(t *testing.T) {
	app := newTestApp(t)
	firstID := app.createUser("first", true)
	secondID := app.createUser("second", true)
	firstToken := app.createSession(firstID)
	secondToken := app.createSession(secondID)

	checkIdentity := func(token string, wantID int64) {
		t.Helper()
		response := app.request(http.MethodGet, "/api/auth/me", token, nil)
		assertStatus(t, response, http.StatusOK)
		var payload struct {
			User models.User `json:"user"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.User.ID != fmt.Sprint(wantID) {
			t.Fatalf("user ID = %s, want %d", payload.User.ID, wantID)
		}
	}

	checkIdentity(firstToken, firstID)
	checkIdentity(secondToken, secondID)
	assertStatus(t, app.request(http.MethodPost, "/api/auth/logout", firstToken, nil), http.StatusOK)
	assertStatus(t, app.request(http.MethodGet, "/api/auth/me", firstToken, nil), http.StatusUnauthorized)
	checkIdentity(secondToken, secondID)
}

func TestRegistrationAvatarFormatsAndFailureCleanup(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var jpg, pngImage, gifImage bytes.Buffer
	if err := jpeg.Encode(&jpg, picture, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngImage, picture); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifImage, picture, nil); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		data []byte
		want int
	}{
		{"optional avatar", nil, http.StatusCreated},
		{"JPEG", jpg.Bytes(), http.StatusCreated},
		{"PNG", pngImage.Bytes(), http.StatusCreated},
		{"GIF", gifImage.Bytes(), http.StatusCreated},
		{"invalid image", []byte("not an image"), http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			t.Chdir(t.TempDir())

			register := func() *http.Response {
				t.Helper()
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				for key, value := range map[string]string{
					"email": "avatar@example.com", "password": "password123",
					"firstName": "Alice", "lastName": "Smith", "dateOfBirth": "2000-01-01",
				} {
					if err := writer.WriteField(key, value); err != nil {
						t.Fatal(err)
					}
				}
				if test.data != nil {
					file, err := writer.CreateFormFile("avatar", "avatar.png")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := file.Write(test.data); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				request, err := http.NewRequest(http.MethodPost, app.server.URL+"/api/auth/register", &body)
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Origin", testOrigin)
				request.Header.Set("Content-Type", writer.FormDataContentType())
				response, err := app.server.Client().Do(request)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { response.Body.Close() })
				return response
			}

			assertStatus(t, register(), test.want)
			if test.want == http.StatusCreated {
				// A second upload for an existing email must not leave a file behind.
				assertStatus(t, register(), http.StatusConflict)
			}
			files, err := os.ReadDir("uploads/avatars")
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			wantFiles := 0
			if test.want == http.StatusCreated && test.data != nil {
				wantFiles = 1
			}
			if len(files) != wantFiles {
				t.Fatalf("saved avatar count = %d, want %d", len(files), wantFiles)
			}
			var userCount int
			if err := app.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
				t.Fatal(err)
			}
			wantUsers := 0
			if test.want == http.StatusCreated {
				wantUsers = 1
			}
			if userCount != wantUsers {
				t.Fatalf("registered user count = %d, want %d", userCount, wantUsers)
			}
		})
	}
}
