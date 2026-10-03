# 뉴스 검색 (search.News)

> `GET https://naverapihub.apigw.ntruss.com/search/v1/news`

네이버 뉴스 검색 결과. 공식 문서: [뉴스 검색 결과 조회](https://api.ncloud-docs.com/docs/naver-api-hub-search-news). 실측일 2026-10-03.

## 요청 인자

| 인자 | 필수 | 기본 | 범위 | 설명 |
| --- | --- | --- | --- | --- |
| query | Y | | | 검색어(UTF-8, URL 인코딩). 따옴표로 묶으면 구절 검색 |
| display | N | 10 | 1~100 | 한 번에 받을 건수. 101 → SE02 |
| start | N | 1 | 1~1000 | 시작 위치. 1001 → SE03. 서버상 최대 1,099건, SDK `NewsAll` 은 1,000건까지 |
| sort | N | sim | sim, date | 정확도순 / 최신순. 그 밖 → SE04 |
| format | N | json | json, xml | xml 은 RSS 2.0. SDK 는 json 고정 |

## 응답 (JSON)

| 필드 | 설명 |
| --- | --- |
| lastBuildDate | 결과 생성 시각, `Sat, 03 Oct 2026 09:38:11 +0900` |
| total | 전체 결과 수 |
| start, display | 이번 응답의 시작 위치·건수(0건이면 display 0) |
| items[].title | 제목. **`<b>` 강조 태그와 HTML 엔티티(`&quot;` 등)가 섞여 온다** |
| items[].originallink | 언론사 원문 URL. 드물게 빈 값(실측: LG 100건 중 3건) |
| items[].link | 네이버 뉴스 URL(`n.news.naver.com`), 없으면 원문 URL과 같음 |
| items[].description | 요약. title 과 같은 태그·엔티티 |
| items[].pubDate | 발행 시각 RFC1123Z(`Fri, 02 Oct 2026 09:16:00 +0900`) |

## 샘플

```
GET /search/v1/news?query=네패스&display=3&sort=date
X-NCP-APIGW-API-KEY-ID: {CLIENT_ID}
X-NCP-APIGW-API-KEY: {CLIENT_SECRET}
```

```json
{
	"lastBuildDate":"Sat, 03 Oct 2026 09:38:11 +0900",
	"total":32921,
	"start":1,
	"display":3,
	"items":[
		{
			"title":"펨트론·엔투텍 급등 주도…고성능 검사·공정 장비주 매수세 유입 뚜렷",
			"originallink":"...",
			"link":"...",
			"description":"... <b>네패스</b> ...",
			"pubDate":"Fri, 02 Oct 2026 09:16:00 +0900"
		}
	]
}
```

0건: `{"total":0,"start":1,"display":0,"items":[]}` (HTTP 200).

## 에러

HTTP 400, `{"errorMessage":"...","errorCode":"SEnn"}`.

| 코드 | 원인 | 메시지 |
| --- | --- | --- |
| SE01 | query 누락 | Incorrect query request (잘못된 쿼리요청입니다.) |
| SE02 | display 범위 밖 | Invalid display value (부적절한 display 값입니다.) |
| SE03 | start 범위 밖 | Invalid start value (부적절한 start 값입니다.) |
| SE04 | sort 값 | Invalid sort value (부적절한 sort값입니다.) |

SDK 는 SE02~SE04 범위를 호출 전에 검사해 서버를 부르지 않는다.

## 함정 — 종목 뉴스로 쓸 때

검색은 **본문에 검색어가 한 번만 나와도** 걸린다. 최신순 100건 중 제목에 검색어가 들어간 기사(실측):

| 검색어 | 100건이 덮는 기간 | 제목 포함 | 섞이는 것 |
| --- | --- | --- | --- |
| 삼성전자 | 약 11시간 | 11 | 본문 언급 기사 |
| 네패스 | 약 16일 | 17 | 다른 장비주 기사 |
| "효성" | 약 1.5일 | 35 | HS효성(별개 회사)·시황 |
| LG | 약 14시간 | 23 | LG 트윈스 야구·공연 |

종목 뉴스로 쓰려면 제목에 종목명이 든 기사만 남기는 거르기가 필요하다. 같은 사건을 여러 언론사가 쓴 기사(예: "네패스 장학금" 6건)도
제목이 거의 같으니 묶는 것이 좋다. 거르기는 SDK 가 아니라 소비자 몫이다(예제 `examples/basic` 참고).
