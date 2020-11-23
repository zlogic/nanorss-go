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
)

func handleBadCredentials(w http.ResponseWriter, r *http.Request, err error) {
	log.WithError(err).Error("Bad credentials for user")
	http.Error(w, "Bad credentials", http.StatusUnauthorized)
}

func validateUserForAPI(w http.ResponseWriter, r *http.Request, s *Services) *data.User {
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
			handleBadCredentials(w, r, fmt.Errorf("User %v does not exist", username))
			return
		}
		err = user.ValidatePassword(password)
		if err != nil {
			handleBadCredentials(w, r, fmt.Errorf("Invalid password for user %v (%w)", username, err))
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
			log.WithError(err).Error("Failed to write response")
		}
	}
}

// FeedHandler returns all feed (and page monitor) items for an authenticated user.
func FeedHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := validateUserForAPI(w, r, s)
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

// FeedItemHandler returns a feed (or page monitor) item for an authenticated user.
func FeedItemHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// There are no secrets, but still better check we have a valid user
		user := validateUserForAPI(w, r, s)
		if user == nil {
			return
		}

		itemType := chi.URLParam(r, "type")
		key := strings.Replace(chi.URLParam(r, "key"), "-", "/", -1)

		type clientFeedItem struct {
			URL           string
			Contents      string
			Date          time.Time
			Plaintext     bool
			MarkUnreadURL string
		}
		getItem := func() *clientFeedItem {
			if itemType == "feeditem" {
				feeditemKey, err := decodeFeeditemKeyFromURL(key)
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
				err = s.db.SetFeeditemReadStatus(user, feedItem.Key, true)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to set read status for feed item")
				}
				return &clientFeedItem{
					Contents:      feedItem.Contents,
					Date:          feedItem.Date,
					URL:           feedItem.URL,
					Plaintext:     false,
					MarkUnreadURL: "api/items/feeditem/" + escapeFeeditemKeyForURL(feeditemKey),
				}
			} else if itemType == "page" {
				pagemonitorKey, err := decodePagemonitorKeyFromURL(key)
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
				err = s.db.SetPageReadStatus(user, pagemonitorPage.Config, true)
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to set read status for page")
				}
				// Bootstrap automatically handles line endings
				return &clientFeedItem{
					Contents:      pagemonitorPage.Delta,
					Date:          pagemonitorPage.Updated,
					URL:           pagemonitorKey.URL,
					Plaintext:     true,
					MarkUnreadURL: "api/items/page/" + escapePagemonitorKeyForURL(pagemonitorKey),
				}
			}
			log.WithField("type", itemType).WithField("key", key).Error("Unknown item key format")
			return nil
		}

		markUnread := func() error {
			if itemType == "feeditem" {
				feeditemKey, err := decodeFeeditemKeyFromURL(key)
				if err != nil {
					return err
				}
				err = s.db.SetFeeditemReadStatus(user, feeditemKey, false)
				if err != nil {
					return err
				}
				return nil
			} else if itemType == "page" {
				pagemonitorKey, err := decodePagemonitorKeyFromURL(key)
				if err != nil {
					return err
				}
				err = s.db.SetPageReadStatus(user, pagemonitorKey, false)
				if err != nil {
					return err
				}
				return nil
			}
			return fmt.Errorf("unknown key format: %v type: %v", key, itemType)
		}

		if r.Method == http.MethodGet {
			item := getItem()

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

			if err := markUnread(); err != nil {
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
			err := user.SetUsername(newUsername)
			if err != nil {
				handleError(w, r, err)
				return
			}

			if err := s.db.SaveUser(user); err != nil {
				handleError(w, r, err)
				return
			}

			if username != newUsername {
				// Force logout
				cookie := s.cookieHandler.NewCookie()
				http.SetCookie(w, cookie)
			}

			//Reload user to return updated values
			username = user.GetUsername()
			user, err = s.db.GetUser(username)
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

// RefreshHandler refreshes all items for an authenticated user.
func RefreshHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := validateUserForAPI(w, r, s)
		if user == nil {
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
		user := validateUserForAPI(w, r, s)
		if user == nil {
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
			Name             string
			Success          bool
			LastSuccess      *time.Time `json:",omitempty"`
			LastFailure      *time.Time `json:",omitempty"`
			LastFailureError string     `json:",omitempty"`
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
				itemStatus.LastFailureError = status.LastFailureError
			}
			if status.LastSuccess != emptyTime {
				itemStatus.LastSuccess = &status.LastSuccess
			}
			itemStatus.Success = status.LastSuccess.After(status.LastFailure)
			return &itemStatus
		}

		for i, feed := range feeds {
			fetchStatus, err := s.db.GetFeedFetchStatus(feed.URL)
			if err != nil {
				handleError(w, r, err)
				return
			}
			itemStatuses[i] = *convertItemStatus(feed.Title, fetchStatus)
		}

		for i, page := range pages {
			fetchStatus, err := s.db.GetPageFetchStatus(&page)
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
