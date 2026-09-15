package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultArxivEndpoint = "https://export.arxiv.org/api/query"

var (
	arxivIDPattern = regexp.MustCompile(`^([0-9]{4}\.[0-9]{4,5}|[a-z][a-z0-9.-]*/[0-9]{7})(v[1-9][0-9]*)?$`)
	versionSuffix  = regexp.MustCompile(`v[1-9][0-9]*$`)
)

type arxivClient struct {
	endpoint string
	client   *http.Client
	gate     chan struct{}
	next     time.Time
	interval time.Duration
	backoff  time.Duration
}

func newArxivClient(endpoint string) (*arxivClient, error) {
	if endpoint == "" {
		endpoint = defaultArxivEndpoint
	}
	address, err := url.Parse(endpoint)
	if err != nil || address.Scheme != "https" || address.User != nil ||
		(address.Host != "export.arxiv.org" && address.Host != "arxiv.org") ||
		address.Path != "/api/query" || address.RawQuery != "" || address.Fragment != "" {
		return nil, errors.New("ARXIV_API_ENDPOINT must be https://export.arxiv.org/api/query or https://arxiv.org/api/query")
	}
	return &arxivClient{
		endpoint: endpoint, client: &http.Client{
			Timeout:       20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		gate: make(chan struct{}, 1), interval: 3 * time.Second, backoff: 3 * time.Second,
	}, nil
}

func (c *arxivClient) Search(ctx context.Context, query string) ([]string, error) {
	if err := validateQuery(query); err != nil {
		return nil, err
	}
	if !strings.Contains(query, ":") {
		query = "all:" + query
	}
	papers, err := c.query(ctx, url.Values{
		"search_query": {query}, "start": {"0"}, "max_results": {"3"},
		"sortBy": {"relevance"}, "sortOrder": {"descending"},
	})
	if err != nil {
		return nil, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, paper := range papers {
		if !seen[paper.ID] {
			ids = append(ids, paper.ID)
			seen[paper.ID] = true
		}
		if len(ids) == 3 {
			break
		}
	}
	return ids, nil
}

func (c *arxivClient) Fetch(ctx context.Context, id string) (paper, error) {
	if !arxivIDPattern.MatchString(id) {
		return paper{}, errors.New("invalid arXiv paper ID")
	}
	papers, err := c.query(ctx, url.Values{"id_list": {id}, "max_results": {"1"}})
	if err != nil {
		return paper{}, err
	}
	for _, item := range papers {
		if item.ID == id || (!versionSuffix.MatchString(id) && versionSuffix.ReplaceAllString(item.ID, "") == id) {
			return item, nil
		}
	}
	return paper{}, errors.New("arXiv did not return the requested paper")
}

func (c *arxivClient) query(ctx context.Context, params url.Values) ([]paper, error) {
	for attempt := 0; attempt < 3; attempt++ {
		data, code, retryAfter, err := c.request(ctx, params)
		if err != nil {
			return nil, err
		}
		if code == http.StatusOK {
			return parseFeed(data)
		}
		if code != http.StatusTooManyRequests && code != http.StatusServiceUnavailable {
			return nil, fmt.Errorf("arXiv returned HTTP %d", code)
		}
		if attempt == 2 {
			return nil, fmt.Errorf("arXiv retry budget exhausted after HTTP %d", code)
		}
		delay := c.backoff * time.Duration(1<<attempt)
		if retryAfter > delay {
			delay = retryAfter
		}
		if err := delayContext(ctx, delay); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("arXiv request did not complete")
}

func (c *arxivClient) request(ctx context.Context, params url.Values) ([]byte, int, time.Duration, error) {
	// Serialize network requests per worker and enforce arXiv's minimum spacing.
	select {
	case c.gate <- struct{}{}:
		defer func() { <-c.gate }()
	case <-ctx.Done():
		return nil, 0, 0, ctx.Err()
	}
	if err := delayContext(ctx, time.Until(c.next)); err != nil {
		return nil, 0, 0, err
	}
	c.next = time.Now().Add(c.interval)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, 0, 0, errors.New("invalid arXiv request URL")
	}
	request.Header.Set("Accept", "application/atom+xml")
	request.Header.Set("User-Agent", "DurableTaskGoResearchSample/1.0 (+https://github.com/Azure-Samples/Durable-Task-Scheduler)")
	response, err := c.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, 0, ctx.Err()
		}
		return nil, 0, 0, errors.New("arXiv network request failed")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil {
		return nil, 0, 0, errors.New("failed to read arXiv response")
	}
	if len(data) > 2*1024*1024 {
		return nil, 0, 0, errors.New("arXiv response exceeded 2 MiB")
	}
	return data, response.StatusCode, retryDelay(response.Header.Get("Retry-After"), time.Now()), nil
}

func delayContext(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func retryDelay(value string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return min(time.Duration(min(seconds, 30))*time.Second, 30*time.Second)
	}
	if instant, err := http.ParseTime(value); err == nil {
		return min(max(instant.Sub(now), 0), 30*time.Second)
	}
	return 0
}

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []atomPaper `xml:"http://www.w3.org/2005/Atom entry"`
}

type atomPaper struct {
	ID        string `xml:"http://www.w3.org/2005/Atom id"`
	Title     string `xml:"http://www.w3.org/2005/Atom title"`
	Summary   string `xml:"http://www.w3.org/2005/Atom summary"`
	Published string `xml:"http://www.w3.org/2005/Atom published"`
	Updated   string `xml:"http://www.w3.org/2005/Atom updated"`
	Authors   []struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	} `xml:"http://www.w3.org/2005/Atom author"`
	Categories []struct {
		Term string `xml:"term,attr"`
	} `xml:"http://www.w3.org/2005/Atom category"`
	PrimaryCategory struct {
		Term string `xml:"term,attr"`
	} `xml:"http://arxiv.org/schemas/atom primary_category"`
	Comment    string `xml:"http://arxiv.org/schemas/atom comment"`
	JournalRef string `xml:"http://arxiv.org/schemas/atom journal_ref"`
	DOI        string `xml:"http://arxiv.org/schemas/atom doi"`
}

func parseFeed(data []byte) ([]paper, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, errors.New("arXiv returned an invalid Atom feed")
	}
	if len(feed.Entries) > 100 {
		return nil, errors.New("arXiv feed exceeded its entry budget")
	}
	papers := make([]paper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		address, err := url.Parse(strings.TrimSpace(entry.ID))
		if err != nil || (address.Scheme != "https" && address.Scheme != "http") || address.User != nil ||
			(address.Host != "arxiv.org" && address.Host != "export.arxiv.org") ||
			!strings.HasPrefix(address.Path, "/abs/") || address.RawQuery != "" || address.Fragment != "" {
			return nil, errors.New("arXiv feed contains an invalid paper URL or API error entry")
		}
		id := strings.TrimPrefix(address.Path, "/abs/")
		if !arxivIDPattern.MatchString(id) || strings.TrimSpace(entry.Title) == "" {
			return nil, errors.New("arXiv feed contains an invalid paper ID or title")
		}
		item := paper{
			ID: id, Title: cleanText(entry.Title, 500), Summary: cleanText(entry.Summary, 4000),
			Published: cleanText(entry.Published, 40), Updated: cleanText(entry.Updated, 40),
			Authors: []string{}, Categories: []string{}, PrimaryCategory: cleanText(entry.PrimaryCategory.Term, 64),
			// Construct canonical links; never follow untrusted feed links.
			AbsURL: "https://arxiv.org/abs/" + id, PDFURL: "https://arxiv.org/pdf/" + id,
			Comment: cleanText(entry.Comment, 500), JournalRef: cleanText(entry.JournalRef, 500),
			DOI: cleanText(entry.DOI, 200), Source: "arxiv",
		}
		for _, author := range entry.Authors[:min(len(entry.Authors), 20)] {
			item.Authors = append(item.Authors, cleanText(author.Name, 100))
		}
		for _, category := range entry.Categories[:min(len(entry.Categories), 8)] {
			item.Categories = append(item.Categories, cleanText(category.Term, 64))
		}
		papers = append(papers, item)
	}
	return papers, nil
}

func cleanText(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}
