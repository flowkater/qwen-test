# bookinfo

AI 기반 도서/강의 정보 수집 CLI 도구. DashScope API (Qwen 모델)를 사용하여 도서 메타데이터, 목차, 리뷰, 관련 강의, 유사 도서 정보를 자동으로 수집합니다.

## 설치

```bash
# 빌드
make build

# 또는 직접
go build -o bin/bookinfo ./cmd/bookinfo
```

## 환경 변수

| 변수 | 설명 | 필수 |
|------|------|------|
| `DASHSCOPE_API_KEY` | DashScope API 키 | O |
| `BOOKINFO_HTTP_TIMEOUT_SEC` | HTTP 타임아웃 (초, 기본 90) | X |

`.env` 파일에 설정하면 Makefile에서 자동 로드합니다.

```bash
# .env
DASHSCOPE_API_KEY=sk-xxxxxxxxxxxx
```

## 사용법

### 기본: 제목으로 조회

```bash
# 기본 (메타데이터만, JSON 출력)
bookinfo "Clean Code"

# 언어 지정
bookinfo "Clean Code" --lang en
bookinfo "클린 코드" --lang ko

# 풀모드 (메타데이터 + 목차 + 리뷰 + 관련 강의 + 유사 도서)
bookinfo "Clean Code" --full

# 텍스트 출력
bookinfo "Clean Code" --format text

# 저자 힌트
bookinfo "Clean Code" --author "Robert C. Martin"

# 출력 파일 지정
bookinfo "Clean Code" --output result.json

# 캐시 무시 (항상 새로 수집)
bookinfo "Clean Code" --no-cache
```

### ISBN-13으로 조회

ISBN 조회 시 `--country` 플래그가 필수입니다.

```bash
bookinfo --isbn 978-0132350884 --country us
bookinfo --isbn 978-0132350884 --country kr --lang ko
bookinfo --isbn 978-0132350884 --country us --full
```

### 강의/코스 조회

강의 모드는 `--lecture`, `--url`, `--country` 플래그가 필수입니다.

```bash
bookinfo --lecture --url "https://www.inflearn.com/course/..." --country kr --lang ko
```

### 배치 처리

텍스트 파일에 한 줄에 하나씩 제목 또는 ISBN-13을 작성합니다. ISBN-13은 자동 감지됩니다 (978/979로 시작하는 13자리 숫자).

```
Clean Code
978-0132350884
Designing Data-Intensive Applications
978-0134685991
리팩터링
```

```bash
# 직렬 처리 (기본)
bookinfo --batch books.txt

# 병렬 처리 (5개 동시)
bookinfo --batch books.txt --workers 5

# 병렬 + 풀모드 + 출력 디렉토리
bookinfo --batch books.txt --workers 3 --full --output ./results/

# 병렬 + 캐시 무시 + 국가 지정 (ISBN 배치에 필요)
bookinfo --batch isbns.txt --workers 3 --no-cache --country us
```

#### 배치 병렬 처리 상세

- `--workers N`: N개의 고루틴이 동시에 API를 호출합니다
- 글로벌 레이트 리미터가 API 호출 간 최소 200ms 간격을 유지하여 DashScope 버스트 감지를 방지합니다
- 결과는 입력 순서를 유지합니다
- 일부 항목이 실패해도 나머지는 정상 처리됩니다
- `--workers`를 생략하면 기존과 동일한 직렬 처리 (adaptive backoff 적용)

#### 배치 파일 자동 분류 규칙

| 입력 패턴 | 분류 | 예시 |
|-----------|------|------|
| 978/979로 시작, 하이픈 제거 시 13자리 숫자 | ISBN-13 | `978-0132350884`, `9780132350884` |
| 그 외 | 제목(Title) | `Clean Code`, `클린 코드` |

## 플래그 전체 목록

| 플래그 | 단축 | 설명 | 기본값 |
|--------|------|------|--------|
| `<title>` | - | 도서 제목 (위치 인자) | - |
| `--isbn` | - | ISBN-13 (`--country` 필수) | - |
| `--author` | `-a` | 저자 힌트 | - |
| `--lang` | `-l` | 응답 언어 (`ko`, `en`, `ja`, `zh-tw`) | - |
| `--full` | `-f` | 풀모드 (메타+목차+리뷰+강의+유사서) | false |
| `--format` | - | 출력 형식 (`json`, `text`) | `json` |
| `--api-key` | - | DashScope API 키 (환경변수 대체) | `$DASHSCOPE_API_KEY` |
| `--model` | `-m` | AI 모델 | `qwen3.5-flash` |
| `--output` | `-o` | 출력 파일/디렉토리 경로 | stdout |
| `--batch` | `-b` | 배치 파일 경로 (줄당 제목/ISBN) | - |
| `--workers` | - | 배치 병렬 워커 수 (양의 정수) | 1 (직렬) |
| `--no-cache` | - | 캐시 읽기 건너뛰기 | false |
| `--country` | - | 국가 코드 (`us`, `kr`, `jp`, `tw`) | - |
| `--lecture` | - | 강의 모드 (`--url`, `--country` 필수) | false |
| `--url` | - | 강의 사이트 URL | - |
| `--help` | `-h` | 도움말 출력 | - |

## Makefile 사용법

```bash
# 빌드 & 테스트
make build
make test
make clean

# 제목 기반
make run TITLE='Clean Code'
make run-en TITLE='Clean Code'
make run-kr TITLE='클린 코드'
make run-text TITLE='Clean Code'
make run-full TITLE='Clean Code'
make run-full-en TITLE='Clean Code'
make run-full-kr TITLE='클린 코드'
make run-nocache TITLE='Clean Code'
make run-full-nocache TITLE='Clean Code'

# ISBN 기반
make run-isbn ISBN='978-0132350884'
make run-isbn-en ISBN='978-0132350884'
make run-isbn-kr ISBN='978-0132350884'
make run-isbn-us ISBN='978-0132350884'

# 강의
make run-lecture URL='https://...' COUNTRY=kr
make run-lecture-kr URL='https://www.inflearn.com/course/...'

# 배치
make run-batch BATCH='titles.txt'
make run-batch BATCH='titles.txt' WORKERS=5
make run-batch BATCH='mixed.txt' WORKERS=3 COUNTRY=us FULL=1

# 조합 변수 (모든 run-* 타겟에 추가 가능)
#   L=en|ko|ja|zh-tw       언어
#   AUTHOR='Robert Martin'  저자
#   MODEL='qwen-plus'       모델
#   OUTPUT='out.json'        출력 경로
#   FORMAT=json|text         출력 형식
#   NOCACHE=1                캐시 무시
#   FULL=1                   풀모드
#   COUNTRY=us|kr|jp|tw      국가
#   URL='https://...'        강의 URL
#   WORKERS=N                병렬 워커 수
#   TIMEOUT_SEC=180          HTTP 타임아웃 (기본 120)
#   ARGS='--extra-flag'      추가 플래그

# 조합 예시
make run TITLE='Clean Code' L=en AUTHOR='Robert C. Martin' NOCACHE=1
make run TITLE='클린 코드' L=ko FORMAT=text FULL=1
make run-batch BATCH='isbns.txt' WORKERS=5 COUNTRY=us NOCACHE=1
```

## 에이전트/프로그래매틱 사용 가이드

이 섹션은 AI 에이전트나 스크립트에서 bookinfo를 프로그래매틱하게 호출할 때 필요한 정보입니다.

### 빠른 시작 체크리스트

```bash
# 1. 환경변수 확인
test -n "$DASHSCOPE_API_KEY" && echo "OK" || echo "DASHSCOPE_API_KEY 미설정"

# 2. 빌드
make build   # → bin/bookinfo 생성

# 3. 단건 실행 (JSON → stdout, 요약 → stderr)
bin/bookinfo "Clean Code" --format json 2>/dev/null

# 4. 종료 코드 확인
echo $?   # 0=성공, 1=실패
```

### Exit Codes

| 코드 | 의미 |
|------|------|
| `0` | 성공 (단건: 정상 수집, 배치: 전체 성공, `--help`) |
| `1` | 실패 (플래그 오류, API 키 없음, API 에러, 배치 중 1건 이상 실패) |

### stdout vs stderr 분리

| 스트림 | 내용 |
|--------|------|
| **stdout** | JSON/Text 결과 데이터 (`--output` 미지정 시), 배치 진행 로그 |
| **stderr** | 에러 메시지 (`error: ...`), 도움말 텍스트 |

에이전트가 JSON만 파싱하려면:

```bash
# JSON만 캡처 (요약/에러는 stderr로 분리됨)
RESULT=$(bin/bookinfo "Clean Code" 2>/dev/null)
echo "$RESULT" | jq '.book.title.original'

# 파일로 저장 후 파싱 (권장)
bin/bookinfo "Clean Code" --output result.json 2>/dev/null
jq '.book.isbn13' result.json
```

### 단건 실행 — 에이전트 패턴

```bash
# 제목으로 풀모드 수집 → 파일 저장
bin/bookinfo "Clean Code" --full --lang en --output clean_code.json 2>/dev/null
if [ $? -eq 0 ]; then
  # 성공: JSON 파싱
  jq '.book.title.original' clean_code.json
  jq '.table_of_contents | length' clean_code.json
  jq '.review.rating' clean_code.json
else
  echo "수집 실패" >&2
fi

# ISBN으로 수집 (--country 필수)
bin/bookinfo --isbn 978-0132350884 --country us --full --output result.json 2>/dev/null
```

### 배치 실행 — 에이전트 패턴

```bash
# 1. 배치 파일 동적 생성
cat > /tmp/books.txt << 'EOF'
Clean Code
978-0132350884
Designing Data-Intensive Applications
EOF

# 2. 병렬 실행 (출력 디렉토리 지정)
mkdir -p /tmp/results
bin/bookinfo --batch /tmp/books.txt --workers 3 --full \
  --country us --output /tmp/results/ 2>/dev/null

# 3. 결과 확인 (exit code 1 = 일부 실패)
if [ $? -eq 0 ]; then
  echo "전체 성공"
else
  echo "일부 실패 — 개별 파일 확인 필요"
fi

# 4. 개별 JSON 파일 파싱
for f in /tmp/results/*.json; do
  echo "$(jq -r '.book.title.original' "$f"): $(jq '.review.rating' "$f")"
done
```

### 배치 stdout 출력 형식

배치 실행 시 stdout에 진행 로그가 출력됩니다:

```
[1/3] "Clean Code" 처리 중...
  ✅ success: /tmp/results/clean_code_20260305.json
[2/3] "978-0132350884" 처리 중...
  ✅ success: /tmp/results/978-0132350884_20260305.json
[3/3] "Designing Data-Intensive Applications" 처리 중...
  ❌ failed: retryable error (code=429): rate limit exceeded
Batch summary: success=2 failed=1
```

마지막 줄 `Batch summary: success=N failed=M`으로 결과를 파싱할 수 있습니다.

### JSON 출력 스키마 요약

| 필드 | 타입 | basic 모드 | full 모드 | 설명 |
|------|------|-----------|----------|------|
| `book` | object | O | O | 메타데이터 (title, author, isbn13, pages 등) |
| `book.title` | `{original, korean}` | O | O | 원제 + 한국어 제목 |
| `table_of_contents` | array | `[]` | O | 목차 (depth, children 트리 구조) |
| `review` | object | 빈값 | O | rating(0-5), pros/cons, recommended_level |
| `related_courses` | array | `[]` | O | 관련 강의 (title, platform, url 등) |
| `similar_books` | array | `[]` | O | 유사 도서 (title, author, difficulty_comparison) |
| `data_availability` | object | O | O | 각 섹션 데이터 존재 여부 (boolean) |
| `metadata` | object | O | O | collected_at, model, source, mode, query |

`data_availability`로 어떤 섹션이 실제로 채워졌는지 확인하세요:

```bash
jq '.data_availability' result.json
# {"table_of_contents": true, "review": true, "related_courses": true, "similar_books": true}
```

### 에러 메시지 매핑

| stderr 메시지 | 원인 | 해결 |
|--------------|------|------|
| `API 키가 없습니다` | `DASHSCOPE_API_KEY` 미설정 | 환경변수 또는 `--api-key` 설정 |
| `제목 또는 --isbn 중 하나는 필수입니다` | 입력 누락 | title 위치 인자 또는 `--isbn` 추가 |
| `--country is required` | ISBN/lecture 모드에서 country 누락 | `--country us\|kr\|jp\|tw` 추가 |
| `--workers must be a positive integer` | workers 값 오류 | 1 이상의 정수 사용 |
| `허용값: ko, en, ja, zh-tw` | 잘못된 lang | 지원 언어 사용 |
| `허용값: json, text` | 잘못된 format | `json` 또는 `text` 사용 |
| `retryable error (code=429)` | API rate limit | workers 줄이거나 재시도 |
| `retryable error (code=408)` | API 타임아웃 | `BOOKINFO_HTTP_TIMEOUT_SEC` 증가 |
| `책을 찾지 못했습니다` | 도서 미발견 | 제목/ISBN 확인, `--author` 추가 |

### 캐시 활용 팁

- 동일 쿼리 반복 호출 시 캐시가 자동으로 사용됩니다 (즉시 반환)
- 에이전트가 최신 데이터를 원하면 `--no-cache` 사용
- 캐시 키: `{title|isbn}_{model}_{mode}` 조합
- 캐시 위치: `~/.cache/bookinfo/`
- 캐시를 초기화하려면: `rm -rf ~/.cache/bookinfo/`

### Rate Limiting 주의사항

- 병렬 실행 시 글로벌 레이트 리미터가 API 호출 간 최소 **200ms** 간격을 자동 유지
- DashScope qwen3.5-flash 국제 리전: 15,000 RPM / 5,000,000 TPM
- `--workers 3~5`가 안정적인 병렬 수준 (풀모드에서 내부적으로 여러 API 호출 발생)
- 429 에러 발생 시 자동 재시도 (exponential backoff, 최대 3회)

## 출력 형식

### JSON (기본)

```json
{
  "book": {
    "title": { "original": "Clean Code", "korean": "클린 코드" },
    "author": "Robert C. Martin",
    "publisher": "Prentice Hall",
    "published_date": "2008-08-01",
    "isbn13": "978-0132350884",
    "pages": 464,
    "language": "en",
    "edition": "1st",
    "selection_note": "..."
  },
  "table_of_contents": [],
  "review": { "rating": 0, "summary": { "pros": [], "cons": [] }, "recommended_level": "", "prerequisites": [] },
  "related_courses": [],
  "similar_books": [],
  "data_availability": { "table_of_contents": false, "review": false, "related_courses": false, "similar_books": false },
  "metadata": {
    "collected_at": "2026-03-05T12:00:00Z",
    "model": "qwen3.5-flash",
    "source": "dashscope",
    "mode": "basic",
    "query": { "title": "Clean Code", "isbn13": "", "author": "", "lang": "" }
  }
}
```

`--full` 모드에서는 `table_of_contents`, `review`, `related_courses`, `similar_books` 필드가 채워집니다.

### Text

`--format text`로 사람이 읽기 쉬운 텍스트 형식으로 출력합니다.

## 캐시

- 캐시 디렉토리: `~/.cache/bookinfo/`
- 동일한 쿼리(제목/ISBN + 모델 + 모드)에 대해 캐시된 결과를 반환합니다
- `--no-cache`: 캐시 읽기를 건너뛰되, 새로 수집한 결과는 캐시에 저장합니다

## 프로젝트 구조

```
cmd/bookinfo/          # CLI 진입점
internal/
  app/                 # 비즈니스 로직 (Service, prompts)
  cli/                 # CLI 파싱, 실행
  domain/              # 도메인 타입, 검증, 에러
  infra/
    cache/             # 파일 기반 캐시
    openrouter/        # DashScope API 클라이언트
    output/            # JSON/Text 출력
    validator/         # JSON 검증
  integration/         # 통합 테스트
```

## 라이선스

Private
