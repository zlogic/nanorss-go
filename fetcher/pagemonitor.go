package fetcher

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/pmezard/go-difflib/difflib"
	log "github.com/sirupsen/logrus"
	"github.com/zlogic/nanorss-go/data"
)

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
			err = fmt.Errorf("Cannot GET page (status code %v)", resp.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("Cannot GET page %v because of %w", config, err)
		}

		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Cannot read response for page %v because of %w", config, err)
		}
		text, err := html2text.FromString(string(respData))

		var textFiltered, previousTextFiltered string
		if config.Match != "" {
			regex, err := regexp.Compile(config.Match)
			if err != nil {
				return fmt.Errorf("Cannot compile match regex %v because of %w", config, err)
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
			return fmt.Errorf("Cannot create diff for page %v because of %w", config, err)
		}
		page.Delta = diff
		page.Contents = text
		page.Updated = time.Now()
		page.Config = config
		err = fetcher.DB.SetReadStatusForAll(config.CreateKey(), false)
		if err != nil {
			return fmt.Errorf("Cannot mark page as unread %v because of %w", config, err)
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
	ch := make(chan *data.User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			pages, err := user.GetPages()
			if err != nil {
				log.WithError(err).Error("Failed to get pages")
				continue
			}
			countPages := len(pages)
			completed := make(chan int)
			for i, page := range pages {
				go func(config data.UserPagemonitor, index int) {
					fetcher.FetchPage(&config)
					completed <- index
				}(page, i)
			}
			for i := 0; i < countPages; i++ {
				<-completed
			}
		}
		close(done)
	}()
	err := fetcher.DB.ReadAllUsers(ch)
	<-done
	if err != nil {
		return err
	}
	return nil
}
