package fetcher

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zlogic/nanorss-go/data"
	"golang.org/x/net/html/charset"
)

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
		Date        string `xml:"dc date"`
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

	// Convert into a common format
	if feedXML.XMLName.Local == "feed" {
		//Atom
		items := make([]*data.Feeditem, len(feedXML.AtomFeed.AtomFeedEntries))

		for i, atomItem := range feedXML.AtomFeed.AtomFeedEntries {
			item := &data.Feeditem{
				Title: atomItem.Title,
			}

			item.Date = time.Now()
			dateParsed, err := time.Parse(time.RFC3339, atomItem.Updated)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", atomItem.Updated).WithError(err).Debug("Failed to parse updated time")
				dateParsed, err = time.Parse(time.RFC3339, atomItem.Published)
				if err == nil {
					item.Date = dateParsed
				} else {
					log.WithField("date", atomItem.Published).WithError(err).Debug("Failed to parse published time")
				}
			}

			if atomItem.Content.Type == "xhtml" {
				item.Contents = strings.TrimSpace(atomItem.Content.InnerXML)
			} else {
				item.Contents = strings.TrimSpace(atomItem.Content.Value)
			}
			if item.Contents == "" {
				item.Contents = atomItem.Summary
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
		return items, nil
	} else if feedXML.XMLName.Local == "rss" {
		// RSS
		items := make([]*data.Feeditem, len(feedXML.RSSFeed.RSSFeedEntries))

		fallbackDate := time.Now()
		dateParsed, err := time.Parse(time.RFC1123, feedXML.RSSFeed.Published)
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
			dateParsed, err := time.Parse(time.RFC1123, rssItem.Published)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", rssItem.Published).WithError(err).Debug("Failed to parse published time")
			}

			item.Contents = rssItem.Content
			if item.Contents == "" {
				item.Contents = rssItem.Description
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
		return items, nil
	} else if feedXML.XMLName.Local == "RDF" {
		// RDF
		items := make([]*data.Feeditem, len(feedXML.RDFFeed.RDFFeedEntries))
		for i, rdfItem := range feedXML.RDFFeed.RDFFeedEntries {
			item := &data.Feeditem{
				Title: rdfItem.Title,
			}

			item.Date = time.Now()
			dateParsed, err := time.Parse(time.RFC3339, rdfItem.Date)
			if err == nil {
				item.Date = dateParsed
			} else {
				log.WithField("date", rdfItem.Date).WithError(err).Debug("Failed to parse time")
			}

			item.Contents = rdfItem.Description

			item.URL = rdfItem.Link

			item.Key = &data.FeeditemKey{
				FeedURL: feedURL,
			}

			item.Key.GUID = item.URL

			items[i] = item
		}
		return items, nil
	}

	return nil, fmt.Errorf("Unknown feed type %v", feedXML.XMLName)
}
