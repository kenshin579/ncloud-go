# ncloud-go

네이버 클라우드 플랫폼(NAVER Cloud Platform) API 의 Go 클라이언트 라이브러리.
루트 패키지 `ncloud` 가 인증·호출·에러를 맡고, 서비스별 API 는 서브패키지로 붙인다.
공식 `ncloud-sdk-go-v2`(인프라 API)가 다루지 않는 NAVER API HUB 부터 지원한다.

## 설치

```bash
go get github.com/kenshin579/ncloud-go@latest
```

Go 1.25+, 외부 의존성 없음(표준 라이브러리만).

## 사용

```go
c, _ := ncloud.NewClientFromEnv() // NAVER_CLIENT_ID, NAVER_CLIENT_SECRET
s := search.New(c)

res, err := s.News(ctx, search.NewsParams{Query: `"네패스"`, Display: 100, Sort: search.SortDate})
for _, it := range res.Items {
    t, _ := it.PubTime()
    fmt.Println(t.Format("01-02 15:04"), it.CleanTitle(), it.URL())
}

// 여러 페이지 (최대 1,000건 — start 상한 1000 때문)
items, err := s.NewsAll(ctx, "삼성전자", search.SortDate, 300)
```

예제: [`examples/basic`](examples/basic/main.go) — 종목명으로 최신 뉴스를 찾고 제목에 종목명이 든 기사만 출력.

## 주의

- **API 마다 콘솔 Application 에서 활성화해야 한다.** 안 하면 `GatewayError`(HTTP 401, code 401).
- 제목·요약에는 `<b>` 태그와 HTML 엔티티가 섞여 온다 — `CleanTitle()`·`CleanDescription()`·`search.Clean`.
- 검색은 본문에 한 번만 나와도 걸린다 — 종목 뉴스로 쓰려면 거르기가 필요하다([뉴스 문서](docs/api/search/news.md#함정--종목-뉴스로-쓸-때)).
- 인자 범위(display 1~100, start 1~1000, sort sim/date)는 호출 전에 검사한다.

## 에러

```go
var ge *ncloud.GatewayError // 키 오류·누락, API 미활성화, 호출 한도 초과 — Code/Message/Details/TraceID
var ae *ncloud.APIError     // 서비스 거부 (검색 SE01~SE04)
var he *ncloud.HTTPError    // 그 밖의 비 2xx (HTML 502 등)

ncloud.StatusCode(err)        // 위 셋의 HTTP 상태, 아니면 0 — 429 쉬기 / 5xx 재시도 / 4xx 영구 실패를 가른다
ncloud.IsRateLimited(err)     // 429
errors.Is(err, search.ErrInvalidParams)       // 인자 범위 오류 (서버를 부르지 않음)
errors.Is(err, context.DeadlineExceeded)      // 타임아웃 — 네트워크 에러 체인은 유지된다
```

## 옵션

```go
c, _ := ncloud.NewClient(id, secret,
    ncloud.WithTimeout(10*time.Second), // 기본 30s
    ncloud.WithBaseURL("https://..."),  // 테스트/프록시
    ncloud.WithHTTPClient(custom),
)
```

## 지원 API

| 서비스 | 패키지 | API | 문서 |
| --- | --- | --- | --- |
| NAVER API HUB 검색 | `search` | 뉴스 검색 | [docs/api/search/news.md](docs/api/search/news.md) |

공통 규약(인증 두 방식, 에러 두 층, Content-Type 함정): [docs/api/README.md](docs/api/README.md)

## 테스트

```bash
go test ./...                     # 단위 테스트 (실측 응답 fixture)
go test -tags integration ./...   # 실 API (NAVER_CLIENT_ID/SECRET 필요)
```

## 릴리스

```bash
./scripts/release.sh vX.Y.Z
```
