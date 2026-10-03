//go:build integration

package ncloud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kenshin579/ncloud-go"
	"github.com/kenshin579/ncloud-go/search"
)

// 실 API 호출: NAVER_CLIENT_ID / NAVER_CLIENT_SECRET 필요, 콘솔에서 뉴스 검색이 활성화돼 있어야 한다.
//
//	go test -tags integration ./...
func TestNewsLive(t *testing.T) {
	c, err := ncloud.NewClientFromEnv()
	if err != nil {
		t.Skip("NAVER_CLIENT_ID/SECRET not set")
	}
	res, err := search.New(c).News(context.Background(), search.NewsParams{Query: "삼성전자", Display: 5, Sort: search.SortDate})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) == 0 || res.Total == 0 {
		t.Fatalf("no items: %+v", res)
	}
	for _, it := range res.Items {
		if _, err := it.PubTime(); err != nil || it.URL() == "" {
			t.Errorf("item %+v: %v", it, err)
		}
	}
	t.Logf("total=%d first=%s", res.Total, res.Items[0].CleanTitle())
}

func TestGatewayErrorLive(t *testing.T) {
	c, _ := ncloud.NewClient("wrong-id", "wrong-secret")
	_, err := search.New(c).News(context.Background(), search.NewsParams{Query: "a"})
	var ge *ncloud.GatewayError
	if !errors.As(err, &ge) || ge.HTTPStatus != 401 {
		t.Fatalf("want gateway 401, got %v", err)
	}
}
