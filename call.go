package ncloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxBody 는 응답 본문을 읽는 상한이다(검색 100건 JSON 은 수십 KB).
const maxBody = 4 << 20

// GetJSON 은 path 에 GET 요청을 보내고 본문을 out 으로 디코드한다.
// 응답의 Content-Type 은 보지 않는다 — API Hub 검색은 JSON 을 text/plain 으로 준다.
// 본문이 에러 모양이면(2xx 여도) *GatewayError / *APIError, 그 밖의 비 2xx 는 *HTTPError 다.
// 네트워크·타임아웃 에러는 원래 에러를 감싸므로 errors.Is(err, context.DeadlineExceeded) 가 된다.
func GetJSON(ctx context.Context, c *Client, path string, q url.Values, out any) error {
	u := c.baseURL + path
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return c.wrap("ncloud: "+path, err)
	}
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return c.wrap("ncloud: GET "+path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return c.wrap("ncloud: GET "+path, err)
	}
	trace := resp.Header.Get("x-ncp-trace-id")
	if err := decodeError(resp.StatusCode, trace, path, body); err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("ncloud: decode %s (http %d): %w: %q", path, resp.StatusCode, err, head(body, 200))
	}
	return nil
}

type gatewayBody struct {
	Error *struct {
		Code    json.RawMessage `json:"errorCode"`
		Message string          `json:"message"`
		Details string          `json:"details"`
	} `json:"error"`
}

type apiBody struct {
	Code    json.RawMessage `json:"errorCode"`
	Message string          `json:"errorMessage"`
}

// decodeError 는 본문 모양으로 에러를 가른다. 2xx 이고 에러 모양이 아니면 nil.
func decodeError(status int, trace, path string, body []byte) error {
	var g gatewayBody
	if json.Unmarshal(body, &g) == nil && g.Error != nil {
		code := rawString(g.Error.Code)
		if code != "" || g.Error.Message != "" {
			return &GatewayError{HTTPStatus: status, Path: path, Code: code, Message: g.Error.Message, Details: g.Error.Details, TraceID: trace}
		}
	}
	var a apiBody
	if json.Unmarshal(body, &a) == nil {
		if code := rawString(a.Code); code != "" {
			return &APIError{HTTPStatus: status, Path: path, Code: code, Message: a.Message, TraceID: trace}
		}
	}
	if status < 200 || status > 299 {
		return &HTTPError{HTTPStatus: status, Path: path, Body: head(body, 200), TraceID: trace}
	}
	return nil
}

// rawString 은 "200" 과 401 을 모두 "200"/"401" 문자열로 맞춘다. 없거나 null 이면 "".
func rawString(r json.RawMessage) string {
	if len(r) == 0 || string(r) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	return strings.TrimSpace(string(bytes.Trim(r, `"`)))
}

// maskedError 는 메시지에서 Client Secret 을 가리되 원래 에러 체인은 남긴다.
type maskedError struct {
	msg string
	err error
}

func (e *maskedError) Error() string { return e.msg }
func (e *maskedError) Unwrap() error { return e.err }

// wrap 은 "prefix: err" 메시지에서 secret 을 가린 에러를 만든다. 키는 헤더로만 가지만
// *url.Error 는 URL 을, prefix 는 path 를 담으므로 문자열 전체를 가린다(방어).
func (c *Client) wrap(prefix string, err error) error {
	msg := strings.ReplaceAll(prefix+": "+err.Error(), c.secret, "{CLIENT_SECRET}")
	return &maskedError{msg: msg, err: err}
}

func head(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return string(b)
}
