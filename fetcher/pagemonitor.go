package fetcher

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/zlogic/nanorss-go/data"
)

// getPreviousResult returns the previous value for the page (or an empty PagemonitorPage if no value exists).
func (fetcher *Fetcher) getPreviousResult(config *data.UserPagemonitor) *data.PagemonitorPage {
	page, err := fetcher.DB.GetPage(config)
	if err != nil {
		log.WithField("page", config).WithError(err).Error("Failed to fetch previous result")
	}
	if page == nil {
		return &data.PagemonitorPage{}
	}
	return page
}

// FetchPage fetches a page and performs a diff based on config.
// On success, it's saved into the database.
func (fetcher *Fetcher) FetchPage(config *data.UserPagemonitor) error {
	err := func() error {
		page := fetcher.getPreviousResult(config)

		resp, err := fetcher.Client.Get(config.URL)
		if err == nil {
			defer resp.Body.Close()
		}

		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("cannot GET page (status code %v)", resp.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("cannot GET page %v: %w", config, err)
		}

		text, err := convertHTMLtoText(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot convert HTML to text %v: %w", config, err)
		}

		var textFiltered, previousTextFiltered string
		if config.Match != "" {
			regex, err := regexp.Compile(config.Match)
			if err != nil {
				return fmt.Errorf("cannot compile match regex %v: %w", config, err)
			}
			textFiltered = regex.ReplaceAllString(text, config.Replace)
			previousTextFiltered = regex.ReplaceAllString(page.Contents, config.Replace)
		} else {
			textFiltered = text
			previousTextFiltered = page.Contents
		}

		if previousTextFiltered == textFiltered {
			// Save if nothing changed to update last seen time
			return fetcher.DB.SavePage(page)
		}

		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:       difflib.SplitLines(previousTextFiltered),
			B:       difflib.SplitLines(textFiltered),
			Context: 3,
		})
		if err != nil {
			return fmt.Errorf("cannot create diff for page %v: %w", config, err)
		}
		page.Delta = diff
		page.Contents = text
		page.Updated = time.Now()
		page.Config = config
		err = fetcher.DB.SetReadStatusForAll(config.CreateKey(), false)
		if err != nil {
			return fmt.Errorf("cannot mark page %v as unread: %w", config, err)
		}

		log.WithField("value", page).WithField("page", config).WithField("delta", page.Delta).Debug("Page has changed")

		return fetcher.DB.SavePage(page)
	}()

	fetchStatus := &data.FetchStatus{}
	if err != nil {
		log.WithField("page", config).WithError(err).Error("Failed to get page")
		fetchStatus.LastFailure = time.Now()
	} else {
		fetchStatus.LastSuccess = time.Now()
	}

	fetchStatusKey := config.CreateKey()
	if err := fetcher.DB.SetFetchStatus(fetchStatusKey, fetchStatus); err != nil {
		log.WithField("page", config).WithError(err).Error("Failed to save fetch status for page")
	}
	return err
}

// FetchAllPages calls FetchPage for all pages for all users.
func (fetcher *Fetcher) FetchAllPages() error {
	usernames, err := fetcher.DB.GetUsers()
	if err != nil {
		log.WithError(err).Error("Failed to get list of users")
		return err
	}
	for _, username := range usernames {
		user, err := fetcher.DB.GetUser(username)
		if err != nil {
			log.WithField("username", username).WithError(err).Error("Failed to get user")
			return err
		}
		pages, err := user.GetPages()
		if err != nil {
			log.WithError(err).Error("Failed to get pages")
			continue
		}
		countPages := len(pages)
		completed := make(chan int)
		for i, page := range pages {
			go func(config data.UserPagemonitor, index int) {
				// TODO: skip this page if it was already fetched this round.
				fetcher.FetchPage(&config)
				completed <- index
			}(page, i)
		}
		for i := 0; i < countPages; i++ {
			<-completed
		}
	}
	return nil
}

func convertHTMLtoText(r io.Reader) (string, error) {
	tokenizer := html.NewTokenizer(r)
	buff := bytes.Buffer{}

	for {
		if tokenizer.Next() == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				return buff.String(), nil
			}
			return "", err
		}
		token := tokenizer.Token()
		if token.Type == html.TextToken {
			text := strings.TrimSpace(html.UnescapeString(token.Data))
			if text == "" {
				continue
			}
			if buff.Len() > 0 {
				buff.WriteString("\n")
			}
			buff.WriteString(text)
		}
	}
}
