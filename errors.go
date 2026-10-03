package ncloud

import (
	"errors"
	"fmt"
	"net/http"
)

// GatewayError 는 API 게이트웨이가 요청을 막았을 때의 에러다 — 본문 {"error":{...}}.
// 예: 키 오류·누락(HTTP 401, code "200"), 활성화 안 된 API(HTTP 401, code "401"), 호출 한도 초과(429).
type GatewayError struct {
	HTTPStatus int
	Path       string
	Code       string // errorCode — 응답에 문자열·숫자 두 형태로 와서 문자열로 맞춘다
	Message    string
	Details    string
	TraceID    string // x-ncp-trace-id 응답 헤더 (문의용)
}

func (e *GatewayError) Error() string {
	msg := e.Message
	if e.Details != "" {
		msg += " — " + e.Details
	}
	return fmt.Sprintf("ncloud: %s: gateway error %s: %s (http %d)", e.Path, e.Code, msg, e.HTTPStatus)
}

// APIError 는 서비스가 요청을 거부했을 때의 에러다 — 본문 {"errorCode":"SE01","errorMessage":"..."}.
// 검색: SE01 query, SE02 display, SE03 start, SE04 sort.
type APIError struct {
	HTTPStatus int
	Path       string
	Code       string
	Message    string
	TraceID    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ncloud: %s: api error %s: %s (http %d)", e.Path, e.Code, e.Message, e.HTTPStatus)
}

// HTTPError 는 본문이 게이트웨이·서비스 에러 모양이 아닌 비 2xx 응답이다(HTML 502, 알 수 없는 429 등).
type HTTPError struct {
	HTTPStatus int
	Path       string
	Body       string // 앞 200바이트
	TraceID    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("ncloud: %s: http %d: %q", e.Path, e.HTTPStatus, e.Body)
}

// StatusCode 는 err 가 이 패키지의 HTTP 응답 에러면 그 상태 코드를, 아니면 0 을 준다.
// 소비자는 이것으로 429(쉬기)·5xx(재시도)·4xx(영구 실패)를 가른다.
func StatusCode(err error) int {
	var ge *GatewayError
	if errors.As(err, &ge) {
		return ge.HTTPStatus
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.HTTPStatus
	}
	var he *HTTPError
	if errors.As(err, &he) {
		return he.HTTPStatus
	}
	return 0
}

// IsRateLimited 는 호출 한도 초과(HTTP 429)인지 알려 준다.
func IsRateLimited(err error) bool { return StatusCode(err) == http.StatusTooManyRequests }
