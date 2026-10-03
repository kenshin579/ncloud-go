package ncloud

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testID     = "abcdefghij"
	testSecret = "0123456789012345678901234567890123456789"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// serve 는 status·body 로 응답하고(Content-Type 은 실측대로 text/plain) 마지막 요청을 *got 에 담는다.
func serve(t *testing.T, status int, body string, got **http.Request) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got != nil {
			*got = r
		}
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		w.Header().Set("x-ncp-trace-id", "trace-1")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(testID, testSecret, WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type result struct {
	Total int `json:"total"`
	Items []struct {
		Title string `json:"title"`
	} `json:"items"`
}

func TestGetJSONSendsAuthAndQuery(t *testing.T) {
	var r *http.Request
	c := serve(t, 200, fixture(t, "news_ok.json"), &r)
	var out result
	if err := GetJSON(context.Background(), c, "/search/v1/news", url.Values{"query": {"네패스"}}, &out); err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("X-NCP-APIGW-API-KEY-ID") != testID || r.Header.Get("X-NCP-APIGW-API-KEY") != testSecret {
		t.Errorf("auth headers = %v", r.Header)
	}
	if r.URL.Path != "/search/v1/news" || r.URL.Query().Get("query") != "네패스" {
		t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
	}
	if strings.Contains(r.URL.RawQuery, testSecret) {
		t.Error("secret must not be in the URL")
	}
	if out.Total == 0 || len(out.Items) == 0 {
		t.Errorf("decoded %+v — text/plain JSON must still decode", out)
	}
}

func TestGetJSONGatewayError(t *testing.T) {
	cases := []struct{ file, code string }{
		{"gateway_bad_key.json", "200"}, // errorCode 가 문자열
		{"gateway_no_key.json", "200"},
		{"gateway_not_enabled.json", "401"}, // errorCode 가 숫자
	}
	for _, tc := range cases {
		c := serve(t, 401, fixture(t, tc.file), nil)
		err := GetJSON(context.Background(), c, "/x", nil, &result{})
		var ge *GatewayError
		if !errors.As(err, &ge) || ge.Code != tc.code || ge.HTTPStatus != 401 || ge.Message == "" || ge.TraceID != "trace-1" {
			t.Errorf("%s: err=%v (%+v)", tc.file, err, ge)
		}
	}
}

func TestGetJSONAPIError(t *testing.T) {
	for _, code := range []string{"SE01", "SE02", "SE03", "SE04"} {
		c := serve(t, 400, fixture(t, "api_"+strings.ToLower(code)+".json"), nil)
		err := GetJSON(context.Background(), c, "/x", nil, &result{})
		var ae *APIError
		if !errors.As(err, &ae) || ae.Code != code || ae.HTTPStatus != 400 || ae.Message == "" {
			t.Errorf("%s: err=%v", code, err)
		}
	}
}

func TestGetJSONUnexpected(t *testing.T) {
	c := serve(t, 502, "<html>Bad Gateway</html>", nil)
	err := GetJSON(context.Background(), c, "/x", nil, &result{})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err=%v", err)
	}
	c = serve(t, 200, "not json", nil)
	if err := GetJSON(context.Background(), c, "/x", nil, &result{}); err == nil {
		t.Fatal("want decode error")
	}
}

func TestGetJSONMasksSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	// secret 이 URL 에 섞인 경로로 에러를 유도해 마스킹을 확인한다.
	c, _ := NewClient(testID, testSecret, WithBaseURL(srv.URL), WithTimeout(20*time.Millisecond))
	err := GetJSON(context.Background(), c, "/"+testSecret, nil, &result{})
	if err == nil || strings.Contains(err.Error(), testSecret) {
		t.Fatalf("secret not masked: %v", err)
	}
}

func TestGetJSONKeepsErrorChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c, _ := NewClient(testID, testSecret, WithBaseURL(srv.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := GetJSON(ctx, c, "/x", nil, &result{})
	// 소비자가 타임아웃·취소를 실패와 구분해 캐시하지 않으려면 체인이 살아 있어야 한다.
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("errors.Is(DeadlineExceeded) = false: %v", err)
	}
}

func TestGetJSONHTTPErrorAndStatus(t *testing.T) {
	cases := []struct {
		status int
		body   string
	}{
		{502, "<html>Bad Gateway</html>"},
		{429, `{"error":"quota"}`},
		{400, `{"error":{}}`},
	}
	for _, tc := range cases {
		c := serve(t, tc.status, tc.body, nil)
		err := GetJSON(context.Background(), c, "/x", nil, &result{})
		var he *HTTPError
		if !errors.As(err, &he) || he.HTTPStatus != tc.status || StatusCode(err) != tc.status {
			t.Errorf("%d %s: err=%v", tc.status, tc.body, err)
		}
	}
	c := serve(t, 429, `{"errorCode":429,"errorMessage":"quota exceeded"}`, nil)
	err := GetJSON(context.Background(), c, "/x", nil, &result{})
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "429" || !IsRateLimited(err) {
		t.Errorf("numeric api errorCode: err=%v", err)
	}
	if StatusCode(errors.New("x")) != 0 {
		t.Error("StatusCode of non-http error must be 0")
	}
}

func TestGetJSONErrorBodyOn200(t *testing.T) {
	c := serve(t, 200, `{"errorMessage":"Incorrect query request","errorCode":"SE01"}`, nil)
	err := GetJSON(context.Background(), c, "/x", nil, &result{})
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "SE01" {
		t.Fatalf("2xx error body must be an error, got %v", err)
	}
}

func TestWithBaseURLTrailingSlash(t *testing.T) {
	var r *http.Request
	c := serve(t, 200, `{"total":0,"items":[]}`, &r)
	c2, _ := NewClient(testID, testSecret, WithBaseURL(c.baseURL+"/"))
	if err := GetJSON(context.Background(), c2, "/search/v1/news", nil, &result{}); err != nil {
		t.Fatal(err)
	}
	if r.URL.Path != "/search/v1/news" {
		t.Errorf("path = %q", r.URL.Path)
	}
}
