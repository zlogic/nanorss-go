package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

func handleBadCredentials(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Bad credentials for user %v", err)
	http.Error(w, "Bad credentials", http.StatusUnauthorized)
}

func LoginHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			handleError(w, r, err)
			return
		}

		username := r.Form.Get("username")
		password := r.Form.Get("password")
		rememberMe, err := strconv.ParseBool(r.Form.Get("rememberMe"))
		if err != nil {
			log.Printf("Failed to parse rememberMe parameter %v", err)
			rememberMe = false
		}

		userService := s.db.NewUserService(username)
		user, err := userService.Get()
		if err != nil {
			handleError(w, r, err)
			return
		}
		if user == nil {
			handleBadCredentials(w, r, fmt.Errorf("User %v does not exist", username))
			return
		}
		err = user.ValidatePassword(password)
		if err != nil {
			handleBadCredentials(w, r, errors.Wrapf(err, "Invalid password for user %v", username))
			return
		}
		cookie := s.cookieHandler.newCookie()
		s.cookieHandler.setCookieUsername(cookie, username)
		if !rememberMe {
			cookie.Expires = time.Time{}
			cookie.MaxAge = 0
		}
		http.SetCookie(w, cookie)
		_, err = io.WriteString(w, "OK")
		if err != nil {
			log.Printf("Failed to write response %v", err)
		}
	}
}

func SettingsHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			handleError(w, r, err)
			return
		}

		username := s.cookieHandler.GetUsername(w, r)
		if username == "" {
			handleBadCredentials(w, r, fmt.Errorf("Unknown username %v", username))
			return
		}

		userService := s.db.NewUserService(username)
		user, err := userService.Get()
		if err != nil {
			handleError(w, r, err)
			return
		}

		newUsername := r.Form.Get("username")
		if username != newUsername {
			err := userService.SetUsername(newUsername)
			if err != nil {
				handleError(w, r, err)
				return
			}
			// Force logout
			cookie := s.cookieHandler.newCookie()
			http.SetCookie(w, cookie)
		}
		newPassword := r.Form.Get("password")
		if newPassword != "" {
			user.SetPassword(newPassword)
		}
		user.Opml = r.Form.Get("opml")
		user.Pagemonitor = r.Form.Get("pagemonitor")
		err = userService.Save(user)
		if err != nil {
			handleError(w, r, err)
		}
		_, err = io.WriteString(w, "OK")
	}
}
