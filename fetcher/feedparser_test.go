package fetcher

import (
	"bytes"
	"testing"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"

	"github.com/zlogic/nanorss-go/data"
)

const parseAtomFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<entry>
<title>Title 1</title>
<link rel="alternate" type="text/html" href="http://site1/link1-good"/>
<link href="http://site1/link1" />
<link rel="edit" href="http://site1/link1"/>
<guid>Item@1</guid>
<updated>2003-12-13T18:30:02Z</updated>
<published>2002-12-13T18:30:02Z</published>
<summary>Summary 1</summary>
<content type="xhtml">
<div xmlns="http://www.w3.org/1999/xhtml">
<p>Content 1</p>
</div>
</content>
</entry>
<entry>
<title>Title 2</title>
<link rel="alternate" href="http://site1/link2-good"/>
<link href="http://site1/link2" />
<published>2003-12-14T18:30:02Z</published>
<summary>Summary 1</summary>
<content type="xhtml">
<div xmlns="http://www.w3.org/1999/xhtml">
<p>Content 2</p>
</div>
</content>
</entry>
<entry>
<title>Title 3</title>
<link type="text/html" href="http://site1/link3-good"/>
<link href="http://site1/link3" />
<guid>Item@3</guid>
<summary>Summary 1</summary>
<content type="html">
&lt;div xmlns=&#34;http://www.w3.org/1999/xhtml&#34;&gt;
&lt;p&gt;Content 3&lt;/p&gt;
&lt;/div&gt;
</content>
</entry>
</feed>`

const parseRssFeed = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
<channel>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<item>
<title>Title 1</title>
<link>http://site1/link1</link>
<description>Text 1</description>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<guid>Item@1</guid>
</item>
<item>
<title>Title 2</title>
<link>http://site1/link2</link>
<description>Text 2</description>
<guid>Item@2</guid>
</item>
<item>
<title>Title 3</title>
<link>http://site1/link3</link>
<description>Text 3</description>
<pubDate>Tue, 07 Jun 2016 13:19:00 GMT</pubDate>
</item>
<item>
<title>Title 4</title>
<link>http://site1/link4</link>
<description>Text 4</description>
<content:encoded>Content 4</content:encoded>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<guid>Item@4</guid>
</item>
</channel>
</rss>`

const parseRdfFeed = `<?xml version="1.0" encoding="utf-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/">
<item>
<dc:date>2013-09-26T22:36:20+01:00</dc:date>
<title>Title 1</title>
<link>http://site1/link1</link>
<description>Description 1</description>
</item>
</rdf:RDF>`

const sanitizeAtomFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<entry>
<content type="xhtml">
<div xmlns="http://www.w3.org/1999/xhtml">
<p>Content 1</p>
<iframe src="http://hackersite.ru"></iframe>
<p>More content</p>
</div>
</content>
</entry>
<entry>
<content type="html">
&lt;div xmlns=&#34;http://www.w3.org/1999/xhtml&#34;&gt;
&lt;p&gt;Content 2&lt;/p&gt;
&lt;iframe src=&#34;http://hackersite.ru&#34;&gt;&lt;/iframe&gt;
&lt;p&gt;More content&lt;/p&gt;
&lt;/div&gt;
</content>
</entry>
</feed>`

const sanitizeRssFeed = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
<channel>
<item>
<description>&lt;p&gt;Content 1&lt;/p&gt;
&lt;iframe src=&#34;http://hackersite.ru&#34;&gt;&lt;/iframe&gt;
&lt;p&gt;More content&lt;/p&gt;</description>
</item>
<item>
<description>Text 2</description>
<content:encoded>&lt;p&gt;Content 2&lt;/p&gt;
&lt;iframe src=&#34;http://hackersite.ru&#34;&gt;&lt;/iframe&gt;
&lt;p&gt;More content&lt;/p&gt;</content:encoded>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<guid>Item@4</guid>
</item>
</channel>
</rss>`

const sanitizeRdfFeed = `<?xml version="1.0" encoding="utf-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/">
<item>
<description>&lt;p&gt;Content&lt;/p&gt;
&lt;iframe src=&#34;http://hackersite.ru&#34;&gt;&lt;/iframe&gt;
&lt;p&gt;More content&lt;/p&gt;</description>
</item>
</rdf:RDF>`

const sanitizeRssFeedURLs = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
<channel>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<item>
<title>Title 1</title>
<link>http://site1/link1</link>
<description>&lt;a href="/relative"&gt;Relative link 1&lt;/a&gt;
&lt;a href="relative2"&gt;Relative link 2&lt;/a&gt;
&lt;a href="https://domain1.example.com/absolute"&gt;Absolute link 1&lt;/a&gt;
&lt;img src="/img/avatar1.png" width="16" height="16"&gt;
&lt;img src="/img/avatar2.png" width="16" height="16" /&gt;</description>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<guid>Item@1</guid>
</item>
</channel>
</rss>`

const sanitizedRSSFeedURLs = `<a href="https://www.example.com/relative" rel="nofollow">Relative link 1</a>
<a href="https://www.example.com/relative2" rel="nofollow">Relative link 2</a>
<a href="https://domain1.example.com/absolute" rel="nofollow">Absolute link 1</a>
<img src="https://www.example.com/img/avatar1.png" width="16" height="16">
<img src="https://www.example.com/img/avatar2.png" width="16" height="16"/>`

func TestParseAtom(t *testing.T) {
	fetcher := Fetcher{}
	beforeParse := time.Now()

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(parseAtomFeed)))
	assert.NoError(t, err)

	assert.Len(t, items, 3)
	assertTimeBetween(t, beforeParse, time.Now(), items[2].Date)
	items[2].Date = time.Time{}

	assert.Equal(t, []*data.Feeditem{
		{
			Title:    "Title 1",
			URL:      "http://site1/link1-good",
			Date:     time.Date(2003, time.December, 13, 18, 30, 2, 0, time.UTC),
			Contents: "<div xmlns=\"http://www.w3.org/1999/xhtml\">\n<p>Content 1</p>\n</div>",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "Item@1",
			},
		},
		{
			Title:    "Title 2",
			URL:      "http://site1/link2-good",
			Date:     time.Date(2003, time.December, 14, 18, 30, 2, 0, time.UTC),
			Contents: "<div xmlns=\"http://www.w3.org/1999/xhtml\">\n<p>Content 2</p>\n</div>",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "http://site1/link2-good",
			},
		},
		{
			Title:    "Title 3",
			URL:      "http://site1/link3-good",
			Contents: "<div xmlns=\"http://www.w3.org/1999/xhtml\">\n<p>Content 3</p>\n</div>",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "Item@3",
			},
		},
	}, items)
}

func TestParseRss(t *testing.T) {
	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(parseRssFeed)))
	assert.NoError(t, err)

	for i := range items {
		items[i].Date = items[i].Date.In(gmt)
	}

	assert.Equal(t, []*data.Feeditem{
		{
			Title:    "Title 1",
			URL:      "http://site1/link1",
			Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, gmt),
			Contents: "Text 1",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "Item@1",
			},
		},
		{
			Title:    "Title 2",
			URL:      "http://site1/link2",
			Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, gmt),
			Contents: "Text 2",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "Item@2",
			},
		},
		{
			Title:    "Title 3",
			URL:      "http://site1/link3",
			Date:     time.Date(2016, time.June, 7, 13, 19, 0, 0, gmt),
			Contents: "Text 3",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "http://site1/link3",
			},
		},
		{
			Title:    "Title 4",
			URL:      "http://site1/link4",
			Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, gmt),
			Contents: "Content 4",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "Item@4",
			},
		},
	}, items)
}

func TestParseRdf(t *testing.T) {
	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(parseRdfFeed)))
	assert.NoError(t, err)

	assert.Len(t, items, 1)
	assert.Equal(t, "2013-09-26 22:36:20 +0100 +0100", items[0].Date.String())
	items[0].Date = time.Time{}

	assert.Equal(t, []*data.Feeditem{
		{
			Title:    "Title 1",
			URL:      "http://site1/link1",
			Contents: "Description 1",
			Key: &data.FeeditemKey{
				FeedURL: "http://sites-site1.com",
				GUID:    "http://site1/link1",
			},
		},
	}, items)
}

func TestSanitizeAtom(t *testing.T) {
	fetcher := Fetcher{TagsPolicy: bluemonday.UGCPolicy()}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(sanitizeAtomFeed)))
	assert.NoError(t, err)

	assert.Len(t, items, 2)

	assert.Equal(t, "<div>\n<p>Content 1</p>\n\n<p>More content</p>\n</div>", items[0].Contents)
	assert.Equal(t, "<div>\n<p>Content 2</p>\n\n<p>More content</p>\n</div>", items[1].Contents)
}

func TestSanitizeRss(t *testing.T) {
	fetcher := Fetcher{TagsPolicy: bluemonday.UGCPolicy()}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(sanitizeRssFeed)))
	assert.NoError(t, err)

	assert.Len(t, items, 2)

	assert.Equal(t, "<p>Content 1</p>\n\n<p>More content</p>", items[0].Contents)
	assert.Equal(t, "<p>Content 2</p>\n\n<p>More content</p>", items[1].Contents)
}

func TestSanitizeRdf(t *testing.T) {
	fetcher := Fetcher{TagsPolicy: bluemonday.UGCPolicy()}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(sanitizeRdfFeed)))
	assert.NoError(t, err)

	assert.Len(t, items, 1)

	assert.Equal(t, "<p>Content</p>\n\n<p>More content</p>", items[0].Contents)
}

func TestSanitizeRssURLs(t *testing.T) {
	fetcher := Fetcher{TagsPolicy: bluemonday.UGCPolicy()}

	items, err := fetcher.ParseFeed("https://www.example.com/feed", bytes.NewBuffer([]byte(sanitizeRssFeedURLs)))
	assert.NoError(t, err)

	assert.Len(t, items, 1)

	assert.Equal(t, sanitizedRSSFeedURLs, items[0].Contents)
}

func TestParseInvalid(t *testing.T) {
	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte("<xml")))
	assert.Error(t, err)
	assert.Empty(t, items)
}

func TestParseAtomEmpty(t *testing.T) {
	emptyAtom := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>`

	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(emptyAtom)))
	assert.NoError(t, err)

	assert.Equal(t, []*data.Feeditem{}, items)
}

func TestParseRssEmpty(t *testing.T) {
	emptyRss := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
</rss>`

	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(emptyRss)))
	assert.NoError(t, err)

	assert.Equal(t, []*data.Feeditem{}, items)
}

func TestParseRdfEmpty(t *testing.T) {
	emptyRdf := `<?xml version="1.0" encoding="utf-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:dc="http://purl.org/dc/elements/1.1/">
</rdf:RDF>`

	fetcher := Fetcher{}

	items, err := fetcher.ParseFeed("http://sites-site1.com", bytes.NewBuffer([]byte(emptyRdf)))
	assert.NoError(t, err)

	assert.Equal(t, []*data.Feeditem{}, items)
}
