// 종목명으로 최신 뉴스를 찾고, 제목에 종목명이 들어간 기사만 출력하는 예제.
//
//	NAVER_CLIENT_ID=... NAVER_CLIENT_SECRET=... go run ./examples/basic 네패스
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kenshin579/ncloud-go"
	"github.com/kenshin579/ncloud-go/search"
)

func main() {
	name := "삼성전자"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	c, err := ncloud.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	items, err := search.New(c).NewsAll(context.Background(), `"`+name+`"`, search.SortDate, 100)
	if err != nil {
		log.Fatal(err)
	}
	for _, it := range items {
		title := it.CleanTitle()
		if !strings.Contains(title, name) { // 본문에만 언급된 기사는 거른다
			continue
		}
		t, _ := it.PubTime()
		fmt.Printf("%s  %s\n            %s\n", t.Format("01-02 15:04"), title, it.URL())
	}
}
