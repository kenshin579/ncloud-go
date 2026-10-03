// Package ncloud 는 네이버 클라우드 플랫폼(NAVER Cloud Platform) API 의 Go 클라이언트다.
// 루트 패키지는 인증·호출·에러를 맡고, 서비스별 API 는 서브패키지(search 등)가 제공한다.
package ncloud

import (
	"errors"
	"net/http"
	"os"
	"time"
)

// DefaultBaseURL 은 NAVER API HUB 게이트웨이 주소다.
const DefaultBaseURL = "https://naverapihub.apigw.ntruss.com"

// 환경변수 이름 — NewClientFromEnv 가 읽는다.
const (
	EnvClientID     = "NAVER_CLIENT_ID"
	EnvClientSecret = "NAVER_CLIENT_SECRET"
)

// Client 는 네이버 클라우드 API 호출 통로다. 동시에 여러 고루틴에서 써도 된다.
type Client struct {
	id, secret string
	baseURL    string
	http       *http.Client
}

// NewClient 는 콘솔의 Application Client ID/Secret 으로 Client 를 만든다.
func NewClient(clientID, clientSecret string, opts ...Option) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("ncloud: clientID and clientSecret are required")
	}
	cfg := clientOptions{baseURL: DefaultBaseURL, timeout: 30 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}
	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.timeout}
	}
	return &Client{id: clientID, secret: clientSecret, baseURL: cfg.baseURL, http: hc}, nil
}

// NewClientFromEnv 는 NAVER_CLIENT_ID / NAVER_CLIENT_SECRET 으로 Client 를 만든다.
func NewClientFromEnv(opts ...Option) (*Client, error) {
	return NewClient(os.Getenv(EnvClientID), os.Getenv(EnvClientSecret), opts...)
}

// authorize 는 요청에 인증 헤더를 붙인다. API Gateway 키 방식(AI·NAVER API, API Hub)이다.
// 인프라 API 의 HMAC 서명 방식은 이 자리에 추가한다.
func (c *Client) authorize(req *http.Request) {
	req.Header.Set("X-NCP-APIGW-API-KEY-ID", c.id)
	req.Header.Set("X-NCP-APIGW-API-KEY", c.secret)
}
