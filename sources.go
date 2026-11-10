package main

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type TipSource struct {
	Title   string
	URL     string
	Tag     string
	Content string
	ID      string
}

type sourceSlot struct {
	kind string // se | gh | rss
	tag  string
	path string // gh path or rss url
	repo string // gh owner/repo
}

func allSlots() []sourceSlot {
	tags := []string{
		"redis", "elasticsearch", "apache-kafka", "rabbitmq", "docker", "kubernetes",
		"go", "postgresql", "mongodb", "microservices", "distributed-systems",
		"design-patterns", "ci-cd", "observability", "linux", "networking", "system-design",
	}
	var slots []sourceSlot
	for _, t := range tags {
		slots = append(slots, sourceSlot{kind: "se", tag: t})
	}
	gh := []struct{ repo, path, tag string }{
		{"redis/redis-doc", "topics/memory-optimization.md", "redis"},
		{"redis/redis-doc", "topics/pipelining.md", "redis"},
		{"redis/redis-doc", "topics/persistence.md", "redis"},
		{"redis/redis-doc", "topics/lru-cache.md", "redis"},
		{"uber-go/guide", "style.md", "go"},
		{"donnemartin/system-design-primer", "README.md", "system-design"},
	}
	for _, g := range gh {
		slots = append(slots, sourceSlot{kind: "gh", repo: g.repo, path: g.path, tag: g.tag})
	}
	for _, u := range []string{
		"https://blog.cloudflare.com/tag/engineering/rss/",
		"https://martin.kleppmann.com/atom.xml",
		"https://www.confluent.io/rss.xml",
	} {
		slots = append(slots, sourceSlot{kind: "rss", path: u, tag: "engineering"})
	}
	return slots
}

func fetchCandidates(slot sourceSlot, st *State, limit int) ([]TipSource, error) {
	switch slot.kind {
	case "se":
		return fetchStackExchange(slot.tag, st, limit)
	case "gh":
		return fetchGitHubSections(slot.repo, slot.path, slot.tag, st, limit)
	case "rss":
		return fetchRSS(slot.path, slot.tag, st, limit)
	default:
		return nil, fmt.Errorf("unknown source kind %q", slot.kind)
	}
}

var httpClient = &http.Client{Timeout: 25 * time.Second}

func httpGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "techtips/1.0 (+linux systemd)")
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r io.Reader = res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	}
	body, err := io.ReadAll(io.LimitReader(r, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: %s", url, res.Status)
	}
	return body, nil
}

func fetchStackExchange(tag string, st *State, limit int) ([]TipSource, error) {
	u := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions?order=desc&sort=votes&tagged=%s&site=stackoverflow&min=50&filter=withbody&pagesize=30",
		tag,
	)
	body, err := httpGet(u)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []struct {
			QuestionID       int    `json:"question_id"`
			Title            string `json:"title"`
			Link             string `json:"link"`
			Body             string `json:"body"`
			AcceptedAnswerID int    `json:"accepted_answer_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var out []TipSource
	for _, it := range payload.Items {
		id := fmt.Sprintf("se:%d", it.QuestionID)
		if st.Has(id) {
			continue
		}
		content := stripHTML(it.Body)
		if it.AcceptedAnswerID != 0 {
			if ans, err := fetchSEAnswer(it.AcceptedAnswerID); err == nil && ans != "" {
				content += "\n\nAccepted answer:\n" + ans
			}
		}
		out = append(out, TipSource{
			ID: id, Title: html.UnescapeString(it.Title), URL: it.Link,
			Tag: tag, Content: truncate(content, 8000),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func fetchSEAnswer(id int) (string, error) {
	u := fmt.Sprintf("https://api.stackexchange.com/2.3/answers/%d?site=stackoverflow&filter=withbody", id)
	body, err := httpGet(u)
	if err != nil {
		return "", err
	}
	var payload struct {
		Items []struct {
			Body string `json:"body"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if len(payload.Items) == 0 {
		return "", nil
	}
	return stripHTML(payload.Items[0].Body), nil
}

func fetchGitHubSections(repo, path, tag string, st *State, limit int) ([]TipSource, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/master/%s", repo, path)
	body, err := httpGet(rawURL)
	if err != nil {
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s", repo, path)
		body, err = httpGet(rawURL)
		if err != nil {
			return nil, err
		}
	}
	sections := splitMarkdownSections(string(body))
	base := fmt.Sprintf("https://github.com/%s/blob/master/%s", repo, path)
	var out []TipSource
	for _, sec := range sections {
		id := fmt.Sprintf("gh:%s:%s#%s", repo, path, slug(sec.Title))
		if st.Has(id) {
			continue
		}
		if len(strings.TrimSpace(sec.Body)) < 120 {
			continue
		}
		out = append(out, TipSource{
			ID: id, Title: sec.Title, URL: base + "#" + slug(sec.Title),
			Tag: tag, Content: truncate(sec.Body, 8000),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

type mdSection struct {
	Title string
	Body  string
}

var headingRe = regexp.MustCompile(`(?m)^(#{2,3})\s+(.+)$`)

func splitMarkdownSections(md string) []mdSection {
	idxs := headingRe.FindAllStringSubmatchIndex(md, -1)
	if len(idxs) == 0 {
		return []mdSection{{Title: "Document", Body: md}}
	}
	var out []mdSection
	for i, loc := range idxs {
		title := strings.TrimSpace(md[loc[4]:loc[5]])
		start := loc[1]
		end := len(md)
		if i+1 < len(idxs) {
			end = idxs[i+1][0]
		}
		out = append(out, mdSection{Title: title, Body: strings.TrimSpace(md[start:end])})
	}
	return out
}

func fetchRSS(feedURL, tag string, st *State, limit int) ([]TipSource, error) {
	body, err := httpGet(feedURL)
	if err != nil {
		return nil, err
	}
	items := parseFeed(body)
	var out []TipSource
	for _, it := range items {
		id := "rss:" + it.Link
		if it.Link == "" || st.Has(id) {
			continue
		}
		content := it.Summary
		if content == "" {
			content = it.Title
		}
		out = append(out, TipSource{
			ID: id, Title: it.Title, URL: it.Link,
			Tag: tag, Content: truncate(stripHTML(content), 8000),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

type feedItem struct {
	Title   string
	Link    string
	Summary string
}

func parseFeed(data []byte) []feedItem {
	type rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	type atom struct {
		Entries []struct {
			Title   string `xml:"title"`
			Summary string `xml:"summary"`
			Content string `xml:"content"`
			Links   []struct {
				Rel  string `xml:"rel,attr"`
				Href string `xml:"href,attr"`
			} `xml:"link"`
			ID string `xml:"id"`
		} `xml:"entry"`
	}
	var r rss
	if err := xml.Unmarshal(data, &r); err == nil && len(r.Channel.Items) > 0 {
		var out []feedItem
		for _, it := range r.Channel.Items {
			out = append(out, feedItem{Title: it.Title, Link: strings.TrimSpace(it.Link), Summary: it.Description})
		}
		return out
	}
	var a atom
	if err := xml.Unmarshal(data, &a); err == nil {
		var out []feedItem
		for _, e := range a.Entries {
			link := e.ID
			for _, l := range e.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = l.Href
					break
				}
			}
			sum := e.Summary
			if sum == "" {
				sum = e.Content
			}
			out = append(out, feedItem{Title: e.Title, Link: link, Summary: sum})
		}
		return out
	}
	return nil
}

var (
	tagRe   = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe = regexp.MustCompile(`\s+`)
)

func stripHTML(s string) string {
	s = html.UnescapeString(tagRe.ReplaceAllString(s, " "))
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9آ-ی]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
