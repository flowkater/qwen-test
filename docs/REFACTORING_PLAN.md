# bookinfo CLI 대규모 리팩토링 계획

> 작성일: 2026-03-04
> 관련 이슈: docs/KNOWN_ISSUES.md

## Context

KNOWN_ISSUES.md에서 발견된 3가지 핵심 문제를 해결한다:
1. 책/강의 구분 없는 단일 프롬프트 → 파라미터 기반 분리
2. enrichTOCParts 순차 호출 → 8개 sparse chapter 타임아웃
3. LLM 비결정성 → 품질 게이트 부재로 sparse 결과 허용

---

## Feature A: 책/강의 프롬프트 분리 + 새 CLI 파라미터

### A.1 BookQuery 확장 (`internal/domain/types.go`)
- `Country string` 추가 (us|kr|jp|tw)
- `Lecture bool` 추가 (--lecture 플래그)
- `URL string` 추가 (강의 사이트 URL)
- `QualityRetry bool` 추가 (내부 전용, CLI 미노출)
- `QueryType() string` 메서드 추가 → "book" 또는 "lecture"

### A.2 검증 강화 (`internal/domain/query.go`, `errors.go`)
- `allowedCountry` 맵: us, kr, jp, tw
- ISBN 모드: `--country` 필수
- Lecture 모드: `--url` + `--country` 필수
- 에러: `ErrMissingURL`, `ErrMissingCountry`, `ErrInvalidCountry`

### A.3 CLI 파서 (`internal/cli/parse.go`)
- `--country`, `--lecture`, `--url` 파싱 추가
- HelpText 업데이트

### A.4 시스템 프롬프트 분리 (`internal/app/prompts.go`)
- `SystemPromptBook`: ISBN/출판사/목차 중심, web_search로 실제 TOC 검색
- `SystemPromptLecture`: URL 기반 커리큘럼 추출, web_extractor 우선, depth 2 제한
- `GetSystemPrompt(q BookQuery) string`: q.Lecture에 따라 분기

### A.5 유저 프롬프트 분리 (`internal/app/prompts.go`)
- `commonLead` → `bookLead` / `lectureLead` 분기
- `BuildUnifiedPrompt` → `buildBookUnifiedPrompt` / `buildLectureUnifiedPrompt` 분기
- 강의: URL을 primary source로, web_extractor 사용 지시
- 책: ISBN + country로 검색 범위 명시

### A.6 Collector 수정 (`internal/infra/openrouter/collector.go`)
- 모든 Collect* 메서드: `app.GetSystemPrompt()` → `app.GetSystemPrompt(q)` 변경

### A.7 CacheKey 수정 (`internal/domain/cache.go`)
- Country, Lecture 플래그 해시에 포함 (QualityRetry는 제외)

### A.8 Makefile 타겟 추가
```makefile
run-isbn-kr: --isbn + --country kr
run-isbn-us: --isbn + --country us
run-lecture: --lecture --url URL (COUNTRY 필수)
run-lecture-kr: --lecture --url --country kr --lang ko
```

---

## Feature B: enrichTOCParts 병렬화

### B.1 Service 필드 추가 (`internal/app/service.go`)
- `EnrichConcurrency int` (기본값: 3)

### B.2 enrichTOCParts 재작성 (`internal/app/service.go`)
- stdlib `sync.WaitGroup` + semaphore channel 패턴
- 외부 의존성 없음 (errgroup은 golang.org/x/sync)
- goroutine당 semaphore 획득 → collectPartTOC → result channel 전송
- ctx.Done() 감시로 취소 지원
- 실패 시 기존 결과 유지 (현재와 동일)

```go
sem := make(chan struct{}, maxConcurrency)  // 동시성 제한
results := make(chan result, len(sparse))
// goroutine fan-out → wg.Wait() → close(results) → range results 수집
```

### B.3 sparse part 수 제한
- 최대 5개까지만 enrichment (나머지는 unified 결과 유지)

---

## Feature C: 품질 게이트 + 재현성 향상

### C.1 품질 판정 함수 (`internal/app/service.go`)
```go
func tocQualityOK(toc []TOCNode, minTopLevel int) bool
```
- depth-1 노드 수 ≥ minTopLevel
- 최소 절반은 children 보유

### C.2 Service 필드 추가
- `MinTOCChapters int` (기본: 책=3, 강의=2)

### C.3 품질 재시도 루프 (`internal/app/service.go`)
```
collectUnifiedWithQualityRetry:
  attempt 0: 정상 호출
  attempt 1-2: QualityRetry=true → 강화 프롬프트
  최대 3회 시도 후 마지막 결과 반환
```

### C.4 프롬프트 강화 (quality retry 시)
- "QUALITY RETRY" 접미사 추가
- 복수 web_search 쿼리 지시
- 최소 chapter/section 수 명시

### C.5 Temperature 0.0 + Seed (`internal/infra/openrouter/client.go`)
- temperature: 0.1 → 0.0
- RequestPayload에 `Seed *int` 추가 (seed=42)
- DashScope가 무시하면 무해, 지원하면 재현성 향상

---

## 구현 순서

1. **Commit 1 — Feature A**: types → errors → query → parse → prompts → collector → cache → Makefile
2. **Commit 2 — Feature B**: service.go 병렬화
3. **Commit 3 — Feature C**: quality gate + temperature/seed

## 수정 대상 파일

| 파일 | Feature |
|------|---------|
| `internal/domain/types.go` | A (BookQuery 확장) |
| `internal/domain/errors.go` | A (새 에러 상수) |
| `internal/domain/query.go` | A (검증 강화) |
| `internal/domain/cache.go` | A (CacheKey 수정) |
| `internal/cli/parse.go` | A (새 플래그 파싱) |
| `internal/app/prompts.go` | A, C (프롬프트 분리 + 품질 재시도) |
| `internal/app/service.go` | B, C (병렬화 + 품질 게이트) |
| `internal/app/interfaces.go` | — (변경 없음) |
| `internal/infra/openrouter/collector.go` | A (GetSystemPrompt 시그니처) |
| `internal/infra/openrouter/client.go` | C (temperature + seed) |
| `Makefile` | A (새 타겟) |

## 검증 방법

1. `go test ./...` — 전체 단위 테스트 통과
2. `make build` — 빌드 성공
3. `make run-isbn-us ISBN='978-0132350884' L=en` — Clean Code 책 테스트
4. `make run-lecture-kr URL='https://www.inflearn.com/course/...'` — 강의 테스트
5. 병렬화 확인: 8개 sparse chapter 책에서 타임아웃 없이 완료
6. 품질 게이트: NOCACHE로 실행 시 최소 chapter 수 보장

---

## 하위 호환성

- 기존 make 타겟(`run`, `run-en`, `run-kr`, `run-isbn`, `run-batch` 등) 변경 없음
- `--country`는 `--isbn` 또는 `--lecture` 사용 시에만 필수
- 타이틀 기반 실행(`make run TITLE='Clean Code'`)은 기존과 동일
- CacheKey 변경으로 기존 캐시 미스 발생 (수동 정리 가능)
- temperature 0.1→0.0: 미세한 동작 변경, 테스트 영향 없음
