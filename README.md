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
