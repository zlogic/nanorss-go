package fetcher

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/zlogic/nanorss-go/data"
)

var dateFormats = []string{
	time.RFC822,
	time.RFC822Z,
	time.RFC3339,
	time.RFC1123,
	time.RFC1123Z,
	"Mon, _2 Jan 2006 15:04:05 +0300",
}

// sanitizeHTML removes unsafe HTML and replaces relative URLs with absolute ones.
func (fetcher *Fetcher) sanitizeHTML(baseURL string, items []*data.Feeditem) {
	if fetcher.TagsPolicy == nil {
		return
	}

	for _, item := range items {
		item.Contents = fetcher.TagsPolicy.Sanitize(item.Contents)
		fixedURLs, err := fixURLs(baseURL, item.Contents)
		if err != nil {
			log.WithError(err).WithField("itemURL", item.URL).Error("failed to process URLs")
			continue
		}
		item.Contents = fixedURLs
	}
}

// fixURLs replaces relative URLs in itemHTML with absolute URLs (relative to baseURL).
func fixURLs(baseURL, itemHTML string) (string, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(itemHTML))
	buff := bytes.Buffer{}

	itemBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse item base URL: %w", err)
	}

	for {
		if tokenizer.Next() == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				return buff.String(), nil
			}
			return "", err
		}
		token := tokenizer.Token()
		if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
			err := fixURLAttributes(&token, itemBaseURL)
			if err != nil {
				return "", err
			}
		}
		buff.WriteString(token.String())
	}
}

// fixURLAttributes will replace relative URLs with absolute in the token's attributes.
func fixURLAttributes(token *html.Token, baseURL *url.URL) error {
	for i := range token.Attr {
		a := &token.Attr[i]
		isImgSrc := token.Data == "img" && a.Key == "src"
		isAHref := token.Data == "a" && a.Key == "href"
		if isImgSrc || isAHref {
			itemURL, err := url.Parse(a.Val)
			if itemURL.IsAbs() {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to parse %v %v URL: %w", token.Data, a.Key, err)
			}
			a.Val = baseURL.ResolveReference(itemURL).String()
		}
	}
	return nil
}

// ParseFeed parses a downloaded XML feed.
func (fetcher *Fetcher) ParseFeed(feedURL string, reader io.Reader) ([]*data.Feeditem, error) {
	// Atom
	type AtomFeedEntry struct {
		Title     string `xml:"http://www.w3.org/2005/Atom title"`
		GUID      string `xml:"http://www.w3.org/2005/Atom guid"`
		Updated   string `xml:"http://www.w3.org/2005/Atom updated"`
		Published string `xml:"http://www.w3.org/2005/Atom published"`
		Summary   string `xml:"http://www.w3.org/2005/Atom summary"`
		Content   struct {
			Type     string `xml:"type,attr"`
			Value    string `xml:",chardata"`
			InnerXML string `xml:",innerxml"`
		} `xml:"http://www.w3.org/2005/Atom content"`
		Links []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"http://www.w3.org/2005/Atom link"`
	}
	type AtomFeed struct {
		AtomFeedEntries []AtomFeedEntry `xml:"http://www.w3.org/2005/Atom entry"`
	}
	// RSS
	type RSSFeedEntry struct {
		Title       string `xml:"title"`
		GUID        string `xml:"guid"`
		Link        string `xml:"link"`
		Published   string `xml:"pubDate"`
		Content     string `xml:"content encoded"`
		Description string `xml:"description"`
	}
	type RSSFeed struct {
		Published      string         `xml:"channel>pubDate"`
		RSSFeedEntries []RSSFeedEntry `xml:"channel>item"`
	}
	// RDF
	type RDFFeedEntry struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Date        string `xml:"http://purl.org/dc/elements/1.1/ date"`
		Description string `xml:"description"`
	}
	type RDFFeed struct {
		RDFFeedEntries []RDFFeedEntry `xml:"item"`
	}
	// Common feed XML
	type FeedXML struct {
		XMLName  xml.Name
		AtomFeed `xml:"http://www.w3.org/2005/Atom feed"`
		RSSFeed  `xml:"rss"`
		RDFFeed  `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF"`
	}

	//Parse XML
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	var feedXML FeedXML
	if err := decoder.Decode(&feedXML); err != nil {
		return nil, err
	}

	// Truncate current time to apply the same losses as gob.
	timeNowTruncate := func() (time.Time, error) {
		currentTime := time.Now()
		currentTimeBin, err := currentTime.GobEncode()
		if err != nil {
			return time.Time{}, fmt.Errorf("error encoding time: %w", err)
		}
		err = currentTime.GobDecode(currentTimeBin)
		if err != nil {
			return time.Time{}, fmt.Errorf("error decoding time: %w", err)
		}
		return currentTime, nil
	}
	currentTime, err := timeNowTruncate()
	if err != nil {
		return nil, err
	}

	// RSS time parser.
	parseRssTime := func(timeStr string) (time.Time, error) {
		timeStr = strings.TrimSpace(timeStr)
		for _, format := range dateFormats {
			date, err := time.Parse(format, timeStr)
			if err == nil {
				return date, nil
			}
			log.WithField("date", timeStr).WithField("format", format).WithError(err).Debug("Failed to parse time")
		}
		return time.Time{}, fmt.Errorf("failed to parse RSS time")
	}

	// Convert into a common format.
	if feedXML.XMLName.Local == "feed" {
		// Atom.
		items := make([]*data.Feeditem, len(feedXML.AtomFeed.AtomFeedEntries))

		for i, atomItem := range feedXML.AtomFeed.AtomFeedEntries {
			item := &data.Feeditem{
				Title: atomItem.Title,
			}

			item.Date = currentTime
			dateParsed, err := time.Parse(time.RFC3339, atomItem.Updated)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", atomItem.Updated).WithError(err).Debug("Failed to parse updated time")
				dateParsed, err = time.Parse(time.RFC3339, atomItem.Published)
				if err == nil {
					item.Date = dateParsed
				} else {
					log.WithField("date", atomItem.Published).WithError(err).Info("Failed to parse published time")
				}
			}

			if atomItem.Content.Type == "xhtml" {
				item.Contents = strings.TrimSpace(atomItem.Content.InnerXML)
			} else {
				item.Contents = strings.TrimSpace(atomItem.Content.Value)
			}
			if item.Contents == "" {
				item.Contents = strings.TrimSpace(atomItem.Summary)
			}

			item.URL = ""
			for _, link := range atomItem.Links {
				if link.Href == "" {
					continue
				}
				if (link.Type == "text/html" || link.Type == "") && (link.Rel == "alternate" || link.Rel == "") {
					item.URL = link.Href
					break
				}
			}

			item.Key = &data.FeeditemKey{
				FeedURL: feedURL,
			}

			item.Key.GUID = atomItem.GUID
			if item.Key.GUID == "" {
				item.Key.GUID = item.URL
			}

			items[i] = item
		}

		fetcher.sanitizeHTML(feedURL, items)

		return items, nil
	} else if feedXML.XMLName.Local == "rss" {
		// RSS.
		items := make([]*data.Feeditem, len(feedXML.RSSFeed.RSSFeedEntries))

		fallbackDate := currentTime
		dateParsed, err := parseRssTime(feedXML.RSSFeed.Published)
		if err == nil {
			fallbackDate = dateParsed
		} else {
			log.WithField("date", feedXML.RSSFeed.Published).WithError(err).Debug("Failed to parse feed published time")
		}

		for i, rssItem := range feedXML.RSSFeed.RSSFeedEntries {
			item := &data.Feeditem{
				Title: rssItem.Title,
			}

			item.Date = fallbackDate
			dateParsed, err := parseRssTime(rssItem.Published)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", rssItem.Published).WithField("item", rssItem).WithError(err).Info("Failed to parse published time")
			}

			item.Contents = strings.TrimSpace(rssItem.Content)
			if item.Contents == "" {
				item.Contents = strings.TrimSpace(rssItem.Description)
			}

			item.URL = rssItem.Link

			item.Key = &data.FeeditemKey{
				FeedURL: feedURL,
			}

			item.Key.GUID = rssItem.GUID
			if item.Key.GUID == "" {
				item.Key.GUID = item.URL
			}

			items[i] = item
		}

		fetcher.sanitizeHTML(feedURL, items)

		return items, nil
	} else if feedXML.XMLName.Local == "RDF" {
		// RDF.
		items := make([]*data.Feeditem, len(feedXML.RDFFeed.RDFFeedEntries))
		for i, rdfItem := range feedXML.RDFFeed.RDFFeedEntries {
			item := &data.Feeditem{
				Title: rdfItem.Title,
			}

			item.Date = currentTime
			dateParsed, err := time.Parse(time.RFC3339, rdfItem.Date)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", rdfItem.Date).WithError(err).Info("Failed to parse time")
			}

			item.Contents = strings.TrimSpace(rdfItem.Description)

			item.URL = rdfItem.Link

			item.Key = &data.FeeditemKey{
				FeedURL: feedURL,
			}

			item.Key.GUID = item.URL

			items[i] = item
		}

		fetcher.sanitizeHTML(feedURL, items)

		return items, nil
	}

	return nil, fmt.Errorf("unknown feed type %v", feedXML.XMLName)
}
