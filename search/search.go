// Package search 는 NAVER API HUB 검색 API 를 제공한다. 지금은 뉴스 검색만 있다.
// 다른 검색(블로그·카페 등)은 콘솔 Application 에서 활성화해야 호출된다 — 안 하면 GatewayError(401).
package search

import "github.com/kenshin579/ncloud-go"

// Service 는 검색 API 묶음이다.
type Service struct{ c *ncloud.Client }

// New 는 ncloud.Client 로 Service 를 만든다.
func New(c *ncloud.Client) *Service { return &Service{c: c} }
