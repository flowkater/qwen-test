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
- [ ] 책 제목을 필수 인자로 받음
- [ ] `--author` / `-a`: 저자명 지정 (선택)
- [ ] `--lang` / `-l`: 대상 언어/국가 지정 (선택, 예: ko, en, ja, zh-tw)
- [ ] `--api-key`: API 키 직접 전달 (선택, 환경변수 오버라이드)
- [ ] `--model` / `-m`: LLM 모델 선택 (기본값: `qwen/qwen3.5-flash`)
- [ ] `-o` / `--output`: 출력 파일 경로 지정 (기본값: `{제목}_{날짜}.json`)
- [ ] `--batch` / `-b`: 배치 파일 경로 (텍스트 파일에서 여러 책 제목 읽기)
- [ ] `--no-cache`: 캐시 무시 플래그

#### FR-2: 데이터 수집 (분리 프롬프트 전략)
각 데이터를 별도의 API 호출로 수집 후 병합한다.

##### FR-2.1: 상세 목차 수집
- [ ] 계층적 목차 추출 (Part → Chapter → Section → Subsection 등)
- [ ] depth 제한 없이 모델이 알려주는 최대한 깊은 단계까지
- [ ] 트리 구조 JSON으로 변환 (재귀적 `children` 필드)
- [ ] 원어 + 한국어 번역 병기

##### FR-2.2: 기본 메타데이터 수집
- [ ] 제목 (원어 + 한국어)
- [ ] 저자
- [ ] 출판사
- [ ] 출판일
- [ ] ISBN
- [ ] 페이지 수
- [ ] 언어
- [ ] 에디션 (해당 시)

##### FR-2.3: 평가 및 후기 수집
- [ ] 전체 평점 (별점)
- [ ] 후기 요약 (장점/단점)
- [ ] 추천 독자 수준 (초급/중급/고급)
- [ ] 전제 지식 (prerequisites)

##### FR-2.4: 관련 동영상 강의 수집
- [ ] 강의명
- [ ] 플랫폼 (YouTube, Udemy, Inflearn, Coursera 등)
- [ ] 강사명
- [ ] 커리큘럼/섹션 목록
- [ ] 평점
- [ ] 가격 정보
- [ ] URL (알려진 경우)

##### FR-2.5: 유사 도서 추천
- [ ] 같은 주제의 추천 도서 목록 (3-5권)
- [ ] 각 추천 도서의 간략 설명
- [ ] 난이도 비교

#### FR-3: JSON 출력 구조
- [ ] 통합 JSON 스키마 정의
- [ ] 목차는 재귀적 트리 구조
- [ ] stdout에 요약 정보 출력
- [ ] JSON 파일 자동 저장

#### FR-4: 캐시 기능
- [ ] 이전 조회 결과를 로컬에 캐시
- [ ] 동일한 책 재조회 시 캐시에서 반환
- [ ] `--no-cache` 플래그로 캐시 무시 가능
- [ ] 캐시 저장 위치: `~/.cache/bookinfo/`

#### FR-5: 배치 처리
- [ ] 텍스트 파일에서 책 제목 목록 읽기 (한 줄에 하나)
- [ ] 순차 처리 (API rate limit 고려)
- [ ] 개별 결과를 각각의 JSON 파일로 저장

### Non-Functional Requirements

#### 성능
- 단일 책 조회: API 응답 시간에 의존 (보통 10-30초)
- 배치 처리 시 API 호출 간 적절한 딜레이

#### 안정성
- JSON 구조 검증 후 실패 시 프롬프트 조정하여 재시도 (최대 3회)
- API 에러(429, 500 등) 시 백오프 재시도

#### 보안
- API 키는 환경변수(`OPENROUTER_API_KEY`) 우선, CLI 플래그로 오버라이드 가능
- API 키를 로그나 출력에 노출하지 않음

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
    "isbn": "978-0132350884",
    "pages": 464,
    "language": "en",
    "edition": "1st"
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
      "pros": ["실용적인 코드 예제", "..."],
      "cons": ["Java 중심", "..."]
    },
    "recommended_level": "중급",
    "prerequisites": ["기본 프로그래밍 경험", "객체지향 이해"]
  },
  "related_courses": [
    {
      "title": "Clean Code with Uncle Bob",
      "platform": "YouTube",
      "instructor": "Robert C. Martin",
      "curriculum": ["Episode 1: Clean Code", "..."],
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
  "metadata": {
    "collected_at": "2026-03-01T10:30:00Z",
    "model_used": "qwen/qwen3.5-flash",
    "source": "openrouter"
  }
}
```

---

## CLI 사용 예시

```bash
# 기본 사용
bookinfo "Clean Code"

# 저자와 언어 지정
bookinfo "클린 코드" --author "로버트 C. 마틴" --lang ko

# 출력 파일 지정
bookinfo "Effective Go" -o ./results/effective-go.json

# 모델 변경
bookinfo "The Go Programming Language" --model "anthropic/claude-sonnet-4"

# 배치 처리
bookinfo --batch books.txt

# 캐시 무시
bookinfo "Clean Code" --no-cache

# API 키 직접 전달
bookinfo "Clean Code" --api-key sk-or-xxx
```

---

## User Flow

1. 사용자가 CLI에 책 제목(+선택 옵션)을 입력
2. 프로그램이 캐시 확인 → 캐시 히트 시 즉시 반환
3. OpenRouter API에 분리된 프롬프트로 순차 요청
   - 요청 1: 상세 목차
   - 요청 2: 기본 메타데이터
   - 요청 3: 평가/후기
   - 요청 4: 관련 동영상 강의
   - 요청 5: 유사 도서 추천
4. 각 응답의 JSON 구조 검증 → 실패 시 재시도 (최대 3회)
5. 모든 데이터를 통합 JSON으로 병합
6. stdout에 요약 출력 (제목, 저자, 목차 1단계, 평점 등)
7. JSON 파일 저장 후 파일 경로 출력
8. 캐시에 결과 저장

---

## Edge Cases & Error Handling

| 시나리오 | 처리 방법 |
|---------|----------|
| 존재하지 않는 책 제목 | 모델 응답에 "찾을 수 없음" 포함 시 경고 출력 후 종료 |
| API 키 미설정 | 명확한 에러 메시지와 설정 방법 안내 |
| API rate limit (429) | 지수 백오프 재시도 (최대 3회) |
| 모델 응답이 JSON이 아닌 경우 | JSON 파싱 실패 시 프롬프트 수정 후 재시도 |
| 네트워크 오류 | 타임아웃 설정 (30초), 재시도 후 실패 시 에러 출력 |
| 동명의 책이 여러 개 | 모델이 가장 유명한 것을 선택, 저자/언어 지정 권장 메시지 |
| 배치 처리 중 일부 실패 | 실패한 항목 로그 출력, 나머지는 계속 처리 |
| 캐시 디렉토리 없음 | 자동 생성 |

---

## Out of Scope

- 웹 스크래핑 (실제 서점 사이트 크롤링) — LLM 지식 기반만 사용
- GUI/웹 인터페이스
- 데이터베이스 연동
- 실시간 가격 비교
- 실제 구매 링크 검증

---

## Success Metrics

- 단일 책 조회 시 목차가 2단계 이상 depth로 추출되는가
- JSON 파일이 정의된 스키마에 맞게 생성되는가
- 한국어/영어/일본어 책 모두 원어+한국어 병기로 출력되는가
- 캐시 동작 여부 확인
- 배치 처리가 정상 작동하는가

---

## Implementation Plan

### Phase 1: 프로젝트 기초 설정
- [ ] 1.1: Go 모듈 초기화 (`go mod init`)
- [ ] 1.2: CLI 프레임워크 선택 및 플래그/인자 파싱 구현 (`cobra` 또는 `flag`)
- [ ] 1.3: 설정 관리 (환경변수, CLI 플래그)
- [ ] 1.4: 프로젝트 디렉토리 구조 설계

### Phase 2: OpenRouter API 클라이언트
- [ ] 2.1: OpenRouter API 호출 HTTP 클라이언트 구현
- [ ] 2.2: 요청/응답 구조체 정의
- [ ] 2.3: 에러 핸들링 (429 백오프, 타임아웃, 재시도)
- [ ] 2.4: JSON 응답 파싱 및 구조 검증

### Phase 3: 프롬프트 설계 및 데이터 수집
- [ ] 3.1: 목차 수집 프롬프트 작성 (계층 구조 유도)
- [ ] 3.2: 메타데이터 수집 프롬프트 작성
- [ ] 3.3: 평가/후기 수집 프롬프트 작성
- [ ] 3.4: 동영상 강의 수집 프롬프트 작성
- [ ] 3.5: 유사 도서 수집 프롬프트 작성
- [ ] 3.6: 원어+한국어 병기 지시 통합

### Phase 4: 데이터 병합 및 출력
- [ ] 4.1: 개별 응답을 통합 JSON으로 병합
- [ ] 4.2: stdout 요약 출력 포맷 구현
- [ ] 4.3: JSON 파일 저장 (파일명 규칙 적용)
- [ ] 4.4: 전체 JSON 스키마 검증

### Phase 5: 부가 기능
- [ ] 5.1: 캐시 시스템 구현 (`~/.cache/bookinfo/`)
- [ ] 5.2: 배치 처리 구현 (파일에서 목록 읽기)
- [ ] 5.3: 모델 선택 기능 구현
- [ ] 5.4: `--no-cache` 플래그 구현

### Phase 6: 테스트 및 완성
- [ ] 6.1: 한국어 책 테스트 (예: "클린 코드")
- [ ] 6.2: 영어 책 테스트 (예: "Clean Code")
- [ ] 6.3: 일본어 책 테스트
- [ ] 6.4: 배치 처리 테스트
- [ ] 6.5: 에러 케이스 테스트

---

## Technical Decisions

| 결정 | 선택 | 이유 |
|------|------|------|
| CLI 프레임워크 | `cobra` | Go 표준, 서브커맨드 확장 용이 |
| HTTP 클라이언트 | `net/http` | 외부 의존성 최소화 |
| JSON 처리 | `encoding/json` | Go 표준 라이브러리 |
| 캐시 저장 형식 | JSON 파일 | 단순, 디버깅 용이 |
| 프롬프트 전략 | 분리 호출 | 응답 품질 향상 |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| LLM 응답 품질 불안정 | 높음 | 중간 | JSON 검증 + 재시도, 프롬프트 최적화 |
| API 비용 증가 (분리 호출) | 중간 | 낮음 | 캐시로 반복 호출 방지, QWEN3.5-flash는 저렴 |
| 일본어/대만어 목차 정확도 | 중간 | 중간 | 원어+한국어 병기로 검증 용이 |
| OpenRouter API 변경 | 낮음 | 높음 | API 클라이언트 추상화 |

---

## Test Strategy

- 수동 테스트: 다양한 언어/장르의 책으로 실제 API 호출 테스트
- JSON 스키마 검증: 출력이 정의된 스키마에 맞는지 자동 확인
- 에러 케이스: 잘못된 API 키, 네트워크 에러 등 시뮬레이션
- 캐시 테스트: 동일 책 2회 조회 시 캐시 히트 확인
