package search

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/kenshin579/ncloud-go"
)

func newService(t *testing.T, h http.HandlerFunc) *Service {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := ncloud.NewClient("id", "secret", ncloud.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	return New(c)
}

func fileHandler(t *testing.T, name string, got *url.URL) http.HandlerFunc {
	b, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if got != nil {
			*got = *r.URL
		}
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		w.Write(b)
	}
}

func TestNews(t *testing.T) {
	var u url.URL
	s := newService(t, fileHandler(t, "news_ok.json", &u))
	res, err := s.News(context.Background(), NewsParams{Query: "네패스", Display: 3, Sort: SortDate})
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != newsPath || u.Query().Get("query") != "네패스" || u.Query().Get("display") != "3" || u.Query().Get("sort") != "date" {
		t.Errorf("request %s?%s", u.Path, u.RawQuery)
	}
	if _, ok := u.Query()["start"]; ok {
		t.Error("zero Start must not be sent")
	}
	if res.Total == 0 || len(res.Items) != 3 {
		t.Fatalf("res total=%d items=%d", res.Total, len(res.Items))
	}
	it := res.Items[0]
	if it.Title == "" || it.URL() == "" || it.PubDate == "" {
		t.Errorf("item %+v", it)
	}
	if _, err := it.PubTime(); err != nil {
		t.Errorf("PubTime: %v", err)
	}
}

func TestNewsEmpty(t *testing.T) {
	s := newService(t, fileHandler(t, "news_empty.json", nil))
	res, err := s.News(context.Background(), NewsParams{Query: "zz"})
	if err != nil || res.Total != 0 || res.Items == nil || len(res.Items) != 0 {
		t.Fatalf("res=%+v err=%v; want empty non-nil items", res, err)
	}
}

func TestNewsValidation(t *testing.T) {
	called := false
	s := newService(t, func(http.ResponseWriter, *http.Request) { called = true })
	bad := []NewsParams{
		{Query: " "},
		{Query: "a", Display: 101},
		{Query: "a", Display: -1},
		{Query: "a", Start: 1001},
		{Query: "a", Sort: "xx"},
	}
	for _, p := range bad {
		if _, err := s.News(context.Background(), p); err == nil {
			t.Errorf("%+v: want error", p)
		}
	}
	if called {
		t.Fatal("invalid params must not reach the server")
	}
}

func TestNewsAllPages(t *testing.T) {
	// total 250 인 가짜 서버: start/display 대로 잘라 준다.
	var starts []int
	s := newService(t, func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconv.Atoi(r.URL.Query().Get("start"))
		display, _ := strconv.Atoi(r.URL.Query().Get("display"))
		starts = append(starts, start)
		n := min(display, max(0, 250-start+1))
		items := "["
		for i := 0; i < n; i++ {
			if i > 0 {
				items += ","
			}
			items += `{"title":"t` + strconv.Itoa(start+i) + `","originallink":"","link":"l","description":"","pubDate":""}`
		}
		w.Write([]byte(`{"total":250,"start":` + strconv.Itoa(start) + `,"display":` + strconv.Itoa(n) + `,"items":` + items + `]}`))
	})
	all, err := s.NewsAll(context.Background(), "a", SortDate, 0)
	if err != nil || len(all) != 250 {
		t.Fatalf("len=%d err=%v", len(all), err)
	}
	if len(starts) != 3 || starts[0] != 1 || starts[1] != 101 || starts[2] != 201 {
		t.Errorf("starts=%v", starts)
	}
	starts = nil
	some, _ := s.NewsAll(context.Background(), "a", SortDate, 150)
	if len(some) != 150 || len(starts) != 2 {
		t.Errorf("max 150: len=%d starts=%v", len(some), starts)
	}
}

func TestClean(t *testing.T) {
	cases := map[string]string{
		"<b>네패스</b>, 장학금 &quot;쾌척&quot;": `네패스, 장학금 "쾌척"`,
		"S&amp;P 500": "S&P 500",
		"  공백  ":      "공백",
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q; want %q", in, got, want)
		}
	}
	it := NewsItem{OriginalLink: "", Link: "https://n.news.naver.com/x"}
	if it.URL() != "https://n.news.naver.com/x" {
		t.Error("URL falls back to Link")
	}
}

func TestNewsAllCapIs1000(t *testing.T) {
	var calls int
	s := newService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		start, _ := strconv.Atoi(r.URL.Query().Get("start"))
		display, _ := strconv.Atoi(r.URL.Query().Get("display"))
		if start > MaxStart {
			t.Errorf("start %d exceeds server max", start)
		}
		items := "["
		for i := 0; i < display; i++ {
			if i > 0 {
				items += ","
			}
			items += `{"title":"t","originallink":"","link":"l","description":"","pubDate":""}`
		}
		w.Write([]byte(`{"total":50000,"start":` + strconv.Itoa(start) + `,"display":` + strconv.Itoa(display) + `,"items":` + items + `]}`))
	})
	for _, limit := range []int{0, MaxResults, MaxResults + 99} {
		calls = 0
		all, err := s.NewsAll(context.Background(), "a", SortDate, limit)
		if err != nil || len(all) != MaxResults || calls != 10 {
			t.Errorf("limit %d: len=%d calls=%d err=%v; want %d in 10 calls", limit, len(all), calls, err, MaxResults)
		}
	}
}

func TestNewsValidationIsSentinel(t *testing.T) {
	s := newService(t, func(http.ResponseWriter, *http.Request) {})
	_, err := s.News(context.Background(), NewsParams{Query: "a", Display: 101})
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams, got %v", err)
	}
}

func TestCleanKeepsBareAngleBrackets(t *testing.T) {
	cases := map[string]string{
		"PER<10배, PBR>1배 종목":   "PER<10배, PBR>1배 종목",
		"A<B 그리고 C>D":          "A<B 그리고 C>D",
		"<B>대문자</B> 태그":        "대문자 태그",
		"&lt;속보&gt; <b>삼성</b>": "<속보> 삼성",
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q; want %q", in, got, want)
		}
	}
}
