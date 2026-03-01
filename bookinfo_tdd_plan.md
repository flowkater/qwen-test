# Test-Driven Implementation Plan: BookInfo CLI

## Overview
`bookinfo_plan.md` 요구사항을 기반으로, Go CLI 애플리케이션을 테스트 우선 방식으로 구현하기 위한 실행 계획이다. 각 항목은 Red-Green 사이클에서 독립 실행 가능한 단일 검증 단위로 분해되어 있으며, `tdd-go`/`tdd-go-loop`가 바로 수행할 수 있도록 Phase/Tier 및 실행 메타데이터를 포함한다.

## Stack Profile
- Target stack: Go CLI
- Architecture profile: `Domain -> Application -> Infrastructure -> Delivery(CLI)`
- test_command_red: `go test -v ./... -run '<TargetTestName>' -count=1`
- test_command_green: `go test -v ./... -count=1`
- integration_test_command: `go test -v -tags=integration ./... -count=1`
- format_command: `gofmt -w $(find . -name '*.go' -not -path './vendor/*')`
- lint_command: `go vet ./...`

## Requirements Digest

### Entities / Objects
- `BookQuery`: title, isbn13, author, lang, model, output, noCache, fullMode, batchPath
- `BookInfo`: 최종 통합 결과 루트 객체
- `BookMetadata`: 도서 메타데이터 (제목/저자/출판사/ISBN-13 등)
- `LocalizedTitle`: 원어/한국어 병기 구조
- `TOCNode`: 재귀형 목차 노드 (`children`)
- `ReviewInfo`: 평점, 장단점, 권장 수준, 전제지식
- `RelatedCourse`: 강의 정보 구조
- `SimilarBook`: 유사 도서 구조
- `DataAvailability`: 각 섹션별 데이터 존재 여부
- `CollectMetadata`: 수집 시각/모델/소스/모드/쿼리정보
- `CacheEntry`: 캐시 키/값/생성시각
- `BatchItemResult`: 배치 단일 항목 결과
- `BackoffConfig`: 초기 지연(2s)/배수(2x)/jitter(0~1s)/최대 대기(30s)/최대 재시도(3회)

### Actions / Use-Cases
- CLI 인자 파싱 및 검증 (제목 또는 ISBN-13 필수)
- `--lang` 허용값 검증 (`ko`, `en`, `ja`, `zh-tw`)
- `--full` 모드 분기 (기본 2회 vs 전체 5회 API 호출)
- API 키 해석 (환경변수 우선, 플래그 오버라이드)
- 캐시 조회/저장 (`--no-cache` 시 읽기만 우회, 쓰기는 수행)
- 기본 모드: 메타데이터 + 목차 수집 (API 2회)
- 전체 모드: 메타데이터 + 목차 + 후기 + 강의 + 유사도서 수집 (API 5회)
- JSON 구조 검증 및 실패 시 프롬프트 보정 재시도
- 응답 조각 병합 + 미수집 섹션 빈 값 채움 + `data_availability` 생성
- stdout 요약 출력
- 파일 저장 (로컬 타임존 날짜, `-o` 오버라이드)
- 배치 파일 순차 처리 (고정 3초 + 429 시 동적 증가, 부분 실패 허용)

### Rules / Constraints
- 단일 실행 시 책 제목 또는 `--isbn` 중 하나 필수 (`--batch` 모드 제외)
- 제목과 `--isbn` 동시 입력 시 ISBN 우선
- 기본 모델은 `qwen/qwen3.5-flash`
- `--api-key` 지정 시 환경변수보다 우선
- 캐시 경로는 `~/.cache/bookinfo/`
- 캐시 키: 제목(또는 ISBN) + 저자 + 언어 + 모델 + 수집모드(basic/full) 해시
- `--no-cache`: 캐시 읽기만 우회, 새 결과는 캐시에 저장
- `--lang` 허용값: `ko`, `en`, `ja`, `zh-tw` (고정), 미지정 시 자동 판단
- 목차 depth는 고정 제한 없음 (모델 응답 최대 깊이 허용)
- 목차/유사도서 제목은 원어+한국어 병기
- 유사 도서 추천 개수는 3~5권
- 동명 도서: 인기 우선, 동저자 시 최신판 선택
- API/JSON 오류 재시도 최대 3회
- 백오프: 초기 2s, 배수 2x, jitter 0~1s, 캡 30s
- 네트워크 타임아웃 기본 30초
- 배치: 순차 처리, 항목 간 고정 3초, 429 시 2배 증가(최대 30s), 3연속 성공 시 복원
- 데이터 없음 시: 빈 배열/객체 (null/에러 아님)
- 파일명 날짜: 로컬 타임존 / JSON `collected_at`: UTC

### API / Interface Contract
- CLI: `bookinfo <title> [flags]` 또는 `bookinfo --isbn <isbn13> [flags]` 또는 `bookinfo --batch <file>`
- LLM Client Interface (가정):
  - `CollectTOC(ctx, query) -> TOC payload`
  - `CollectMetadata(ctx, query) -> Metadata payload`
  - `CollectReview(ctx, query) -> Review payload` (full mode only)
  - `CollectCourses(ctx, query) -> Courses payload` (full mode only)
  - `CollectSimilarBooks(ctx, query) -> Similar payload` (full mode only)
- Validator Interface (가정):
  - `ValidateTOCJSON(raw)`
  - `ValidateMetadataJSON(raw)`
  - `ValidateReviewJSON(raw)`
  - `ValidateCoursesJSON(raw)`
  - `ValidateSimilarBooksJSON(raw)`
- Cache Interface (가정):
  - `Get(key)`, `Set(key, value)`, `Exists(key)`

### Error / Edge Cases
- 제목/ISBN 둘 다 없음
- 제목 불일치(존재하지 않는 책) 응답 처리
- ISBN으로 책을 못 찾는 경우
- API 키 미설정
- `--lang` 허용값 외 입력
- `429/500` 백오프 재시도
- JSON 파싱 실패 및 프롬프트 보정 재시도
- 네트워크 타임아웃
- 동명 도서 다수 (인기 우선, 동저자 시 최신판)
- 배치 중 일부 실패
- 캐시 디렉토리 미존재
- 캐시 파일 손상 → cache miss 처리
- 전체 모드에서 후기/강의/유사도서 데이터 없음 → 빈 배열

---

## Phase 1: Contract / Interface / DTO [T1] <!-- T1:auto -->

### 1.1 CLI Contract
- [ ] Test: 단일 실행에서 제목 인자를 전달하면 `BookQuery.Title`이 채워진다.
- [ ] Test: `--isbn` 입력 시 `BookQuery.ISBN13`이 설정된다.
- [ ] Test: 제목과 `--isbn` 동시 입력 시 `BookQuery.ISBN13`이 우선 사용된다.
- [ ] Test: 단일 실행에서 제목도 `--isbn`도 `--batch`도 없으면 validation error를 반환한다.
- [ ] Test: `--author` 입력 시 `BookQuery.Author`가 설정된다.
- [ ] Test: `--lang` 입력 시 `BookQuery.Lang`가 설정된다.
- [ ] Test: `--full` 입력 시 `BookQuery.FullMode`가 true가 된다.
- [ ] Test: `--full` 미입력 시 `BookQuery.FullMode`가 false(기본값)이다.
- [ ] Test: `--model` 미입력 시 기본값 `qwen/qwen3.5-flash`가 설정된다.
- [ ] Test: `-o/--output` 입력 시 사용자 지정 경로가 적용된다.
- [ ] Test: `--batch` 입력 시 배치 모드로 분기되고 제목/ISBN 필수 검사를 건너뛴다.
- [ ] Test: `--no-cache` 입력 시 `BookQuery.NoCache`가 true가 된다.

### 1.2 Config / Security Contract
- [ ] Test: `OPENROUTER_API_KEY`가 있으면 API 키 소스로 환경변수를 사용한다.
- [ ] Test: `OPENROUTER_API_KEY`와 `--api-key`가 동시에 있으면 플래그 값을 우선 사용한다.
- [ ] Test: API 키가 둘 다 없으면 사용자 가이드 포함 에러를 반환한다.
- [ ] Test: 에러/로그 문자열에 API 키 원문이 포함되지 않는다.

### 1.3 DTO / Schema Contract
- [ ] Test: `BookInfo` 직렬화 시 `book`, `table_of_contents`, `review`, `related_courses`, `similar_books`, `data_availability`, `metadata` 루트 필드가 존재한다.
- [ ] Test: `LocalizedTitle` 직렬화 시 `original`, `korean` 필드가 모두 존재한다.
- [ ] Test: `TOCNode` 직렬화 시 `children` 필드가 nil 대신 빈 배열로 출력된다.
- [ ] Test: `SimilarBook` 직렬화 시 `difficulty_comparison` 필드가 누락되지 않는다.
- [ ] Test: `CollectMetadata.collected_at`은 UTC RFC3339 형식 문자열로 직렬화된다.
- [ ] Test: `BookMetadata` 직렬화 시 `isbn13` 필드가 포함된다 (`isbn` 대신).
- [ ] Test: `BookMetadata` 직렬화 시 `selection_note` 필드가 포함된다.
- [ ] Test: `DataAvailability` 직렬화 시 `table_of_contents`, `review`, `related_courses`, `similar_books` boolean 필드가 모두 존재한다.
- [ ] Test: `CollectMetadata` 직렬화 시 `mode` 필드가 `"basic"` 또는 `"full"`로 존재한다.
- [ ] Test: `CollectMetadata` 직렬화 시 `query` 객체에 `title`, `isbn13`, `author`, `lang`이 존재한다.

---

## Phase 2: Core Rule / Domain Logic [T2] <!-- T2:review -->

### 2.1 Query Validation Rules
- [ ] Test: 제목이 공백 문자열이면 invalid title error를 반환한다.
- [ ] Test: `--isbn` 값이 공백이면 invalid isbn error를 반환한다.
- [ ] Test: `--lang` 값이 허용 목록(`ko`, `en`, `ja`, `zh-tw`)에 없으면 invalid lang error를 반환한다.
- [ ] Test: `--lang` 미지정 시 빈 문자열로 설정되고 validation을 통과한다.
- [ ] Test: `--batch` 모드에서 빈 파일 경로면 invalid batch path error를 반환한다.
- [ ] Test: 출력 파일 경로가 디렉토리면 invalid output path error를 반환한다.

### 2.2 Cache Key / Policy Rules
- [ ] Test: 캐시 키 생성 시 제목+저자+언어+모델+모드 조합이 동일하면 동일 키를 생성한다.
- [ ] Test: 캐시 키 생성 시 ISBN+저자+언어+모델+모드 조합이 동일하면 동일 키를 생성한다.
- [ ] Test: 캐시 키 생성 시 모델이 다르면 다른 키를 생성한다.
- [ ] Test: 캐시 키 생성 시 수집 모드(basic/full)가 다르면 다른 키를 생성한다.
- [ ] Test: `--no-cache=true`면 캐시 **읽기** 단계가 skip된다.
- [ ] Test: `--no-cache=true`여도 캐시 **쓰기** 단계는 수행된다 (덮어쓰기).

### 2.3 Retry / Backoff Rules
- [ ] Test: 429 에러 발생 시 재시도 카운터가 증가한다.
- [ ] Test: 500 에러 발생 시 재시도 카운터가 증가한다.
- [ ] Test: 최대 재시도 3회 초과 시 최종 에러를 반환한다.
- [ ] Test: 백오프 지연은 시도 횟수 증가에 따라 단조 증가한다 (2s → 4s → 8s).
- [ ] Test: 백오프 지연에 0~1초 jitter가 추가된다.
- [ ] Test: 백오프 지연이 최대 캡(30초)을 초과하지 않는다.
- [ ] Test: JSON 파싱 실패 시 프롬프트 보정 함수가 호출된다.
- [ ] Test: JSON 파싱 실패가 3회 연속이면 invalid response error를 반환한다.

### 2.4 Merge / Consistency Rules
- [ ] Test: 기본 모드에서 2개 수집 조각(메타+목차)이 유효하면 `BookInfo` 병합이 성공한다.
- [ ] Test: 전체 모드에서 5개 수집 조각이 모두 유효하면 `BookInfo` 병합이 성공한다.
- [ ] Test: 기본 모드 병합 시 `review`는 빈 객체, `related_courses`/`similar_books`는 빈 배열로 채워진다.
- [ ] Test: 기본 모드 병합 시 `data_availability.review`/`related_courses`/`similar_books`가 false이다.
- [ ] Test: 전체 모드에서 후기 데이터가 없으면 `review`는 빈 객체, `data_availability.review`는 false이다.
- [ ] Test: `table_of_contents` 노드 depth 값은 부모보다 1 이상이어야 한다.
- [ ] Test: 목차 노드 제목은 원어+한국어가 모두 채워져야 유효 처리된다.
- [ ] Test: 유사 도서 추천 수가 2 이하이면 validation error를 반환한다 (전체 모드).
- [ ] Test: 유사 도서 추천 수가 6 이상이면 validation error를 반환한다 (전체 모드).
- [ ] Test: 평점 값이 허용 범위(0~5) 밖이면 validation error를 반환한다 (전체 모드).

### 2.5 Duplicate Book Selection Rules
- [ ] Test: 프롬프트에 동명 도서 선택 기준(인기 우선)이 포함된다.
- [ ] Test: 프롬프트에 동저자+동명 시 최신판 선택 지시가 포함된다.
- [ ] Test: 응답에 `selection_note` 필드가 채워진다.

---

## Phase 3: Application Orchestration [T2] <!-- T2:review -->

### 3.1 Single Book Flow — Basic Mode
- [ ] Test: 캐시 히트 시 API 호출 없이 결과를 반환한다.
- [ ] Test: `--no-cache` 시 캐시가 존재해도 API 호출을 수행한다.
- [ ] Test: `--no-cache` 시 수집 완료 후 캐시에 결과를 저장(덮어쓰기)한다.
- [ ] Test: 기본 모드에서 수집 순서가 Metadata→TOC 순으로 실행된다 (API 2회).
- [ ] Test: 기본 모드에서 Review/Courses/SimilarBooks 수집이 호출되지 않는다.
- [ ] Test: 중간 단계 실패 시 이후 단계 호출이 중단되고 오류가 반환된다.
- [ ] Test: 2개 단계 성공 후 merge가 한 번만 호출된다.
- [ ] Test: 성공 결과 반환 전에 stdout 요약 DTO가 생성된다.

### 3.2 Single Book Flow — Full Mode
- [ ] Test: 전체 모드에서 수집 순서가 Metadata→TOC→Review→Courses→SimilarBooks로 실행된다 (API 5회).
- [ ] Test: 전체 모드에서 5개 단계 성공 후 merge가 한 번만 호출된다.

### 3.3 ISBN Search Flow
- [ ] Test: `--isbn` 입력 시 프롬프트에 ISBN-13이 포함된다.
- [ ] Test: `--isbn` 입력 시 캐시 키가 ISBN 기반으로 생성된다.
- [ ] Test: 제목과 ISBN 동시 입력 시 프롬프트에 ISBN이 우선 사용된다.

### 3.4 Prompt Separation Orchestration
- [ ] Test: TOC 수집 프롬프트는 depth 제한 문구 없이 구성된다.
- [ ] Test: Metadata 수집 프롬프트는 ISBN-13/pages/language/edition 필드를 요구한다.
- [ ] Test: Review 수집 프롬프트는 pros/cons/recommended_level/prerequisites를 요구한다.
- [ ] Test: Courses 수집 프롬프트는 platform/instructor/curriculum/rating/price/url을 요구한다.
- [ ] Test: SimilarBooks 수집 프롬프트는 3~5권과 난이도 비교를 요구한다.
- [ ] Test: 모든 프롬프트에 원어+한국어 병기 지시가 포함된다.
- [ ] Test: 모든 프롬프트에 동명 도서 선택 기준(인기 우선, 동저자 시 최신판)이 포함된다.

### 3.5 Output Orchestration
- [ ] Test: 출력 파일 경로 미지정 시 `{title}_{yyyymmdd}.json` 패턴 파일명을 생성한다 (로컬 타임존).
- [ ] Test: ISBN 검색 시 파일명이 `{isbn13}_{yyyymmdd}.json` 패턴을 따른다.
- [ ] Test: 생성 파일명에서 OS 금지 문자를 안전한 문자로 정규화한다.
- [ ] Test: 파일 저장 성공 시 절대 경로를 결과에 포함한다.
- [ ] Test: 파일 저장 실패 시 stdout 요약은 출력하되 종료 코드는 실패로 반환한다.
- [ ] Test: JSON 내부 `collected_at`은 UTC RFC3339 형식이다.

### 3.6 Batch Orchestration
- [ ] Test: 배치 파일에서 빈 줄은 무시하고 유효 제목만 추출한다.
- [ ] Test: 배치 항목은 동시 실행 없이 순차 처리된다.
- [ ] Test: 배치 항목 간 기본 딜레이가 3초이다.
- [ ] Test: 배치 중 429 발생 시 다음 항목 간 딜레이가 2배 증가한다 (최대 30초).
- [ ] Test: 배치 중 3연속 성공 시 딜레이가 기본값(3초)으로 복원된다.
- [ ] Test: 항목 실패 시 실패 로그를 기록하고 다음 항목을 계속 처리한다.
- [ ] Test: 배치 완료 시 성공/실패 집계 요약을 출력한다.
- [ ] Test: 배치 모드에서도 `--full` 플래그가 전체 항목에 적용된다.

---

## Phase 4: Data / Integration (Cache, IO, External API) [T3] <!-- T3:auto -->

### 4.1 OpenRouter Client Integration
- [ ] Test: OpenRouter 요청 헤더에 인증 토큰이 포함된다.
- [ ] Test: 모델명이 비어 있으면 기본 모델(`qwen/qwen3.5-flash`)로 요청한다.
- [ ] Test: HTTP 타임아웃 30초가 클라이언트에 설정된다.
- [ ] Test: OpenRouter 응답이 200이고 JSON 형식이면 payload extractor가 호출된다.
- [ ] Test: OpenRouter 응답이 429면 retryable error로 매핑된다.
- [ ] Test: OpenRouter 응답이 500이면 retryable error로 매핑된다.
- [ ] Test: OpenRouter 응답이 4xx 비재시도 케이스면 non-retryable error로 매핑된다.

### 4.2 JSON Validation Integration
- [ ] Test: TOC payload가 JSON 객체/배열 구조를 따르지 않으면 schema error를 반환한다.
- [ ] Test: Metadata payload에 필수 필드 누락 시 schema error를 반환한다.
- [ ] Test: Review payload에서 `pros` 또는 `cons`가 배열이 아니면 schema error를 반환한다.
- [ ] Test: Courses payload에서 `curriculum`이 배열이 아니면 schema error를 반환한다.
- [ ] Test: Similar payload에서 아이템 수가 3~5 범위가 아니면 schema error를 반환한다.

### 4.3 Cache Persistence Integration
- [ ] Test: 캐시 디렉토리가 없으면 `~/.cache/bookinfo/`를 생성한다.
- [ ] Test: 캐시 저장 시 키별 파일이 생성된다.
- [ ] Test: 캐시 조회 시 파일이 존재하면 JSON 역직렬화 후 반환한다.
- [ ] Test: 캐시 파일이 손상되었으면 cache miss로 처리하고 수집 플로우를 계속한다.
- [ ] Test: 캐시 쓰기 권한 오류가 발생해도 수집 결과 자체는 사용자에게 반환한다.
- [ ] Test: `--no-cache` 시 캐시 읽기를 skip하지만, 수집 후 캐시에 덮어쓰기 저장한다.
- [ ] Test: 기본 모드와 전체 모드의 캐시가 별도 키로 저장된다.

### 4.4 File IO Integration
- [ ] Test: 출력 경로의 부모 디렉토리가 없으면 자동 생성한다.
- [ ] Test: JSON 파일 쓰기 후 다시 읽었을 때 구조가 동일하다.
- [ ] Test: UTF-8 한글 문자열(제목/번역)이 손실 없이 저장된다.

---

## Phase 5: Delivery Surface (CLI UX, Error Mapping, Logging) [T4] <!-- T4:auto -->

### 5.1 CLI User Experience
- [ ] Test: 기본 모드 성공 시 stdout에 제목/저자/목차1단계요약/저장경로가 출력된다.
- [ ] Test: 전체 모드 성공 시 stdout에 제목/저자/평점/목차1단계요약/저장경로가 출력된다.
- [ ] Test: 배치 실행 성공 시 항목별 상태와 최종 집계가 출력된다.
- [ ] Test: 배치 실행 시 진행 상황이 출력된다 (`[2/10] "Clean Code" 처리 중...`).
- [ ] Test: `--help` 출력에 모든 플래그(`--isbn`, `--full`, `--lang`, `--no-cache` 등)가 문서화된다.
- [ ] Test: 잘못된 플래그 입력 시 사용법 안내를 포함한 에러를 출력한다.
- [ ] Test: `--lang` 잘못된 값 입력 시 "허용값: ko, en, ja, zh-tw" 메시지가 출력된다.
- [ ] Test: 동명 도서 선택 시 stdout에 "동명 도서가 있을 수 있습니다" 힌트가 출력된다.

### 5.2 Error Mapping
- [ ] Test: API 키 누락 에러는 설정 방법 안내 메시지로 매핑된다.
- [ ] Test: 제목/ISBN 둘 다 없는 에러는 "제목 또는 --isbn 중 하나는 필수" 메시지로 매핑된다.
- [ ] Test: 책을 찾지 못한 응답은 경고 메시지와 함께 종료 코드 1로 매핑된다.
- [ ] Test: rate limit 에러는 백오프 재시도 후 최종 실패 메시지로 매핑된다.
- [ ] Test: JSON 검증 실패 에러는 "응답 구조 재시도 실패" 메시지로 매핑된다.
- [ ] Test: 네트워크 타임아웃은 "네트워크 오류" 카테고리 메시지로 매핑된다.

### 5.3 Logging / Observability
- [ ] Test: debug 로그 활성화 시 단계별 API 호출 시작/종료가 기록된다.
- [ ] Test: debug 로그에 수집 모드(basic/full)가 표시된다.
- [ ] Test: 로그 출력에 API 키, Authorization 헤더 값이 마스킹된다.
- [ ] Test: 배치 모드 로그에 각 항목 처리 시간(ms)이 포함된다.

---

## Phase 6: End-to-End / Regression / Performance [T4] <!-- T4:auto -->

### 6.1 E2E (Happy Path)
- [ ] Test: 단일 영어 도서 기본 모드 입력으로 메타+목차 JSON 생성이 성공한다.
- [ ] Test: 단일 영어 도서 전체 모드 입력으로 전체 JSON 생성이 성공한다.
- [ ] Test: ISBN-13 입력으로 기본 모드 JSON 생성이 성공한다.
- [ ] Test: 단일 한국어 도서 입력으로 원어+한국어 병기 필드가 유지된다.
- [ ] Test: 일본어 또는 번체중국어 입력으로도 동일 스키마가 유지된다.
- [ ] Test: 기본 모드 출력에서 `review`/`related_courses`/`similar_books`가 빈 값이고 `data_availability`가 false이다.

### 6.2 E2E (Failure Path)
- [ ] Test: API 키가 없는 환경에서 명확한 실패 메시지와 비정상 종료 코드를 반환한다.
- [ ] Test: API 429를 강제한 시뮬레이션에서 최대 3회 재시도 후 실패 처리된다.
- [ ] Test: JSON 비정상 응답 시 프롬프트 보정 재시도 3회 후 실패 처리된다.
- [ ] Test: 배치 파일 3건 중 1건 실패 케이스에서 2건 결과 파일은 정상 생성된다.
- [ ] Test: `--lang xyz` 입력 시 명확한 에러 메시지와 비정상 종료 코드를 반환한다.

### 6.3 Regression / Non-Functional
- [ ] Test: 동일 입력 2회 실행 시 2회차가 캐시 히트 경로로 단축된다.
- [ ] Test: `--no-cache` 실행 시 기존 캐시가 있어도 API 호출 경로를 사용한다.
- [ ] Test: `--no-cache` 실행 후 다음 조회 시 갱신된 캐시가 반환된다.
- [ ] Test: 기본 모드 캐시와 전체 모드 캐시가 독립적으로 관리된다.
- [ ] Test: 배치 20건 샘플 실행에서 순차 처리 규칙이 깨지지 않는다.
- [ ] Test: stdout 요약 포맷이 릴리스 기준 스냅샷과 일치한다.

### 6.4 Performance Guardrail
- [ ] Test: 모의 API 환경에서 기본 모드 orchestration 오버헤드가 500ms 이내다.
- [ ] Test: 모의 API 환경에서 전체 모드 orchestration 오버헤드가 500ms 이내다.
- [ ] Test: 캐시 히트 경로의 파일 I/O + 직렬화 시간이 100ms 이내다.

---

## Tier Summary
- T1 (Phase 1): CLI/계약/DTO 스캐폴딩
- T2 (Phase 2-3): 핵심 규칙 및 오케스트레이션 (Deep Review 대상)
- T3 (Phase 4): 외부 API/캐시/파일 통합
- T4 (Phase 5-6): CLI 전달 계층 및 E2E 회귀

## Completion Log

| Date | Phase | Test Item | Status | Commit | Notes |
|------|-------|-----------|--------|--------|-------|
| YYYY-MM-DD | P1 | 예: CLI title required | TODO | - | - |

## Notes

### Business Rules (Confirmed)
- 기본 모드: 메타데이터 + 목차 수집 (API 2회 호출)
- 전체 모드(`--full`): 메타 + 목차 + 후기 + 강의 + 유사도서 (API 5회 호출)
- 검색: 제목 또는 ISBN-13으로 가능 (동시 입력 시 ISBN 우선)
- `--no-cache`: 읽기만 우회, 새 결과는 캐시에 덮어쓰기 저장
- `--lang` 허용값: `ko`, `en`, `ja`, `zh-tw` (고정)
- 동명 도서: 인기 우선, 동저자 시 최신판 선택
- 파일명 날짜: 로컬 타임존 / JSON `collected_at`: UTC
- 데이터 없음: 빈 배열/객체 + `data_availability` 표시
- 백오프: 초기 2s, 배수 2x, jitter 0~1s, 캡 30s, 최대 3회
- 배치: 항목 간 고정 3초, 429 시 2배 증가(최대 30s), 3연속 성공 시 복원
- 캐시 키: 입력(제목/ISBN) + 저자 + 언어 + 모델 + 모드 해시
- 캐시는 `~/.cache/bookinfo/`를 사용한다.
- 배치 처리는 순차 처리이며 부분 실패를 허용한다.
- API 키는 환경변수 기본, `--api-key`로 오버라이드 가능하다.

### Out of Scope
- 웹 스크래핑 기반 실데이터 수집
- GUI/웹 인터페이스
- DB 영속화 및 검색 인덱스
- 실시간 가격 비교/구매링크 검증
- ISBN 유효성 체크섬 검증 (모델에 위임)

### Open Issues (Ambiguity Interview Needed)
*모든 항목 해결 완료 — open issue 없음*

## Execution Entry Order (Recommended)
1. Phase 1 전 항목 완료
2. Phase 2 규칙 검증 완료
3. Phase 3 오케스트레이션 완료
4. Phase 4 통합 테스트 완료
5. Phase 5 CLI 전달 계층 완료
6. Phase 6 E2E/회귀/성능 가드레일 완료

## Final DoD (Definition of Done)

### A. 기능 완성 기준
- [ ] CLI가 제목 또는 `--isbn` 입력을 정상 처리하고, 동시 입력 시 ISBN 우선 규칙을 따른다.
- [ ] `--full` 미사용 시 기본 모드(메타+목차, API 2회), 사용 시 전체 모드(API 5회)로 정확히 분기한다.
- [ ] `--lang`은 `ko`, `en`, `ja`, `zh-tw`만 허용하고, 잘못된 값에 대해 명확한 오류를 반환한다.
- [ ] `--no-cache`는 캐시 읽기만 우회하고, 새 결과를 캐시에 저장한다.
- [ ] 캐시 키가 입력값(제목/ISBN)+저자+언어+모델+모드 조합으로 생성된다.
- [ ] 기본 모드 출력에서 `review`/`related_courses`/`similar_books`는 빈 값으로 유지되고 `data_availability`가 false를 반영한다.
- [ ] 전체 모드 출력에서 후기/강의/유사도서를 수집·병합하고, 데이터 부재 시 빈 값 정책을 유지한다.
- [ ] 동명 도서 선택 기준(인기 우선, 동저자 시 최신판)이 프롬프트와 `selection_note`에 반영된다.
- [ ] 출력 파일 기본명은 로컬 타임존 날짜(`yyyymmdd`)를 사용하고, JSON `collected_at`은 UTC RFC3339을 사용한다.

### B. 안정성/복원력 기준
- [ ] API 429/500에 대해 백오프(2s, 4s, 8s + jitter, cap 30s) 재시도 후 최대 3회에서 종료한다.
- [ ] JSON 파싱/스키마 실패 시 프롬프트 보정 재시도를 수행하고 최대 3회 실패 시 명확한 오류를 반환한다.
- [ ] 네트워크 타임아웃(30초) 및 외부 API 오류가 정의된 도메인 오류로 매핑된다.
- [ ] 캐시 파일 손상/캐시 쓰기 실패 상황에서도 프로세스가 비정상 종료되지 않고 결과 제공 정책을 지킨다.
- [ ] 배치 처리에서 항목별 실패를 격리하고 나머지 항목 처리를 계속한다.

### C. 배치/처리 정책 기준
- [ ] 배치 항목이 순차 처리되며 기본 간격 3초 정책이 적용된다.
- [ ] 배치 중 429 발생 시 항목 간 간격이 2배 증가(최대 30초)한다.
- [ ] 3연속 성공 시 배치 간격이 기본값(3초)으로 복원된다.
- [ ] 배치 완료 시 성공/실패 집계와 항목별 상태 로그가 출력된다.

### D. 테스트/품질 게이트 기준
- [ ] Phase 1~6 체크리스트 항목이 모두 완료(`- [x]`)된다.
- [ ] `go test -v ./... -count=1`가 통과한다.
- [ ] 통합 경로가 있는 경우 `go test -v -tags=integration ./... -count=1`가 통과한다.
- [ ] `go vet ./...` 경고/오류가 없고, 포맷 명령(`gofmt`) 적용 후 추가 diff가 없다.
- [ ] 기본 모드/전체 모드/ISBN 모드의 핵심 E2E 시나리오가 모두 통과한다.

### E. UX/문서/보안 기준
- [ ] `--help`에 모든 플래그(`--isbn`, `--full`, `--lang`, `--no-cache` 등)와 동작이 최신 요구사항대로 문서화된다.
- [ ] stdout 요약 포맷이 스냅샷 기준과 일치하고, 오류 메시지는 사용자 행동 가이드를 포함한다.
- [ ] 로그/오류 출력에서 API 키 및 인증 헤더가 완전히 마스킹된다.
- [ ] `bookinfo_plan.md`와 `bookinfo_tdd_plan.md`의 기능/정책/스키마 설명이 상호 일치한다.
