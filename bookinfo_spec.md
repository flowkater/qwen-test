# BookInfo CLI - 요구사항 명세서 & 구현 계획

## Overview

OpenRouter API(QWEN3.5-flash 기본)를 사용하여 전 세계(미국, 한국, 일본, 대만 등) 출판 도서의 상세 목차, 메타데이터, 후기, 동영상 강의 정보, 유사 도서 추천을 자동 수집하는 Go CLI 도구.

## Problem Statement

학습 커리큘럼 설계와 콘텐츠 기획/리서치 시, 다국적 도서의 상세 목차·메타데이터·관련 강의 정보를 일일이 AI에 수동 질문하여 수집하는 과정이 반복적이고 비효율적이다. 이를 자동화하여 한 번의 CLI 명령으로 구조화된 JSON 데이터를 생성한다.

## Target Users

- 주 사용자: 본인 (개인 도구)
- 역할: 학습 커리큘럼 설계자, 콘텐츠 기획자
- Go 중급 수준

---

## Requirements

### Functional Requirements

#### FR-1: CLI 인터페이스
- [ ] 책 제목을 필수 인자로 받음 (positional argument)
- [ ] `--isbn`: ISBN-13으로 검색 (제목 대신 사용 가능, 상호 배타적이지 않음)
- [ ] `--author` / `-a`: 저자명 지정 (선택)
- [ ] `--lang` / `-l`: 대상 언어/국가 지정 (선택, 허용값: `ko`, `en`, `ja`, `zh-tw`)
- [ ] `--full` / `-f`: 전체 데이터 수집 모드 (기본 off → 메타+목차만, on → 후기+강의+유사도서 포함)
- [ ] `--api-key`: API 키 직접 전달 (선택, 환경변수 오버라이드)
- [ ] `--model` / `-m`: LLM 모델 선택 (기본값: `qwen/qwen3.5-flash`)
- [ ] `-o` / `--output`: 출력 파일 경로 지정 (기본값: `{제목}_{날짜}.json`, 날짜는 로컬 타임존)
- [ ] `--batch` / `-b`: 배치 파일 경로 (텍스트 파일에서 여러 책 제목 읽기)
- [ ] `--no-cache`: 캐시 읽기 우회 (새 결과는 캐시에 저장됨)

#### FR-1.1: 검색 입력 방식
- [ ] 제목 검색: `bookinfo "Clean Code"`
- [ ] ISBN-13 검색: `bookinfo --isbn 978-0132350884`
- [ ] 제목 + 저자 복합: `bookinfo "Clean Code" --author "Robert C. Martin"`
- [ ] ISBN + 추가 옵션: `bookinfo --isbn 978-0132350884 --full`
- [ ] 제목과 `--isbn` 동시 입력 시 ISBN 우선

#### FR-1.2: 수집 모드
- [ ] **기본 모드** (default): 메타데이터 + 목차만 수집 (API 2회 호출)
- [ ] **전체 모드** (`--full`): 메타데이터 + 목차 + 후기 + 강의 + 유사도서 수집 (API 5회 호출)

#### FR-2: 데이터 수집 (분리 프롬프트 전략)
각 데이터를 별도의 API 호출로 수집 후 병합한다.

##### FR-2.1: 상세 목차 수집 (기본 모드 포함)
- [ ] 계층적 목차 추출 (Part → Chapter → Section → Subsection 등)
- [ ] depth 제한 없이 모델이 알려주는 최대한 깊은 단계까지
- [ ] 트리 구조 JSON으로 변환 (재귀적 `children` 필드)
- [ ] 원어 + 한국어 번역 병기

##### FR-2.2: 기본 메타데이터 수집 (기본 모드 포함)
- [ ] 제목 (원어 + 한국어)
- [ ] 저자
- [ ] 출판사
- [ ] 출판일
- [ ] ISBN-13
- [ ] 페이지 수
- [ ] 언어
- [ ] 에디션 (해당 시)

##### FR-2.3: 평가 및 후기 수집 (`--full` 모드 전용)
- [ ] 전체 평점 (별점)
- [ ] 후기 요약 (장점/단점)
- [ ] 추천 독자 수준 (초급/중급/고급)
- [ ] 전제 지식 (prerequisites)

##### FR-2.4: 관련 동영상 강의 수집 (`--full` 모드 전용)
- [ ] 강의명
- [ ] 플랫폼 (YouTube, Udemy, Inflearn, Coursera 등)
- [ ] 강사명
- [ ] 커리큘럼/섹션 목록
- [ ] 평점
- [ ] 가격 정보
- [ ] URL (알려진 경우)

##### FR-2.5: 유사 도서 추천 (`--full` 모드 전용)
- [ ] 같은 주제의 추천 도서 목록 (3-5권)
- [ ] 각 추천 도서의 간략 설명
- [ ] 난이도 비교

#### FR-3: JSON 출력 구조
- [ ] 통합 JSON 스키마 정의
- [ ] 목차는 재귀적 트리 구조
- [ ] stdout에 요약 정보 출력
- [ ] JSON 파일 자동 저장
- [ ] 기본 모드에서 `--full` 전용 필드는 빈 배열/빈 객체로 출력 (스키마 일관성 유지)
- [ ] `data_availability` 필드로 어떤 섹션에 실제 데이터가 있는지 표시

#### FR-4: 캐시 기능
- [ ] 이전 조회 결과를 로컬에 캐시
- [ ] 동일한 책 재조회 시 캐시에서 반환
- [ ] `--no-cache`: 캐시 **읽기만** 우회 (새 결과는 캐시에 덮어쓰기 저장)
- [ ] 캐시 저장 위치: `~/.cache/bookinfo/`
- [ ] 캐시 키: 제목(또는 ISBN) + 저자 + 언어 + 모델 + 수집모드(basic/full) 조합의 해시
- [ ] 캐시 디렉토리 없으면 자동 생성

#### FR-5: 배치 처리
- [ ] 텍스트 파일에서 책 제목 목록 읽기 (한 줄에 하나)
- [ ] 순차 처리 (API rate limit 고려)
- [ ] 항목 간 기본 딜레이: 고정 3초
- [ ] 429 발생 시 다음 항목 간격 2배 증가 (최대 30초), 3연속 성공 시 기본값 복원
- [ ] 개별 결과를 각각의 JSON 파일로 저장
- [ ] 부분 실패 허용: 실패 항목은 로그 출력, 나머지 계속 처리
- [ ] 완료 시 성공/실패 집계 요약 출력

### Non-Functional Requirements

#### 성능
- 단일 책 기본 모드: API 2회 호출 (보통 5-15초)
- 단일 책 전체 모드: API 5회 호출 (보통 15-40초)
- 배치 처리 시 항목 간 기본 3초 딜레이

#### 안정성
- JSON 구조 검증 후 실패 시 프롬프트 조정하여 재시도 (최대 3회)
- API 에러(429, 500 등) 시 지수 백오프 재시도
  - 초기 지연: 2초
  - 배수: 2x (2s → 4s → 8s)
  - Jitter: 0~1초 랜덤
  - 최대 대기 캡: 30초
  - 최대 재시도: 3회

#### 보안
- API 키는 환경변수(`OPENROUTER_API_KEY`) 우선, CLI 플래그로 오버라이드 가능
- API 키를 로그나 출력에 노출하지 않음

---

## Design Decisions

### DD-1: `--no-cache` 동작 — 읽기만 우회
- `--no-cache`는 캐시 **읽기를 skip**하고, 새 결과는 캐시에 **덮어쓰기 저장**한다.
- 근거: "최신 데이터로 갱신하고 싶다"는 의도가 대부분이며, 새 결과를 저장해두면 다음 조회 시 갱신된 캐시 사용 가능.

### DD-2: `--lang` 허용값 — 고정 목록
- 허용값: `ko`, `en`, `ja`, `zh-tw` (4개 고정)
- 미지정 시: 모델이 제목으로 자동 판단
- 근거: lang 값이 캐시 키, 파일명, 프롬프트 언어 지시에 모두 쓰이므로 검증된 값만 허용.

### DD-3: 동명 도서 선택 기준 — 인기 우선, 동저자 시 최신판
- 프롬프트에 삽입할 우선순위:
  1. 해당 분야에서 가장 인기 있는(많이 인용/추천되는) 책
  2. 저자명까지 동일한 경우 → **최신 에디션** 선택
  3. `--lang` 지정 시 해당 언어권 도서 우선
  4. `--author` 지정 시 정확히 매칭
- JSON에 `selection_note` 필드를 추가하여 선택 근거 기록
- stdout에 "동명 도서가 있을 수 있습니다. `--author`로 특정하세요" 힌트 출력

### DD-4: 기본 출력 파일명 날짜 타임존
- **파일명**: 로컬 타임존 (`clean-code_20260301.json`)
- **JSON 내부 `collected_at`**: UTC (ISO 8601, `2026-03-01T10:30:00Z`)
- 근거: 파일명은 사용자 직관, 데이터 타임스탬프는 국제 표준.

### DD-5: 데이터 없음 시 정책 — 빈 배열/빈 객체
- `related_courses: []`, `similar_books: []`, `review: { rating: 0, summary: { pros: [], cons: [] }, ... }`
- null이나 에러가 아닌 빈 값으로 일관 처리 (스키마 구조 유지)
- `data_availability` 필드로 어떤 섹션에 데이터가 있는지 요약 표시
- 근거: Go의 zero value 철학과 일치, JSON 소비 측에서 `len() == 0` 체크만으로 충분.

### DD-6: 백오프 파라미터 — 하드코딩 상수
- 초기 지연: 2초 / 배수: 2x / 최대 재시도: 3회 / Jitter: 0~1초 / 최대 대기 캡: 30초
- 개인 도구이므로 설정 파일 불필요, 상수로 충분.

### DD-7: 배치 항목 간 delay — 고정 3초 + 동적 증가
- 기본 간격: 고정 3초
- 429 발생 시: 다음 항목 간격 2배 증가 (최대 30초)
- 3연속 성공 시: 기본값(3초)으로 복원
- stdout에 진행 상황 표시: `[2/10] "Clean Code" 처리 중...`

### DD-8: 수집 모드 — 기본(메타+목차) / 전체(--full)
- 기본 모드: 메타데이터 + 목차만 수집 (API 2회), 빠르고 저렴
- 전체 모드: 메타+목차+후기+강의+유사도서 (API 5회)
- 근거: 대부분의 사용 사례에서 목차와 메타데이터만 필요, 나머지는 선택적.

### DD-9: ISBN-13 검색 지원
- `--isbn` 플래그로 ISBN-13 직접 입력 가능
- 제목과 ISBN 동시 입력 시 ISBN 우선 (더 정확한 식별)
- ISBN은 프롬프트에 그대로 전달하여 모델이 정확한 도서를 특정하도록 함
- 캐시 키에도 ISBN 포함

---

## JSON 출력 스키마

```json
{
  "book": {
    "title": {
      "original": "Clean Code",
      "korean": "클린 코드"
    },
    "author": "Robert C. Martin",
    "publisher": "Prentice Hall",
    "published_date": "2008-08-01",
    "isbn13": "978-0132350884",
    "pages": 464,
    "language": "en",
    "edition": "1st",
    "selection_note": "동명 도서 중 Robert C. Martin의 2008년 원서를 선택 (가장 널리 알려진 판본)"
  },
  "table_of_contents": [
    {
      "title": {
        "original": "Chapter 1: Clean Code",
        "korean": "1장: 깨끗한 코드"
      },
      "depth": 1,
      "children": [
        {
          "title": {
            "original": "There Will Be Code",
            "korean": "코드는 존재할 것이다"
          },
          "depth": 2,
          "children": []
        }
      ]
    }
  ],
  "review": {
    "rating": 4.5,
    "summary": {
      "pros": ["실용적인 코드 예제"],
      "cons": ["Java 중심"]
    },
    "recommended_level": "중급",
    "prerequisites": ["기본 프로그래밍 경험", "객체지향 이해"]
  },
  "related_courses": [
    {
      "title": "Clean Code with Uncle Bob",
      "platform": "YouTube",
      "instructor": "Robert C. Martin",
      "curriculum": ["Episode 1: Clean Code"],
      "rating": 4.8,
      "price": "무료",
      "url": "https://..."
    }
  ],
  "similar_books": [
    {
      "title": {
        "original": "Refactoring",
        "korean": "리팩터링"
      },
      "author": "Martin Fowler",
      "brief_description": "코드 개선을 위한 체계적 접근법",
      "difficulty_comparison": "비슷한 수준"
    }
  ],
  "data_availability": {
    "table_of_contents": true,
    "review": true,
    "related_courses": true,
    "similar_books": true
  },
  "metadata": {
    "collected_at": "2026-03-01T10:30:00Z",
    "model_used": "qwen/qwen3.5-flash",
    "source": "openrouter",
    "mode": "full",
    "query": {
      "title": "Clean Code",
      "isbn13": "",
      "author": "",
      "lang": ""
    }
  }
}
```

**기본 모드 출력 시**: `review`, `related_courses`, `similar_books`는 빈 객체/빈 배열, `data_availability`에서 해당 항목은 `false`.

---

## CLI 사용 예시

```bash
# 기본 사용 (메타+목차만)
bookinfo "Clean Code"

# 전체 데이터 수집
bookinfo "Clean Code" --full

# ISBN-13으로 검색
bookinfo --isbn 978-0132350884

# ISBN + 전체 모드
bookinfo --isbn 978-0132350884 --full

# 저자와 언어 지정
bookinfo "클린 코드" --author "로버트 C. 마틴" --lang ko

# 출력 파일 지정
bookinfo "Effective Go" -o ./results/effective-go.json

# 모델 변경
bookinfo "The Go Programming Language" --model "anthropic/claude-sonnet-4"

# 배치 처리 (전체 모드)
bookinfo --batch books.txt --full

# 캐시 읽기 우회 (새 결과는 캐시에 저장)
bookinfo "Clean Code" --no-cache

# API 키 직접 전달
bookinfo "Clean Code" --api-key sk-or-xxx
```

---

## User Flow

### 기본 모드 (default)
1. 사용자가 CLI에 책 제목 또는 ISBN(+선택 옵션)을 입력
2. 입력 검증: `--lang` 허용값 확인, 제목/ISBN 중 하나 필수
3. 캐시 확인 → 캐시 히트 시 즉시 반환 (`--no-cache` 시 읽기 skip)
4. OpenRouter API에 분리된 프롬프트로 순차 요청
   - 요청 1: 기본 메타데이터
   - 요청 2: 상세 목차
5. 각 응답의 JSON 구조 검증 → 실패 시 재시도 (최대 3회)
6. 데이터를 통합 JSON으로 병합 (review/courses/similar는 빈 값)
7. stdout에 요약 출력 (제목, 저자, 목차 1단계)
8. JSON 파일 저장 후 파일 경로 출력
9. 캐시에 결과 저장 (`--no-cache`여도 저장)

### 전체 모드 (`--full`)
1~3 동일
4. OpenRouter API에 분리된 프롬프트로 순차 요청
   - 요청 1: 기본 메타데이터
   - 요청 2: 상세 목차
   - 요청 3: 평가/후기
   - 요청 4: 관련 동영상 강의
   - 요청 5: 유사 도서 추천
5~9 동일 (후기/강의/유사도서 데이터 없으면 빈 배열로 처리)

---

## Edge Cases & Error Handling

| 시나리오 | 처리 방법 |
|---------|----------|
| 존재하지 않는 책 제목/ISBN | 모델 응답에 "찾을 수 없음" 포함 시 경고 출력 후 종료 |
| API 키 미설정 | 명확한 에러 메시지와 설정 방법 안내 출력 |
| API rate limit (429) | 지수 백오프 재시도 (2s→4s→8s, jitter 0~1s, 최대 3회) |
| 모델 응답이 JSON이 아닌 경우 | JSON 파싱 실패 시 프롬프트 수정 후 재시도 (최대 3회) |
| 네트워크 오류 | 타임아웃 설정 (30초), 재시도 후 실패 시 에러 출력 |
| 동명의 책이 여러 개 | 인기 우선 선택, 동저자 시 최신판. `selection_note`에 근거 기록 |
| 배치 처리 중 일부 실패 | 실패한 항목 로그 출력, 나머지는 계속 처리, 최종 집계 출력 |
| 캐시 디렉토리 없음 | 자동 생성 |
| `--lang` 허용값 외 입력 | "허용값: ko, en, ja, zh-tw" 에러 메시지 출력 |
| 제목과 ISBN 둘 다 없음 | "제목 또는 --isbn 중 하나는 필수" 에러 메시지 출력 |
| 전체 모드에서 후기/강의/유사도서 없음 | 빈 배열/객체로 출력, `data_availability`에 false 표시 |
| 캐시 파일 손상 | cache miss로 처리, 새로 수집 후 덮어쓰기 |

---

## Out of Scope

- 웹 스크래핑 (실제 서점 사이트 크롤링) — LLM 지식 기반만 사용
- GUI/웹 인터페이스
- 데이터베이스 연동
- 실시간 가격 비교
- 실제 구매 링크 검증
- ISBN 유효성 체크섬 검증 (모델에 위임)

---

## Success Metrics

- 단일 책 기본 모드 조회 시 목차가 2단계 이상 depth로 추출되는가
- JSON 파일이 정의된 스키마에 맞게 생성되는가
- 한국어/영어/일본어 책 모두 원어+한국어 병기로 출력되는가
- 기본 모드에서 `--full` 전용 필드가 빈 배열/객체로 일관 출력되는가
- ISBN-13 검색이 제목 검색과 동일한 스키마로 동작하는가
- `--no-cache` 사용 후 다음 조회 시 갱신된 캐시가 반환되는가
- 배치 처리가 정상 작동하는가

---

## Implementation Plan

### Phase 1: 프로젝트 기초 설정
- [ ] 1.1: Go 모듈 초기화 (`go mod init`)
- [ ] 1.2: CLI 프레임워크 선택 및 플래그/인자 파싱 구현 (`cobra` 또는 `flag`)
- [ ] 1.3: 검색 입력 처리 (제목 positional arg + `--isbn` 플래그)
- [ ] 1.4: `--lang` 허용값 검증 (`ko`, `en`, `ja`, `zh-tw`)
- [ ] 1.5: `--full` 모드 플래그 구현
- [ ] 1.6: 설정 관리 (환경변수 `OPENROUTER_API_KEY`, CLI `--api-key` 오버라이드)
- [ ] 1.7: 프로젝트 디렉토리 구조 설계

### Phase 2: OpenRouter API 클라이언트
- [ ] 2.1: OpenRouter API 호출 HTTP 클라이언트 구현
- [ ] 2.2: 요청/응답 구조체 정의
- [ ] 2.3: 에러 핸들링 — 지수 백오프 (초기 2s, 배수 2x, jitter 0~1s, 캡 30s, 최대 3회)
- [ ] 2.4: JSON 응답 파싱 및 구조 검증
- [ ] 2.5: 모델 선택 기능 (`--model` 플래그)

### Phase 3: 프롬프트 설계 및 데이터 수집
- [ ] 3.1: 메타데이터 수집 프롬프트 작성 (ISBN-13 입력 지원)
- [ ] 3.2: 목차 수집 프롬프트 작성 (계층 구조 유도, depth 무제한)
- [ ] 3.3: 평가/후기 수집 프롬프트 작성 (`--full` 전용)
- [ ] 3.4: 동영상 강의 수집 프롬프트 작성 (`--full` 전용)
- [ ] 3.5: 유사 도서 수집 프롬프트 작성 (`--full` 전용)
- [ ] 3.6: 원어+한국어 병기 지시 통합
- [ ] 3.7: 동명 도서 선택 기준 프롬프트 삽입 (인기 우선, 동저자 시 최신판)

### Phase 4: 데이터 병합 및 출력
- [ ] 4.1: 수집 모드별 응답 병합 (기본 2개, 전체 5개)
- [ ] 4.2: 미수집 섹션 빈 배열/객체 채움 + `data_availability` 생성
- [ ] 4.3: stdout 요약 출력 포맷 구현
- [ ] 4.4: JSON 파일 저장 (파일명: 로컬 타임존 날짜, `-o` 오버라이드)
- [ ] 4.5: 전체 JSON 스키마 검증

### Phase 5: 부가 기능
- [ ] 5.1: 캐시 시스템 구현 (`~/.cache/bookinfo/`, 키 = 입력+모드 해시)
- [ ] 5.2: `--no-cache` 구현 (읽기 skip, 쓰기 수행)
- [ ] 5.3: 배치 처리 구현 (순차, 고정 3초 + 동적 증가)
- [ ] 5.4: 배치 진행 상황 및 집계 출력

### Phase 6: 테스트 및 완성
- [ ] 6.1: 영어 책 기본 모드 테스트 (예: "Clean Code")
- [ ] 6.2: 영어 책 전체 모드 테스트
- [ ] 6.3: ISBN-13 검색 테스트
- [ ] 6.4: 한국어 책 테스트 (예: "클린 코드")
- [ ] 6.5: 일본어/대만 책 테스트
- [ ] 6.6: 배치 처리 테스트
- [ ] 6.7: 캐시 동작 테스트 (히트, `--no-cache` 갱신)
- [ ] 6.8: 에러 케이스 테스트

---

## Technical Decisions

| 결정 | 선택 | 이유 |
|------|------|------|
| CLI 프레임워크 | `cobra` | Go 표준, 서브커맨드 확장 용이 |
| HTTP 클라이언트 | `net/http` | 외부 의존성 최소화 |
| JSON 처리 | `encoding/json` | Go 표준 라이브러리 |
| 캐시 저장 형식 | JSON 파일 | 단순, 디버깅 용이 |
| 프롬프트 전략 | 분리 호출 | 응답 품질 향상 |
| 기본 수집 모드 | 메타+목차만 | API 비용 절감, 빠른 응답 |
| 데이터 없음 처리 | 빈 배열/객체 | 스키마 일관성, Go zero value |
| 백오프 파라미터 | 하드코딩 상수 | 개인 도구, 설정 파일 불필요 |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| LLM 응답 품질 불안정 | 높음 | 중간 | JSON 검증 + 재시도, 프롬프트 최적화 |
| API 비용 증가 (분리 호출) | 중간 | 낮음 | 기본 모드 2회만, 캐시로 반복 방지, QWEN3.5-flash 저렴 |
| 일본어/대만어 목차 정확도 | 중간 | 중간 | 원어+한국어 병기로 검증 용이 |
| OpenRouter API 변경 | 낮음 | 높음 | API 클라이언트 추상화 |
| ISBN으로 책을 못 찾는 경우 | 낮음 | 중간 | 제목 fallback 안내 메시지 |

---

## Test Strategy

- 수동 테스트: 다양한 언어/장르의 책으로 실제 API 호출 테스트
- JSON 스키마 검증: 출력이 정의된 스키마에 맞는지 자동 확인
- 모드 테스트: 기본 모드/전체 모드 각각의 출력 스키마 검증
- ISBN 검색: ISBN-13 입력 시 정확한 도서 매칭 확인
- 에러 케이스: 잘못된 API 키, 네트워크 에러, 잘못된 lang 값 등 시뮬레이션
- 캐시 테스트: 동일 책 2회 조회 시 캐시 히트, `--no-cache` 후 갱신 확인
