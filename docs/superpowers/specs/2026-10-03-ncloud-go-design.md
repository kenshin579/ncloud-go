# ncloud-go 설계 — 공통 규약 + API Hub 뉴스 검색 (v0.1.0)

- 날짜: 2026-10-03
- 상태: 승인됨 (사용자: 네이버 클라우드용 SDK를 자체 저장소로, 이름 `ncloud-go`)
- 소비처: moneyflow 종목 뉴스(국내 종목 상세의 뉴스 영역) — moneyflow 쪽은 별도 스펙

## 배경

국내 종목 뉴스 출처를 검토한 결과(2026-10-02~03) 링크·요약이 있는 공식 API 는 네이버 뉴스 검색이고, 지금은
네이버 클라우드 **NAVER API HUB**(`naverapihub.apigw.ntruss.com`)로 제공된다. 공식 Go SDK
(`NaverCloudPlatform/ncloud-sdk-go-v2`)는 인프라 API 용이고 오래돼 API Hub 를 다루지 않는다. 검색 외 네이버 클라우드
API 도 쓸 수 있으므로 서비스별 서브패키지를 붙이는 범용 SDK 로 만든다.

## 범위

- 이번(v0.1.0): 루트 공통(인증·호출·에러) + `search` 패키지의 **뉴스 검색 하나**.
- 같은 키로 실측한 결과 뉴스만 활성화돼 있다 — 블로그·카페·지식iN·웹문서·이미지·백과·지역·성인·오타 9개는
  401 `요청한 API는 이 Application에서 활성화되어 있지 않습니다.` 응답 형식이 같으므로 콘솔에서 켜면 같은 패턴으로 붙인다.
- 검색어 트렌드·쇼핑 인사이트(POST JSON)·Papago·CLOVA 등은 다음 단계.
- 인프라 API 의 HMAC 서명 인증(`X-NCP-APIGW-SIGNATURE-V2`)은 이번에 구현하지 않지만 루트 설계가 막지 않게 한다.

## 실측 (2026-10-03, 키 `NAVER_CLIENT_ID`/`NAVER_CLIENT_SECRET` — ID 10자, Secret 40자)

- `GET https://naverapihub.apigw.ntruss.com/search/v1/news?query=&display=&start=&sort=&format=`
  - 헤더 `X-NCP-APIGW-API-KEY-ID`, `X-NCP-APIGW-API-KEY`
  - display 1~100(기본 10), start 1~1000(기본 1), sort `sim`|`date`(기본 sim), format `json`|`xml`
- 성공: HTTP 200, **`Content-Type: text/plain;charset=UTF-8` 인데 본문은 JSON** — 헤더로 형식을 판단하지 않는다.
  `{lastBuildDate, total, start, display, items:[{title, originallink, link, description, pubDate}]}`.
  title·description 은 `<b>` 강조 태그와 HTML 엔티티가 섞여 온다. pubDate 는 RFC1123Z(`Sat, 03 Oct 2026 09:12:00 +0900`).
- 0건: HTTP 200, `total:0, display:0, items:[]`.
- 서비스 에러: HTTP 400, `{"errorMessage":"Incorrect query request (잘못된 쿼리요청입니다.)","errorCode":"SE01"}` —
  query 누락 SE01, display 범위 SE02, start 범위 SE03, sort 값 SE04.
- 게이트웨이 에러: HTTP 401, `{"error":{"errorCode":"200","message":"Authentication Failed","details":"..."}}`
  (잘못된 키·키 누락). 활성화 안 된 API 는 `{"error":{"errorCode":401,"message":"요청한 API는 ..."}}` —
  **errorCode 가 문자열일 때와 숫자일 때가 있다.**
- 공지된 상태 코드: 300(URL 오류)·400·401·403(HTTP 호출 등)·429(일 허용량 초과)·500.
- 응답 헤더 `x-ncp-trace-id` — 에러에 담아 문의 때 쓴다.

## 패키지 구조

```
ncloud-go/                   module github.com/kenshin579/ncloud-go
├── client.go config.go      package ncloud — Client(키·HTTP), Option, NewClient/NewClientFromEnv
├── call.go                  GetJSON — GET + 인증 헤더 + 에러 판별 + JSON 디코드 (서비스 패키지가 쓰는 통로)
├── errors.go                GatewayError(HTTP·code·message·details·traceID), APIError(HTTP·code·message·traceID)
├── search/
│   ├── search.go            Service, New(c)
│   ├── news.go              NewsParams, NewsResult, NewsItem, News, NewsAll(페이지 반복), Clean(태그·엔티티 제거)
│   └── *_test.go
├── testdata/                실측 응답(JSON) — 성공·0건·SE01~04·게이트웨이 401 두 형태
├── docs/api/README.md       공통 규약(인증·호스트·에러 두 층·Content-Type 함정)
├── docs/api/search/news.md  뉴스 검색 명세 + 실측
├── examples/basic/main.go
├── integration_test.go      //go:build integration
└── scripts/release.sh       opendata-go 것 재사용
```

## 루트 `ncloud`

- `NewClient(clientID, clientSecret string, opts ...Option) (*Client, error)` — 둘 다 필수.
  `NewClientFromEnv()` 는 `NAVER_CLIENT_ID`/`NAVER_CLIENT_SECRET`.
- 옵션 `WithBaseURL`(기본 `https://naverapihub.apigw.ntruss.com`), `WithTimeout`(30s), `WithHTTPClient`.
- `GetJSON(ctx, c, path string, q url.Values, out any) error`
  - 2xx: 본문을 out 으로 JSON 디코드(Content-Type 무시).
  - 비 2xx: 본문이 `{"error":{...}}` 면 `*GatewayError`, `{"errorCode":..,"errorMessage":..}` 면 `*APIError`,
    그 밖엔 상태·본문 앞부분을 담은 일반 에러. errorCode 는 문자열/숫자 모두 받는다(`json.RawMessage` → 문자열).
  - 키는 헤더로만 가므로 URL 에 없지만, 에러 문자열에 secret 이 섞이지 않게 마스킹한다(방어).
- 인증 방식은 `Client` 안의 헤더 서명 함수 하나로 둔다 — 나중에 HMAC 서명 방식을 같은 자리에 추가할 수 있다.

## `search` 패키지

- `News(ctx, NewsParams{Query, Display, Start, Sort}) (*NewsResult, error)` — 진입 시 검증(Query 필수,
  Display 0 또는 1~100, Start 0 또는 1~1000, Sort ""|SortSim|SortDate). 0 은 서버 기본값.
- `NewsItem` 은 원문 필드를 그대로 두고 `PubTime() (time.Time, error)` 와 `CleanTitle()`·`CleanDescription()`
  (태그 제거 + HTML 엔티티 해제)를 준다.
- `NewsAll(ctx, query, sort, max int)` — display 100 으로 start 를 넘기며 max(≤1100) 건까지 모은다. 빈 페이지에서 멈춘다.
- format 은 JSON 고정(XML 은 지원하지 않는다).

## 테스트·릴리스

- `httptest` + testdata 실측 응답으로 성공·0건·SE01~04·게이트웨이 두 형태·비 JSON 응답·헤더 전송·검증 실패 시
  서버 미호출을 확인. `NewsAll` 페이지 반복·중단 조건. `Clean`·`PubTime` 표 테스트.
- `-tags integration`: 실 키로 뉴스 1건 + 잘못된 키 → GatewayError.
- `scripts/release.sh v0.1.0` → moneyflow 가 go.mod 태그로 소비.

## 완료 기준

`go vet ./... && go test ./...` 통과, 통합 테스트 통과, README(설치·사용·에러·활성화 안내), v0.1.0 릴리스.
