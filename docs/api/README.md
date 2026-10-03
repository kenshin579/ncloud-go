# 네이버 클라우드 API 공통 규약

이 저장소가 다루는 네이버 클라우드 API 의 공통 사항. 서비스별 문서는 하위 디렉토리에 있다 — [검색](search/news.md).
실측일 2026-10-03.

## 인증

네이버 클라우드에는 인증 방식이 둘 있다.

| 대상 | 방식 | 이 SDK |
| --- | --- | --- |
| NAVER API HUB(검색 등)·AI·NAVER API(Papago·CLOVA·Maps) | Application 의 Client ID/Secret 을 헤더로 | 지원 |
| 인프라 API(서버·VPC·Object Storage 등) | Access Key + HMAC 서명(`X-NCP-APIGW-SIGNATURE-V2`) | 미지원(루트 `authorize` 자리에 추가 예정) |

```
X-NCP-APIGW-API-KEY-ID: {Client ID}       # 10자
X-NCP-APIGW-API-KEY:    {Client Secret}   # 40자
```

- 콘솔 › NAVER API HUB › Application 에서 발급한다. 환경변수 `NAVER_CLIENT_ID` / `NAVER_CLIENT_SECRET`.
- **API 마다 Application 에서 활성화해야 한다.** 안 하면 키가 맞아도 401(아래 표). 2026-10-03 기준 활성화: 뉴스 검색.
- 네이버 개발자센터(developers.naver.com)의 검색 API 키(ID 20자·Secret 10자, `X-Naver-Client-Id` 헤더)와는 다른 체계다 —
  개발자센터 키를 넣으면 `NID AUTH Result Invalid` 같은 에러가 나는 것도 그 때문이다.

## 호스트

| 서비스 | 베이스 URL |
| --- | --- |
| NAVER API HUB | `https://naverapihub.apigw.ntruss.com` |

## 응답

- 성공은 HTTP 200 + JSON. **`Content-Type` 이 `text/plain;charset=UTF-8` 로 온다** — 헤더로 형식을 판단하면 안 된다.
- 응답 헤더 `x-ncp-trace-id` — 문의할 때 쓰는 추적 ID. SDK 는 에러에 담는다.

## 에러 — 두 층

본문 모양이 다르므로 상태 코드가 아니라 **본문으로** 가른다.

### 게이트웨이 (`ncloud.GatewayError`)

```json
{"error":{"errorCode":"200","message":"Authentication Failed","details":"Invalid authentication information."}}
```

| 상황 | HTTP | errorCode | message / details |
| --- | --- | --- | --- |
| 잘못된 키 | 401 | `"200"` (문자열) | Authentication Failed / Invalid authentication information. |
| 키 헤더 없음 | 401 | `"200"` (문자열) | Authentication Failed / Authentication information are missing. |
| Application 에서 활성화 안 된 API | 401 | `401` (**숫자**) | 요청한 API는 이 Application에서 활성화되어 있지 않습니다. |

`errorCode` 가 문자열일 때와 숫자일 때가 있다 — 파서는 둘 다 받아야 한다(SDK 는 문자열로 맞춘다).

### 서비스 (`ncloud.APIError`)

```json
{"errorMessage":"Incorrect query request (잘못된 쿼리요청입니다.)","errorCode":"SE01"}
```

검색의 코드는 [검색 문서](search/news.md#에러)에 있다.

### 공지된 상태 코드 (미실측 포함)

| HTTP | 의미 |
| --- | --- |
| 300 | API 요청 URL 이 잘못됨 |
| 400 | 필수 변수 부재·변수명 오류·URL 인코딩 미처리 (실측: 검색 SE01~SE04) |
| 401 | 인증 실패·API 권한 미설정 (실측: 위 세 가지) |
| 403 | 서버가 허용하지 않는 호출(HTTP 호출 등) — 미실측 |
| 429 | 하루 허용량 초과 — 미실측 |
| 500 | 서버 오류 — 미실측 |

## 요금

NAVER API HUB 는 사용량 기반 과금이다. 정확한 단가·무료 제공량은 콘솔 요금 화면에서 확인할 것(이 문서 작성 시 공식 요금표로 확인하지 못함).
