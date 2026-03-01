# Phase 3-4 Results (Orchestration / Integration)

## Implemented
- Application orchestration service (`internal/app/service.go`)
  - basic mode flow: Metadata → TOC
  - full mode flow: Metadata → TOC → Review → Courses → SimilarBooks
  - 중간 단계 실패 시 이후 단계 중단
  - merge single-shot
- Cache policy:
  - cache hit short-circuit
  - `--no-cache` read skip + write 유지
  - corrupted cache => cache miss 처리
- Retry/backoff integration:
  - retryable(429/5xx) retry
  - jitter(0~1s) 적용
  - validation 실패 재시도 후 `ErrInvalidResponse`
- Prompt separation:
  - TOC/Metadata/Review/Courses/Similar 전용 prompt builder
  - 원어+한국어, 동명도서 선택 기준 포함
- Output orchestration:
  - default filename `{identifier}_{yyyymmdd}.json` (local)
  - parent dir auto-create
  - JSON UTF-8 persistence
- Batch orchestration:
  - empty line skip
  - sequential processing
  - base delay 3s
  - 429 후 delay x2 (max 30s)
  - 3연속 성공 시 base reset
  - partial-failure continue
  - 중복/정규화 충돌 제목 포함 시에도 suffix(`-2`, `-3`)로 항목별 고유 파일 경로 생성(덮어쓰기 방지)
- Integration adapters:
  - OpenRouter HTTP client + auth header + status mapping(404/semantic not-found 매핑 포함)
  - JSON validator (metadata/toc/review/courses/similar)
  - file cache persistence
  - JSON output writer

## Changed Files
- `internal/app/*`
- `internal/infra/openrouter/*`
- `internal/infra/cache/*`
- `internal/infra/validator/*`
- `internal/infra/output/*`
- Tests:
  - `internal/app/service_test.go`
  - `internal/infra/**/_test.go`

## Commands + Results
- `go test ./...` ✅
- `go test -tags=integration ./...` ✅
- `go vet ./...` ✅
- `go build ./...` ✅

## Review Findings
- Critical: none
- Major: none
- Minor:
  - 실제 external API E2E smoke(실토큰) 테스트는 환경 의존으로 수동 권장

## Follow-up TODO
- logging verbosity와 사용자 메시지 카테고리 추가 미세조정
