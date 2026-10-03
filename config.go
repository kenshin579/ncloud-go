package ncloud

import (
	"net/http"
	"time"
)

type clientOptions struct {
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// Option 은 functional option 이다.
type Option func(*clientOptions)

// WithBaseURL 은 게이트웨이 주소를 바꾼다 (테스트/프록시용). 기본 DefaultBaseURL.
func WithBaseURL(u string) Option { return func(o *clientOptions) { o.baseURL = u } }

// WithTimeout 은 HTTP 타임아웃을 지정한다 (기본 30s). WithHTTPClient 를 쓰면 무시된다.
func WithTimeout(d time.Duration) Option { return func(o *clientOptions) { o.timeout = d } }

// WithHTTPClient 는 사용자 정의 *http.Client 를 주입한다.
func WithHTTPClient(c *http.Client) Option { return func(o *clientOptions) { o.httpClient = c } }
