package fetcher

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/zlogic/nanorss-go/data"
)

func (fetcher *Fetcher) getPreviousResult(config *data.UserPagemonitor) *data.PagemonitorPage {
	page, err := fetcher.DB.GetPage(config)
	if err != nil {
		log.Printf("Failed to fetch previous result %v for %v", err, config)
	}
	if page == nil {
		return &data.PagemonitorPage{}
	}
	return page
}

func (fetcher *Fetcher) FetchPage(config *data.UserPagemonitor) error {
	page := fetcher.getPreviousResult(config)

	resp, err := fetcher.Client.Get(config.URL)
	if err != nil {
		defer resp.Body.Close()
	}

	if err == nil && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Cannot GET page (status code %v)", resp.StatusCode)
	}
	if err != nil {
		return errors.Wrapf(err, "Cannot GET page %v", config)
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Cannot read response for page %v", config)
	}
	text, err := html2text.FromString(string(respData))

	var textFiltered, previousTextFiltered string
	if config.Match != "" {
		regex, err := regexp.Compile(config.Match)
		if err != nil {
			return errors.Wrapf(err, "Cannot compile match regex %v", config)
		}
		textFiltered = regex.ReplaceAllString(text, config.Replace)
		previousTextFiltered = regex.ReplaceAllString(page.Contents, config.Replace)
	} else {
		textFiltered = text
		previousTextFiltered = page.Contents
	}

	if previousTextFiltered == textFiltered {
		// No changes
		return nil
	}

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:       difflib.SplitLines(previousTextFiltered),
		B:       difflib.SplitLines(textFiltered),
		Context: 3,
	})
	if err != nil {
		return errors.Wrapf(err, "Cannot create diff for page %v", config)
	}
	page.Delta = diff
	page.Contents = text
	page.Updated = time.Now()
	return fetcher.DB.SavePage(config, page)
}

func (fetcher *Fetcher) FetchAllPages() error {
	failed := false
	ch := make(chan *data.User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			pages, err := user.GetPages()
			if err != nil {
				log.Printf("Failed to get pages for user %v %v", user, err)
				failed = true
				continue
			}
			countPages := len(pages)
			completed := make(chan int)
			for i, page := range pages {
				go func(config data.UserPagemonitor, index int) {
					err := fetcher.FetchPage(&config)
					if err != nil {
						log.Printf("Failed to get page %v %v", config, err)
						failed = true
					}
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
	if failed {
		return fmt.Errorf("At least one page failed to fetch properly")
	}
	return nil
}
