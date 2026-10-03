package ncloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetJSON 은 path 에 GET 요청을 보내고 2xx 본문을 out 으로 디코드한다.
// 응답의 Content-Type 은 보지 않는다 — API Hub 검색은 JSON 을 text/plain 으로 준다.
// 비 2xx 는 본문 모양으로 *GatewayError / *APIError 를 가른다.
func GetJSON(ctx context.Context, c *Client, path string, q url.Values, out any) error {
	u := c.baseURL + path
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return c.errorf("ncloud: %s: %v", path, err)
	}
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return c.errorf("ncloud: GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.errorf("ncloud: GET %s: %v", path, err)
	}
	trace := resp.Header.Get("x-ncp-trace-id")
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return decodeError(resp.StatusCode, trace, path, body)
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
	Code    string `json:"errorCode"`
	Message string `json:"errorMessage"`
}

func decodeError(status int, trace, path string, body []byte) error {
	var g gatewayBody
	if json.Unmarshal(body, &g) == nil && g.Error != nil {
		return &GatewayError{HTTPStatus: status, Code: rawString(g.Error.Code), Message: g.Error.Message, Details: g.Error.Details, TraceID: trace}
	}
	var a apiBody
	if json.Unmarshal(body, &a) == nil && a.Code != "" {
		return &APIError{HTTPStatus: status, Code: a.Code, Message: a.Message, TraceID: trace}
	}
	return fmt.Errorf("ncloud: GET %s: http %d: %q", path, status, head(body, 200))
}

// rawString 은 "200" 과 401 을 모두 "200"/"401" 문자열로 맞춘다.
func rawString(r json.RawMessage) string {
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	return strings.TrimSpace(string(bytes.Trim(r, `"`)))
}

// errorf 는 메시지 전체에서 Client Secret 을 지운 에러를 만든다. 키는 헤더로만 가지만
// *url.Error 는 URL 을, 접두사는 path 를 담으므로 문자열 전체를 가린다(방어).
func (c *Client) errorf(format string, args ...any) error {
	return errors.New(strings.ReplaceAll(fmt.Sprintf(format, args...), c.secret, "{CLIENT_SECRET}"))
}

func head(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return string(b)
}
