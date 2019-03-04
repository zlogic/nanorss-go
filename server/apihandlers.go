package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/zlogic/nanorss-go/data"
)

func handleBadCredentials(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Bad credentials for user %v", err)
	http.Error(w, "Bad credentials", http.StatusUnauthorized)
}

func validateUserForApi(w http.ResponseWriter, r *http.Request, s *Services) *data.User {
	username := s.cookieHandler.GetUsername(w, r)
	if username == "" {
		http.Error(w, "Bad credentials", http.StatusUnauthorized)
		return nil
	}

	user, err := s.db.GetUser(username)
	if err != nil {
		handleError(w, r, err)
		return nil
	}
	if user == nil {
		handleBadCredentials(w, r, fmt.Errorf("Unknown username %v", username))
	}
	return user
}

func LoginHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
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

		user, err := s.db.GetUser(username)
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
		cookie := s.cookieHandler.NewCookie()
		s.cookieHandler.SetCookieUsername(cookie, username)
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

func FeedHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := validateUserForApi(w, r, s)
		if user == nil {
			return
		}

		items, err := s.feedListHelper.GetAllItems(user)
		if items == nil {
			items = make([]*Item, 0)
		}
		if err != nil {
			handleError(w, r, err)
			return
		}
		err = json.NewEncoder(w).Encode(items)
		if err != nil {
			handleError(w, r, err)
			return
		}
	}
}

func FeedItemHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// There are no secrets, but still better check we have a valid user
		user := validateUserForApi(w, r, s)
		if user == nil {
			return
		}

		type clientFeedItem struct {
			URL       string
			Contents  string
			Date      time.Time
			Plaintext bool
		}

		getItem := func(key string) *clientFeedItem {
			if strings.HasPrefix(key, data.FeeditemKeyPrefix) {
				feeditemKey, err := data.DecodeFeeditemKey([]byte(key))
				if err != nil {
					log.Printf("Failed to parse feed item key %v", err)
					return nil
				}
				feedItem, err := s.db.GetFeeditem(feeditemKey)
				if err != nil {
					log.Printf("Failed to parse feed item key %v", err)
					return nil
				}
				if feedItem == nil {
					return nil
				}
				return &clientFeedItem{
					Contents:  feedItem.Contents,
					Date:      feedItem.Date,
					URL:       feedItem.URL,
					Plaintext: false,
				}
			} else if strings.HasPrefix(key, data.PagemonitorKeyPrefix) {
				pagemonitorKey, err := data.DecodePagemonitorKey([]byte(key))
				if err != nil {
					log.Printf("Failed to parse pagemonitor page key %v", err)
					return nil
				}
				pagemonitorPage, err := s.db.GetPage(pagemonitorKey)
				if err != nil {
					log.Printf("Failed to parse pagemonitor page key %v", err)
					return nil
				}
				if pagemonitorPage == nil {
					return nil
				}
				// Bootstrap automatically handles line endings
				return &clientFeedItem{
					Contents:  pagemonitorPage.Delta,
					Date:      pagemonitorPage.Updated,
					URL:       pagemonitorKey.URL,
					Plaintext: true,
				}
			}
			log.Printf("Unknown item key format %v", key)
			return nil
		}

		vars := mux.Vars(r)
		key := strings.Replace(vars["key"], "-", "/", -1)
		item := getItem(key)

		if item == nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(item); err != nil {
			handleError(w, r, err)
		}
	}
}

func SettingsHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := s.cookieHandler.GetUsername(w, r)
		if username == "" {
			handleBadCredentials(w, r, fmt.Errorf("Unknown username %v", username))
			return
		}

		user, err := s.db.GetUser(username)
		if err != nil {
			handleError(w, r, err)
			return
		}
		if user == nil {
			handleBadCredentials(w, r, fmt.Errorf("Unknown username %v", username))
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				handleError(w, r, err)
				return
			}

			newPassword := r.Form.Get("Password")
			if newPassword != "" {
				user.SetPassword(newPassword)
			}
			user.Opml = r.Form.Get("Opml")
			user.Pagemonitor = r.Form.Get("Pagemonitor")

			newUsername := r.Form.Get("Username")
			if username != newUsername {
				err := s.db.SetUsername(user, newUsername)
				if err != nil {
					handleError(w, r, err)
					return
				}
				username = newUsername
				// Force logout
				cookie := s.cookieHandler.NewCookie()
				http.SetCookie(w, cookie)
			} else if err := s.db.SaveUser(user); err != nil {
				handleError(w, r, err)
				return
			}

			//Reload user to return updated values
			user, err = s.db.GetUser(newUsername)
			if err != nil {
				handleError(w, r, err)
				return
			}
		}

		type clientUser struct {
			Username    string
			Opml        string
			Pagemonitor string
		}

		returnUser := &clientUser{Username: username, Opml: user.Opml, Pagemonitor: user.Pagemonitor}

		if err := json.NewEncoder(w).Encode(returnUser); err != nil {
			handleError(w, r, err)
		}
	}
}

func RefreshHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := validateUserForApi(w, r, s)
		if user == nil {
			return
		}

		s.fetcher.Refresh()

		if _, err := io.WriteString(w, "OK"); err != nil {
			log.Printf("Failed to write response %v", err)
		}
	}
}
