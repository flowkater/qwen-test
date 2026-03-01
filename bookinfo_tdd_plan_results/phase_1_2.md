# Phase 1-2 Results (Contract / DTO / Domain Rules)

## Implemented
- Go module bootstrap (`go.mod`) and layered structure (`internal/domain`, `internal/cli`, `internal/app`, `internal/infra`, `cmd/bookinfo`).
- CLI argument contract parsing:
  - title positional
  - `--isbn`, `--author|-a`, `--lang|-l`, `--full|-f`, `--api-key`, `--model|-m`, `--output|-o`, `--batch|-b`, `--no-cache`
  - title+isbn 동시 입력 시 ISBN 우선 식별자 사용
- Query validation rules:
  - title/isbn/batch 필수 조건
  - lang allowlist (`ko`, `en`, `ja`, `zh-tw`)
  - output path directory 차단
- Config/security contract:
  - API key 우선순위 (`--api-key` > env)
  - missing key 가이드 에러
  - secret masking helper
- DTO/schema contract:
  - root fields serialization
  - `TOCNode.children` empty array 보장
  - `SimilarBook.difficulty_comparison` 보장
  - `CollectMetadata.collected_at` UTC RFC3339 직렬화
- Domain rules:
  - cache key hashing (identifier+author+lang+model+mode)
  - backoff (2s, 4s, 8s... + jitter + 30s cap)
  - merge/basic/full consistency + data availability
  - TOC depth/title integrity checks

## Changed Files
- `go.mod`
- `internal/domain/*`
- `internal/cli/parse.go`
- Tests:
  - `internal/domain/*_test.go`
  - `internal/cli/parse_test.go`

## Commands + Results
- `go test ./...` ✅
- `go test -tags=integration ./...` ✅
- `go vet ./...` ✅
- `go build ./...` ✅

## Review Findings
- Critical: none
- Major: none
- Minor:
  - plan checklist item granularity(세부 항목별 1:1 테스트 명)는 추후 추가 가능

## Follow-up TODO
- 일부 세부 UX/logging 표현 강화 (Phase 5 영역)
