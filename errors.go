package ncloud

import "fmt"

// GatewayError 는 API 게이트웨이가 요청을 막았을 때의 에러다 — 본문 {"error":{...}}.
// 예: 키 오류·누락(HTTP 401, code "200"), 활성화 안 된 API(HTTP 401, code "401"), 호출 한도 초과(429).
type GatewayError struct {
	HTTPStatus int
	Code       string // errorCode — 응답에 문자열·숫자 두 형태로 와서 문자열로 맞춘다
	Message    string
	Details    string
	TraceID    string // x-ncp-trace-id 응답 헤더 (문의용)
}

func (e *GatewayError) Error() string {
	s := fmt.Sprintf("ncloud: gateway error %s: %s (http %d", e.Code, e.Message, e.HTTPStatus)
	if e.Details != "" {
		s = fmt.Sprintf("ncloud: gateway error %s: %s — %s (http %d", e.Code, e.Message, e.Details, e.HTTPStatus)
	}
	return s + ")"
}

// APIError 는 서비스가 요청을 거부했을 때의 에러다 — 본문 {"errorCode":"SE01","errorMessage":"..."}.
// 검색: SE01 query, SE02 display, SE03 start, SE04 sort.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	TraceID    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ncloud: api error %s: %s (http %d)", e.Code, e.Message, e.HTTPStatus)
}
