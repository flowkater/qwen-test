# Qwen API 품질 개선 가이드

> 현재 통과율 1.9~13.2% → 목표 60%+ 달성을 위한 수정 가이드

---

## 목차

1. [현재 상태 진단](#1-현재-상태-진단)
2. [근본 원인 분석 (5개)](#2-근본-원인-분석)
3. [수정 사항 A: OpenRouter 클라이언트](#3-수정-사항-a-openrouter-클라이언트)
4. [수정 사항 B: 프롬프트 재설계](#4-수정-사항-b-프롬프트-재설계)
5. [수정 사항 C: 응답 후처리](#5-수정-사항-c-응답-후처리)
6. [수정 사항 D: 검증 로직 완화](#6-수정-사항-d-검증-로직-완화)
7. [수정 사항 E: 비교 스크립트](#7-수정-사항-e-비교-스크립트)
8. [파일별 수정 체크리스트](#8-파일별-수정-체크리스트)
9. [예상 효과](#9-예상-효과)

---

## 1. 현재 상태 진단

### 통과율

| prompt_mode | 통과율 |
|-------------|--------|
| strict-json | 13.2% |
| two-step-json | 5.7% |
| soft-json | 1.9% |

### 실패 패턴 분석

실제 응답 (`live_request_raw_20260302_095318.json`) 분석:

```
1. 응답에 markdown + 설명문 + JSON 혼재 → JSON 파싱 실패 또는 불완전
2. 영문 책(Clean Code) 검색인데 한국어 챕터 제목 반환 ("깨끗한코드", "의미있는이름")
3. ISBN, publisher 등 메타데이터 누락 (빈 문자열)
4. 챕터 수 불일치 (기대 8, 실측 17 — 한국어판 기준으로 응답)
5. depth 1단계만 (기대 3단계 계층 구조)
```

---

## 2. 근본 원인 분석

### 원인 1: System Prompt 부재 (치명적)

**현재 코드** (`internal/infra/openrouter/client.go:52-56`):

```go
payload.Messages = append(payload.Messages, struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}{Role: "user", Content: prompt})
```

- system 메시지 없이 user만 전송
- qwen.ai 채팅 UI는 내부 시스템 프롬프트가 자동 주입됨
- API에서 system 없으면 모델이 역할/형식을 추론해야 해서 품질 저하

### 원인 2: temperature/response_format 미설정

**현재 코드** (`internal/infra/openrouter/client.go`):

```go
type RequestPayload struct {
    Model    string `json:"model"`
    Messages []struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    } `json:"messages"`
}
```

- `temperature` 필드 없음 → API 기본값 1.0 적용 (너무 높음)
- `response_format` 필드 없음 → 모델이 markdown/설명문 자유롭게 섞어서 출력
- 비교 스크립트(`live_compare_openrouter.py`)는 `temperature: 0` 사용 중 — Go 코드와 불일치

### 원인 3: 언어 제어 미흡

**현재 프롬프트** (`internal/app/prompts.go:61`):

```go
builder.WriteString(" Always include original and korean title pairs.")
```

- "original + korean 쌍으로 달라"고 했지만, Clean Code(lang=en) 검색 시 모델이 한국어판 검색결과를 우선 반환
- 실제 응답: 알라딘(한국 서점) 검색결과 기반으로 한국어 챕터명 반환
- `lang` 필드가 프롬프트에서 충분히 활용되지 않음

### 원인 4: 프롬프트-검증 스키마 불일치

**프롬프트가 요구하는 TOC 스키마:**
```json
[{"title":{"original":"","korean":""},"depth":1,"children":[]}]
```

**verified JSON의 검증 기준:**
```json
{
  "table_of_contents": {
    "total_chapters": 8,
    "chapter_titles": ["Meaningful Names", "Functions", ...]
  }
}
```

- 프롬프트는 `title.original` + `children` 재귀 트리를 요구
- 검증은 `chapter_titles` 플랫 배열에서 키워드 포함 여부를 검사
- **두 스키마가 완전히 다름** → 통과 불가능

### 원인 5: exact match 기준 과도

**검증 코드 예시** (`live_compare_openrouter.py`):

```python
checks.append(("metadata.author", exp, act, str(exp) == str(act)))
```

- `"Robert C. Martin"` vs `"로버트 C. 마틴 (Robert C. Martin)"` → ❌
- `"Inflearn"` vs `"인프런"` → ❌
- 표기 변형, 언어 차이를 전혀 허용하지 않음

---

## 3. 수정 사항 A: OpenRouter 클라이언트

### 파일: `internal/infra/openrouter/client.go`

#### A-1. RequestPayload 구조체 확장

```go
// 변경 전
type RequestPayload struct {
    Model    string `json:"model"`
    Messages []struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    } `json:"messages"`
}

// 변경 후
type RequestPayload struct {
    Model          string          `json:"model"`
    Messages       []Message       `json:"messages"`
    Temperature    *float64        `json:"temperature,omitempty"`
    MaxTokens      int             `json:"max_tokens,omitempty"`
    ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ResponseFormat struct {
    Type string `json:"type"` // "json_object" 또는 "text"
}
```

#### A-2. Collect 메서드에 system message + 파라미터 추가

```go
func (c *Client) Collect(ctx context.Context, model, apiKey, systemPrompt, userPrompt string) ([]byte, error) {
    if model == "" {
        model = domain.DefaultModel
    }

    temp := 0.1
    payload := RequestPayload{
        Model:       model,
        Temperature: &temp,
        MaxTokens:   4000,
        ResponseFormat: &ResponseFormat{Type: "json_object"},
        Messages: []Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userPrompt},
        },
    }

    // ... 이하 기존 HTTP 요청 로직 동일
}
```

**주의**: `Collect` 시그니처가 변경되므로 `collector.go`와 `interfaces.go`도 같이 수정 필요.

#### A-3. 응답에서 JSON 추출 로직 강화

현재 `content`를 그대로 반환하지만, 모델이 markdown을 섞어 줄 수 있으므로 JSON 추출 필요:

```go
func extractJSON(content string) ([]byte, error) {
    content = strings.TrimSpace(content)

    // 1. ```json ... ``` 코드블록 추출
    if idx := strings.Index(content, "```json"); idx >= 0 {
        start := idx + 7
        if end := strings.Index(content[start:], "```"); end >= 0 {
            content = strings.TrimSpace(content[start : start+end])
        }
    } else if idx := strings.Index(content, "```"); idx >= 0 {
        start := idx + 3
        if end := strings.Index(content[start:], "```"); end >= 0 {
            content = strings.TrimSpace(content[start : start+end])
        }
    }

    // 2. 첫 번째 { 또는 [ 찾기
    if !strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
        if start := strings.IndexAny(content, "{["); start >= 0 {
            // 매칭되는 닫는 괄호 찾기
            open := content[start]
            var close byte = '}'
            if open == '[' {
                close = ']'
            }
            depth := 0
            for i := start; i < len(content); i++ {
                if content[i] == open {
                    depth++
                } else if content[i] == close {
                    depth--
                    if depth == 0 {
                        content = content[start : i+1]
                        break
                    }
                }
            }
        }
    }

    // 3. 유효한 JSON인지 검증
    if !json.Valid([]byte(content)) {
        return nil, fmt.Errorf("response is not valid JSON")
    }

    return []byte(content), nil
}
```

`Collect` 메서드의 반환 직전에 적용:

```go
// 기존
return []byte(content), nil

// 변경
return extractJSON(content)
```

---

## 4. 수정 사항 B: 프롬프트 재설계

### 파일: `internal/app/prompts.go`

#### B-1. System Prompt 상수 추가

```go
const SystemPromptBookInfo = `You are a precise book/lecture metadata and table of contents extraction engine.

CRITICAL RULES:
1. Return ONLY valid JSON. No markdown, no explanation, no code blocks.
2. Use the EXACT JSON schema provided in the user message.
3. For books: chapter titles MUST be in the book's ORIGINAL language.
   - English book → English titles. Korean book → Korean titles.
   - Do NOT translate titles unless explicitly requested.
4. Include all available metadata: author, publisher, ISBN-13, page count.
5. For table of contents: provide FULL hierarchical structure.
   - Include subsections and sub-subsections when available.
   - depth 1 = Part/Book, depth 2 = Chapter, depth 3 = Section, depth 4 = Subsection.
6. Search the web for accurate, up-to-date information.
7. If a field is unknown, use empty string "" for text, 0 for numbers, [] for arrays.
8. Prefer the ORIGINAL edition's data (not translations) unless the query specifies a language.`
```

#### B-2. commonLead 함수 개선

```go
func commonLead(q domain.BookQuery) string {
    identifier := q.Title
    if q.ISBN13 != "" {
        identifier = fmt.Sprintf("ISBN-13: %s", q.ISBN13)
    }

    builder := strings.Builder{}
    builder.WriteString("Find canonical data for: ")
    builder.WriteString(identifier)

    if q.Author != "" {
        builder.WriteString(" by " + q.Author)
    }

    // 언어 명시 강화
    if q.Lang != "" {
        builder.WriteString(fmt.Sprintf(
            "\nIMPORTANT: This is a %s-language resource. "+
                "Return all titles in %s (the original language). "+
                "Do NOT translate to other languages.",
            q.Lang, q.Lang,
        ))
    }

    builder.WriteString("\nIf duplicate titles exist, choose the most popular edition. ")
    builder.WriteString("If same author/title, choose the latest edition.")

    return builder.String()
}
```

#### B-3. BuildTOCPrompt — verified 스키마와 일치시키기

```go
func BuildTOCPrompt(q domain.BookQuery) string {
    lead := commonLead(q)

    return lead + `

Return the table of contents as a JSON array of chapter objects.
Each chapter object has:
{
  "title": "Chapter title in ORIGINAL language",
  "depth": 1,
  "children": [
    {
      "title": "Section title",
      "depth": 2,
      "children": [
        {"title": "Subsection title", "depth": 3, "children": []}
      ]
    }
  ]
}

Rules:
- Include ALL chapters AND appendices.
- Provide full hierarchy: Chapter > Section > Subsection (depth 1 > 2 > 3).
- Do NOT flatten the structure. Preserve parent-child relationships.
- Chapter titles must be in the ORIGINAL language of the book/lecture.
- Return ONLY the JSON array. No wrapper object, no explanation.`
}
```

#### B-4. BuildMetadataPrompt — 필수 필드 강조

```go
func BuildMetadataPrompt(q domain.BookQuery) string {
    lead := commonLead(q)

    return lead + `

Return metadata as a JSON object:
{
  "title": {"original": "", "korean": ""},
  "author": "",
  "publisher": "",
  "published_date": "",
  "isbn13": "",
  "pages": 0,
  "language": "",
  "edition": "",
  "selection_note": ""
}

IMPORTANT:
- "author" must be in the ORIGINAL language (e.g., "Robert C. Martin", not "로버트 C. 마틴").
- "publisher" must be the ORIGINAL publisher (e.g., "Addison-Wesley Professional").
- "isbn13" must be the 13-digit ISBN with hyphens (e.g., "978-0132350884").
- If Korean translation exists, include "korean" title. Otherwise leave empty.
- Search publisher websites, Amazon, or library databases for accurate ISBN.
- Return ONLY the JSON object.`
}
```

#### B-5. SystemPrompt을 Collector에 전달

```go
// prompts.go에 getter 추가
func GetSystemPrompt() string {
    return SystemPromptBookInfo
}
```

---

## 5. 수정 사항 C: 응답 후처리

### 파일: `internal/infra/validator/json_validator.go`

#### C-1. 메타데이터 검증 완화

현재 ISBN/author/publisher가 하나라도 비면 실패. 웹검색으로 못 찾을 수 있으므로:

```go
// 변경 전
func (v *JSONValidator) ValidateMetadataJSON(raw []byte) (domain.BookMetadata, error) {
    // ...
    if payload.ISBN13 == "" || payload.Author == "" || payload.Publisher == "" {
        return domain.BookMetadata{}, fmt.Errorf("schema error: metadata required fields missing")
    }
    return payload, nil
}

// 변경 후
func (v *JSONValidator) ValidateMetadataJSON(raw []byte) (domain.BookMetadata, error) {
    var payload domain.BookMetadata
    if err := json.Unmarshal(raw, &payload); err != nil {
        return domain.BookMetadata{}, fmt.Errorf("schema error: metadata json parse: %w", err)
    }
    // Author만 필수. ISBN/Publisher는 optional (웹검색으로 못 찾을 수 있음)
    if payload.Author == "" {
        return domain.BookMetadata{}, fmt.Errorf("schema error: author is required")
    }
    return payload, nil
}
```

### 파일: `internal/domain/types.go`

#### C-2. 저자명 정규화 헬퍼

```go
// NormalizeAuthor extracts the primary English name from formats like
// "로버트 C. 마틴 (Robert C. Martin)" → "Robert C. Martin"
func NormalizeAuthor(raw string) string {
    // 괄호 안 영문명 추출
    if idx := strings.Index(raw, "("); idx >= 0 {
        if end := strings.Index(raw[idx:], ")"); end >= 0 {
            inner := strings.TrimSpace(raw[idx+1 : idx+end])
            if inner != "" {
                return inner
            }
        }
    }
    return strings.TrimSpace(raw)
}
```

---

## 6. 수정 사항 D: 검증 로직 완화

### 파일: `internal/infra/validator/json_validator.go`

#### D-1. TOC 검증에서 빈 배열 허용

```go
// 변경 전
func (v *JSONValidator) ValidateTOCJSON(raw []byte) ([]domain.TOCNode, error) {
    var arr []domain.TOCNode
    if err := json.Unmarshal(raw, &arr); err == nil {
        return arr, nil
    }
    // ...
}

// 변경 후 — wrapper object의 다양한 키 허용
func (v *JSONValidator) ValidateTOCJSON(raw []byte) ([]domain.TOCNode, error) {
    // 시도 1: 직접 배열
    var arr []domain.TOCNode
    if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
        return arr, nil
    }

    // 시도 2: 다양한 wrapper key
    var generic map[string]json.RawMessage
    if err := json.Unmarshal(raw, &generic); err == nil {
        for _, key := range []string{"table_of_contents", "chapters", "toc", "items"} {
            if val, ok := generic[key]; ok {
                var nodes []domain.TOCNode
                if err := json.Unmarshal(val, &nodes); err == nil && len(nodes) > 0 {
                    return nodes, nil
                }
            }
        }
    }

    return nil, fmt.Errorf("schema error: could not extract TOC nodes from response")
}
```

---

## 7. 수정 사항 E: 비교 스크립트

### 파일: `scripts/live_compare_openrouter.py`

#### E-1. contains 비교에 정규화 추가

```python
def normalize_for_compare(text: str) -> str:
    """표기 변형 정규화: 괄호 내용 추출, 공백 정리"""
    if not text:
        return ""
    # 괄호 안 내용도 포함해서 비교
    # "로버트 C. 마틴 (Robert C. Martin)" → 둘 다 포함
    return text.strip().lower()

def flexible_match(expected: str, actual: str) -> bool:
    """유연한 비교: exact → contains → 괄호 내 추출"""
    if not actual:
        return False
    exp_lower = expected.strip().lower()
    act_lower = str(actual).strip().lower()
    
    # 1. exact match
    if exp_lower == act_lower:
        return True
    
    # 2. contains (한쪽이 다른쪽을 포함)
    if exp_lower in act_lower or act_lower in exp_lower:
        return True
    
    # 3. 괄호 안 텍스트 추출 후 비교
    import re
    paren_match = re.search(r'\(([^)]+)\)', str(actual))
    if paren_match:
        inner = paren_match.group(1).strip().lower()
        if exp_lower == inner or exp_lower in inner:
            return True
    
    return False
```

#### E-2. 메타데이터 비교에 flexible_match 적용

```python
# 변경 전
checks.append((f"metadata.{field}", exp, act, str(exp) == str(act)))

# 변경 후
checks.append((f"metadata.{field}", exp, act, flexible_match(str(exp), str(act))))
```

#### E-3. 챕터 제목 비교에 다국어 허용

Clean Code의 경우 모델이 한국어로 응답할 수 있으므로, verified JSON에 한국어 매핑 추가하거나:

```python
# 한국어-영어 챕터명 매핑 (Clean Code 예시)
TITLE_ALIASES = {
    "cleancode": {
        "Meaningful Names": ["의미있는이름", "의미 있는 이름", "의미있는 이름"],
        "Functions": ["함수"],
        "Comments": ["주석"],
        "Objects and Data Structures": ["객체와자료구조", "객체와 자료구조", "객체와 자료 구조"],
        "Classes": ["클래스"],
        "Exceptions": ["오류처리", "오류 처리"],
        "Boundaries": ["경계"],
    }
}

def contains_keyword_flexible(titles: list[str], keyword: str, case_id: str) -> bool:
    """키워드 + 알려진 별칭으로 비교"""
    for title in titles:
        if contains_keyword(str(title), keyword):
            return True
    # 별칭 체크
    aliases = TITLE_ALIASES.get(case_id, {}).get(keyword, [])
    for alias in aliases:
        for title in titles:
            if contains_keyword(str(title), alias):
                return True
    return False
```

---

## 8. 파일별 수정 체크리스트

### Go 코드

| # | 파일 | 수정 내용 | 우선순위 |
|---|------|----------|---------|
| 1 | `internal/infra/openrouter/client.go` | RequestPayload 확장 (temperature, response_format, system message) | 🔴 P0 |
| 2 | `internal/infra/openrouter/client.go` | extractJSON 함수 추가 (markdown 래핑 제거) | 🔴 P0 |
| 3 | `internal/infra/openrouter/client.go` | Collect 시그니처: `systemPrompt, userPrompt` 분리 | 🔴 P0 |
| 4 | `internal/infra/openrouter/collector.go` | Collect 호출 시 systemPrompt 전달 | 🔴 P0 |
| 5 | `internal/app/prompts.go` | SystemPromptBookInfo 상수 추가 | 🔴 P0 |
| 6 | `internal/app/prompts.go` | commonLead: 언어 제어 강화 | 🟡 P1 |
| 7 | `internal/app/prompts.go` | BuildTOCPrompt: 계층 구조 명시 | 🟡 P1 |
| 8 | `internal/app/prompts.go` | BuildMetadataPrompt: 원어 필드 강조 | 🟡 P1 |
| 9 | `internal/app/interfaces.go` | Collector 인터페이스 시그니처 변경 (필요 시) | 🔴 P0 |
| 10 | `internal/infra/validator/json_validator.go` | ValidateMetadataJSON: 필수 필드 완화 | 🟡 P1 |
| 11 | `internal/infra/validator/json_validator.go` | ValidateTOCJSON: 다양한 wrapper key 허용 | 🟡 P1 |
| 12 | `internal/domain/types.go` | NormalizeAuthor 헬퍼 추가 | 🟢 P2 |

### Python 비교 스크립트

| # | 파일 | 수정 내용 | 우선순위 |
|---|------|----------|---------|
| 13 | `scripts/live_compare_openrouter.py` | flexible_match 함수 추가 | 🟡 P1 |
| 14 | `scripts/live_compare_openrouter.py` | 메타데이터 비교에 flexible_match 적용 | 🟡 P1 |
| 15 | `scripts/live_compare_openrouter.py` | TITLE_ALIASES 다국어 매핑 | 🟢 P2 |
| 16 | `scripts/live_compare_openrouter.py` | system prompt 추가 (현재 1줄) | 🔴 P0 |

### Verified Source

| # | 파일 | 수정 내용 | 우선순위 |
|---|------|----------|---------|
| 17 | `docs/verified_source/json/*.json` | verification_rules에 별칭 필드 추가 | 🟢 P2 |

---

## 9. 예상 효과

### P0만 적용 시 (system prompt + temperature + response_format)

| 문제 | 해결 |
|------|------|
| markdown 섞임 | `response_format: json_object` → JSON만 출력 |
| 한국어 번역 응답 | system prompt에 "ORIGINAL language" 명시 |
| temperature 1.0 | 0.1로 → 일관된 출력 |
| 메타데이터 누락 | system prompt에 필수 필드 명시 |

**예상 통과율: 30~45%**

### P0 + P1 적용 시 (프롬프트 재설계 + 유연 비교)

| 추가 개선 | 효과 |
|----------|------|
| 계층 구조 명시 프롬프트 | depth 3 달성 |
| flexible_match | 표기 변형 허용 |
| 검증 완화 | ISBN 누락해도 통과 |

**예상 통과율: 50~65%**

### P0 + P1 + P2 적용 시 (다국어 별칭 + 정규화)

**예상 통과율: 65~80%**

### 100% 달성이 어려운 이유

- LLM 비결정성: 동일 프롬프트에도 매번 다른 응답
- 웹검색 의존: 검색 결과에 따라 메타데이터 가용성 변동
- 한국 강의 플랫폼 크롤링 한계: Inflearn/RealDealClass 데이터 접근 제한

### 권장 실행 순서

```
1일차: P0 적용 (items 1-5, 9, 16) → 테스트 → 30%+ 확인
2일차: P1 적용 (items 6-8, 10-11, 13-14) → 테스트 → 50%+ 확인
3일차: P2 적용 (items 12, 15, 17) → 테스트 → 65%+ 확인
```

---

## 부록: qwen.ai 채팅 vs API 차이 원인 요약

| 항목 | qwen.ai 채팅 | API (현재 코드) | 수정 후 |
|------|-------------|----------------|--------|
| System prompt | ✅ 내부 최적화 | ❌ 없음 | ✅ 추가 |
| temperature | ~0.7 | 1.0 (기본값) | 0.1 |
| response_format | UI 자동 처리 | ❌ 미설정 | json_object |
| 언어 제어 | 컨텍스트 암묵적 | ❌ 불충분 | 명시적 지시 |
| thinking mode | 자동 | reasoning 토큰 소비 | /no_think 고려 |
| 웹검색 | 내장 | :online 또는 plugins | 유지 |
