package server

import (
	"log"
	"net/http"
	"path"
	"strings"
	"text/template"

	"github.com/gorilla/mux"

	"github.com/zlogic/nanorss-go/data"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Error while handling request %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func validateUser(w http.ResponseWriter, r *http.Request, s *services) string {
	username := s.cookieHandler.GetUsername(w, r)
	if username == "" {
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
	return username
}

func loadTemplate(pageName string) (*template.Template, error) {
	return template.ParseFiles(path.Join("templates", "layout.html"), path.Join("templates", "pages", pageName+".html"))
}

type viewData struct {
	User     *data.User
	Username string
	Name     string
}

func RootHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := s.cookieHandler.GetUsername(w, r)
		var url string
		if username == "" {
			url = "login"
		} else {
			url = "feed"
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

func LogoutHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := s.cookieHandler.newCookie()
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
}

func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join("static", "favicon.ico"))
}

func HtmlLoginHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := s.cookieHandler.GetUsername(w, r)
		if username != "" {
			http.Redirect(w, r, "feed", http.StatusSeeOther)
			return
		}
		t, err := loadTemplate("login")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{})
	}
}

func HtmlFeedHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := validateUser(w, r, s)
		if username == "" {
			return
		}
		userService := s.db.NewUserService(username)
		user, err := userService.Get()
		if err != nil {
			handleError(w, r, err)
			return
		}

		t, err := loadTemplate("feed")
		if err != nil {
			handleError(w, r, err)
			return
		}
		type feedViewData struct {
			viewData
			Items []*Item
		}
		items, err := GetAllItems(userService)
		if items == nil {
			items = make([]*Item, 0)
		}
		/*
			if err != nil {
				handleError(w, r, err)
				return
			}
		*/
		t.ExecuteTemplate(w, "layout", &feedViewData{
			viewData: viewData{User: user, Username: username, Name: mux.CurrentRoute(r).GetName()},
			Items:    items,
		})
	}
}

func HtmlSettingsHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := validateUser(w, r, s)
		if username == "" {
			return
		}
		userService := s.db.NewUserService(username)
		user, err := userService.Get()
		if err != nil {
			handleError(w, r, err)
			return
		}

		t, err := loadTemplate("settings")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: username, Name: mux.CurrentRoute(r).GetName()})
	}
}

func HtmlItemHandler(s *services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// There are no secrets, but still better check we have a valid user
		username := validateUser(w, r, s)
		if username == "" {
			return
		}

		getItem := func(key string) *data.Feeditem {
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
				return feedItem
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
				// Bootstrap automatically handles line endings
				contents := "<pre>" + pagemonitorPage.Delta + "</pre>"
				feedItem := &data.Feeditem{
					Contents: contents,
					Date:     pagemonitorPage.Updated,
					URL:      pagemonitorKey.URL,
				}
				return feedItem
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

		t, err := template.ParseFiles(path.Join("templates", "pages", "item.html"))
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", item)
	}
}
