package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/server/auth"
)

// APIAuthHandler checks to see if the API is accessed by an authorized user,
// and returns an error if the request is done by an unauthorized user.
func APIAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			http.Error(w, "Bad credentials", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoginHandler authenticates the user and sets the encrypted session cookie if the user provided valid credentials.
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
			log.WithError(err).Error("Failed to parse rememberMe parameter")
			rememberMe = false
		}

		user, err := s.db.GetUser(username)
		if err != nil {
			handleError(w, r, err)
			return
		}
		if user == nil {
			log.Errorf("User %v doesn't exist", username)
			http.Error(w, "Bad credentials", http.StatusUnauthorized)
			return
		}
		err = user.ValidatePassword(password)
		if err != nil {
			log.WithError(err).Errorf("Invalid password for user %v", username)
			http.Error(w, "Bad credentials", http.StatusUnauthorized)
			return
		}
		err = s.cookieHandler.SetCookieUsername(w, username, rememberMe)
		if err != nil {
			log.WithError(err).Error("Failed to set username cookie")
			http.Error(w, "Failed to set username cookie", http.StatusInternalServerError)
			return
		}
		if _, err := io.WriteString(w, "OK"); err != nil {
			log.WithError(err).Error("Failed to write response")
		}
	}
}

// FeedHandler returns all feed (and page monitor) items for an authenticated user.
func FeedHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
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

// FeedItemHandler returns a feed (or page monitor) item for an authenticated user.
func FeedItemHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// There are no secrets, but still better check we have a valid user.
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		type clientFeedItem struct {
			URL           string
			Contents      string
			Date          time.Time
			Plaintext     bool
			MarkUnreadURL string
		}

		getItem := func(key []byte) *clientFeedItem {
			if data.IsFeeditemKey(key) {
				feeditemKey, err := data.DecodeFeeditemKey(key)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to parse feed item key")
					return nil
				}
				feedItem, err := s.db.GetFeeditem(feeditemKey)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to get feed item")
					return nil
				}
				if feedItem == nil {
					return nil
				}
				err = s.db.SetReadStatus(user, []byte(key), true)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to set read status for feed item")
				}
				return &clientFeedItem{
					Contents:      feedItem.Contents,
					Date:          feedItem.Date,
					URL:           feedItem.URL,
					Plaintext:     false,
					MarkUnreadURL: "api/items/" + escapeKeyForURL(key),
				}
			} else if data.IsPagemonitorKey(key) {
				pagemonitorKey, err := data.DecodePagemonitorKey(key)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to parse pagemonitor page key")
					return nil
				}
				pagemonitorPage, err := s.db.GetPage(pagemonitorKey)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to get pagemonitor page")
					return nil
				}
				if pagemonitorPage == nil {
					return nil
				}
				err = s.db.SetReadStatus(user, []byte(key), true)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to set read status for page")
				}
				// Bootstrap automatically handles line endings
				return &clientFeedItem{
					Contents:      pagemonitorPage.Delta,
					Date:          pagemonitorPage.Updated,
					URL:           pagemonitorKey.URL,
					Plaintext:     true,
					MarkUnreadURL: "api/items/" + escapeKeyForURL(key),
				}
			}
			log.WithField("key", key).Error("Unknown item key format")
			return nil
		}

		key := strings.Replace(chi.URLParam(r, "key"), "-", "/", -1)

		if r.Method == http.MethodGet {
			item := getItem([]byte(key))

			if item == nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			if err := json.NewEncoder(w).Encode(item); err != nil {
				handleError(w, r, err)
			}
		} else if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				handleError(w, r, err)
				return
			}

			readStatus := r.Form.Get("Read")
			if readStatus != "false" {
				handleError(w, r, fmt.Errorf("Unsupported update operation %v", r.Form))
				return
			}

			if err := s.db.SetReadStatus(user, []byte(key), false); err != nil {
				handleError(w, r, err)
				return
			}

			if _, err := io.WriteString(w, "OK"); err != nil {
				log.WithError(err).Error("Failed to write response")
			}
		}
	}
}

// SettingsHandler gets or updates settings for an authenticated user.
func SettingsHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
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
			err := user.SetUsername(newUsername)
			if err != nil {
				handleError(w, r, err)
				return
			}

			if err := s.db.SaveUser(user); err != nil {
				handleError(w, r, err)
				return
			}

			if user.GetUsername() != newUsername {
				// Force logout.
				err := s.cookieHandler.SetCookieUsername(w, "", false)
				if err != nil {
					log.WithError(err).Error("Error while clearing the cookie during logout")
				}
			}

			// Reload user to return updated values.
			user, err = s.db.GetUser(user.GetUsername())
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

		returnUser := &clientUser{Username: user.GetUsername(), Opml: user.Opml, Pagemonitor: user.Pagemonitor}

		if err := json.NewEncoder(w).Encode(returnUser); err != nil {
			handleError(w, r, err)
		}
	}
}

// RefreshHandler refreshes all items for an authenticated user.
func RefreshHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		s.fetcher.Refresh()

		if _, err := io.WriteString(w, "OK"); err != nil {
			log.WithError(err).Error("Failed to write response")
		}
	}
}

// StatusHandler returns the fetch status for all monitored items for an authenticated user.
func StatusHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		feeds, err := user.GetFeeds()
		if err != nil {
			handleError(w, r, err)
			return
		}
		pages, err := user.GetPages()
		if err != nil {
			handleError(w, r, err)
			return
		}

		type itemStatus struct {
			Name        string
			Success     bool
			LastFailure *time.Time `json:",omitempty"`
			LastSuccess *time.Time `json:",omitempty"`
		}
		itemStatuses := make([]itemStatus, len(feeds)+len(pages))

		convertItemStatus := func(name string, status *data.FetchStatus) *itemStatus {
			itemStatus := itemStatus{Name: name}
			if status == nil {
				return &itemStatus
			}
			var emptyTime = time.Time{}
			if status.LastFailure != emptyTime {
				itemStatus.LastFailure = &status.LastFailure
			}
			if status.LastSuccess != emptyTime {
				itemStatus.LastSuccess = &status.LastSuccess
			}
			itemStatus.Success = status.LastSuccess.After(status.LastFailure)
			return &itemStatus
		}

		for i, feed := range feeds {
			fetchStatus, err := s.db.GetFetchStatus(feed.CreateKey())
			if err != nil {
				handleError(w, r, err)
				return
			}
			itemStatuses[i] = *convertItemStatus(feed.Title, fetchStatus)
		}

		for i, page := range pages {
			fetchStatus, err := s.db.GetFetchStatus(page.CreateKey())
			if err != nil {
				handleError(w, r, err)
				return
			}
			itemStatuses[len(feeds)+i] = *convertItemStatus(page.Title, fetchStatus)
		}

		if err := json.NewEncoder(w).Encode(itemStatuses); err != nil {
			handleError(w, r, err)
		}
	}
}
