package search

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kenshin579/ncloud-go"
)

const newsPath = "/search/v1/news"

// 정렬.
const (
	SortSim  = "sim"  // 정확도순 (서버 기본)
	SortDate = "date" // 최신순
)

// 서버가 받는 범위 — 넘으면 SE02/SE03 이므로 미리 막는다.
const (
	MaxDisplay = 100
	MaxStart   = 1000
	// MaxResults 는 NewsAll 이 모으는 최대 건수. start 는 1~1000 이라 display 100 페이지로는
	// 1, 101, …, 901 의 10페이지 = 1,000건까지다(서버 이론상 최대 1,099건은 마지막 페이지를
	// 겹쳐 받아야 해서 쓰지 않는다 — 종목 뉴스에 1,000건이면 충분하다).
	MaxResults = MaxStart
)

// NewsParams 는 뉴스 검색 인자다. 0·빈 값은 서버 기본값(display 10, start 1, sort sim).
type NewsParams struct {
	Query   string // 필수. 따옴표로 묶으면 구절 검색("효성")
	Display int    // 1~100
	Start   int    // 1~1000
	Sort    string // SortSim | SortDate
}

// NewsResult 는 뉴스 검색 응답이다.
type NewsResult struct {
	LastBuildDate string     `json:"lastBuildDate"`
	Total         int        `json:"total"`
	Start         int        `json:"start"`
	Display       int        `json:"display"`
	Items         []NewsItem `json:"items"`
}

// NewsItem 은 기사 하나다. Title·Description 에는 <b> 강조 태그와 HTML 엔티티가 섞여 온다 — Clean* 을 쓴다.
type NewsItem struct {
	Title        string `json:"title"`
	OriginalLink string `json:"originallink"` // 언론사 원문 URL (드물게 빈 값)
	Link         string `json:"link"`         // 네이버 뉴스 URL 이거나, 없으면 원문 URL
	Description  string `json:"description"`
	PubDate      string `json:"pubDate"` // RFC1123Z, 예: "Fri, 02 Oct 2026 09:16:00 +0900"
}

// CleanTitle 은 태그를 지우고 엔티티를 푼 제목이다.
func (n NewsItem) CleanTitle() string { return Clean(n.Title) }

// CleanDescription 은 태그를 지우고 엔티티를 푼 요약이다.
func (n NewsItem) CleanDescription() string { return Clean(n.Description) }

// PubTime 은 PubDate 를 파싱한다(시간대는 응답의 +0900 그대로).
func (n NewsItem) PubTime() (time.Time, error) { return time.Parse(time.RFC1123Z, n.PubDate) }

// URL 은 원문 링크가 있으면 원문, 없으면 Link 를 준다.
func (n NewsItem) URL() string {
	if n.OriginalLink != "" {
		return n.OriginalLink
	}
	return n.Link
}

// tagRe 는 네이버가 강조에 쓰는 <b>·</b> 만 지운다. 일반 "<" 를 태그로 보면 "PER<10배, PBR>1배" 가 망가진다.
var tagRe = regexp.MustCompile(`(?i)</?b>`)

// Clean 은 HTML 태그를 지우고 엔티티(&quot; &amp; 등)를 푼다.
func Clean(s string) string {
	return strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(s, "")))
}

// News 는 뉴스 검색을 한 번 호출한다. 인자가 범위를 벗어나면 서버를 부르지 않고 에러를 낸다.
func (s *Service) News(ctx context.Context, p NewsParams) (*NewsResult, error) {
	q, err := p.query()
	if err != nil {
		return nil, err
	}
	var out NewsResult
	if err := ncloud.GetJSON(ctx, s.c, newsPath, q, &out); err != nil {
		return nil, err
	}
	if out.Items == nil {
		out.Items = []NewsItem{}
	}
	return &out, nil
}

// NewsAll 은 display 100 으로 start 를 넘기며 최대 limit 건(≤ MaxResults)을 모은다.
// 결과가 모자라 짧은 페이지가 오거나 total 에 닿으면 멈춘다. limit ≤ 0 이면 MaxResults.
//
// 도중에 에러가 나면 그때까지 받은 결과(비어 있으면 [])와 에러를 함께 돌려준다.
// SortDate 로 넘기는 사이 새 기사가 들어오면 페이지 경계에서 같은 기사가 두 번 올 수 있다 —
// 필요하면 URL() 로 중복을 거른다.
func (s *Service) NewsAll(ctx context.Context, query, sort string, limit int) ([]NewsItem, error) {
	if limit <= 0 || limit > MaxResults {
		limit = MaxResults
	}
	all := []NewsItem{}
	for start := 1; start <= MaxStart && len(all) < limit; start += MaxDisplay {
		n := min(MaxDisplay, limit-len(all))
		res, err := s.News(ctx, NewsParams{Query: query, Display: n, Start: start, Sort: sort})
		if err != nil {
			return all, err
		}
		all = append(all, res.Items...)
		if len(res.Items) < n || start+len(res.Items) > res.Total {
			break
		}
	}
	return all, nil
}

// ErrInvalidParams 는 인자가 범위를 벗어나 서버를 부르지 않았을 때의 에러다(재시도해도 소용없다).
var ErrInvalidParams = errors.New("search: invalid params")

func (p NewsParams) query() (url.Values, error) {
	if strings.TrimSpace(p.Query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidParams)
	}
	if p.Display < 0 || p.Display > MaxDisplay {
		return nil, fmt.Errorf("%w: display %d out of range 1~%d", ErrInvalidParams, p.Display, MaxDisplay)
	}
	if p.Start < 0 || p.Start > MaxStart {
		return nil, fmt.Errorf("%w: start %d out of range 1~%d", ErrInvalidParams, p.Start, MaxStart)
	}
	if p.Sort != "" && p.Sort != SortSim && p.Sort != SortDate {
		return nil, fmt.Errorf("%w: sort %q must be %q or %q", ErrInvalidParams, p.Sort, SortSim, SortDate)
	}
	q := url.Values{"query": {p.Query}}
	if p.Display > 0 {
		q.Set("display", strconv.Itoa(p.Display))
	}
	if p.Start > 0 {
		q.Set("start", strconv.Itoa(p.Start))
	}
	if p.Sort != "" {
		q.Set("sort", p.Sort)
	}
	return q, nil
}
