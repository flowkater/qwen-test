# Known Issues — bookinfo CLI

> 최종 업데이트: 2026-03-04
> API Quality Improvement Guide 16개 항목 구현 완료 후 NOCACHE 통합 테스트에서 발견된 문제

---

## 1. Clean Code ISBN enrichment 타임아웃

### 증상
- `978-0132350884` (Clean Code) ISBN으로 NOCACHE 실행 시 반복적으로 타임아웃 발생
- DashScope 408 응답 또는 컨텍스트 deadline 초과
- 180초, 300초 타임아웃으로도 완료 불가

### 근본 원인
`enrichTOCParts`가 sparse part마다 **순차적**으로 API 호출:

```
collectUnified → 8개 sparse chapter 감지 → enrichTOCParts
  → collectPartTOC(Chapter 1) → 10~60초 + retry 최대 3회
  → collectPartTOC(Chapter 2) → 10~60초 + retry 최대 3회
  → ... (총 8회 순차)
```

- 최악의 경우: 8 parts × 3 retries × 60초 = **24분**
- DashScope Responses API의 web_search 도구 사용 시 응답 시간이 불규칙 (10~90초)

### 영향 범위
- 대형 도서 (8+ chapters가 모두 sparse인 경우)
- Clean Code처럼 depth-1이 많고 unified 호출에서 section 정보를 잘 가져오지 못하는 케이스

### 관련 코드
- `internal/app/service.go:291-320` (`enrichTOCParts` — 순차 루프)
- `internal/app/service.go:332-350` (`sparseParts` — sparse 판별)

### 해결 방향
- [ ] enrichment 병렬화 (`errgroup` 활용, rate limit 고려)
- [ ] sparse part 수 제한 (예: 최대 5개만 enrichment)
- [ ] 전체 sparse인 경우 enrichment 스킵하고 unified 결과 그대로 사용

---

## 2. LLM 비결정성으로 인한 NOCACHE 품질 편차

### 증상
동일한 쿼리를 NOCACHE로 실행할 때마다 결과 품질이 크게 달라짐:

| 케이스 | 캐시 (best) | NOCACHE (최근) | 차이 |
|--------|------------|---------------|------|
| 인프런 시스템디자인 | 4 sections / 24 lectures | 3 parts / 7 lectures | **-70% 콘텐츠** |
| 리얼딜클래스 | 12 depth-1 / 47 depth-2 | 5 parts / 13 lectures | **-72% 콘텐츠** |
| MCAT | 7 Parts / 126 items | 7 Parts / 105+ items | 양호 |
| Clean Code | 25 depth-1 / 215 depth-2 | 타임아웃 | 실패 |

### 근본 원인
1. **DashScope LLM의 비결정적 출력**: temperature 0으로 설정해도 web_search 도구 결과에 따라 응답 변동
2. **웹검색 결과 변동**: 같은 쿼리여도 검색 결과 순서/가용성이 매번 달라짐
3. **한국어 강의 플랫폼 크롤링 한계**: 인프런/리얼딜클래스의 커리큘럼 데이터가 웹검색으로 안정적으로 접근 불가
4. **캐시가 문제를 숨김**: 이전 "운 좋은" 호출 결과가 캐시에 저장되어 품질이 좋아 보였음

### 캐시 분석 결과 (`~/.cache/bookinfo/`)
```
인프런:
  - 캐시 A: 4 sections, 24 lectures (good) ← 운 좋은 호출
  - 캐시 B: 3 sections, 7 lectures (sparse) ← 운 나쁜 호출

리얼딜클래스:
  - 캐시 A: 12 depth-1, 47 depth-2 (good)
  - 캐시 B: 5 depth-1, 17 depth-2 (sparse)
```

### 영향 범위
- 모든 NOCACHE 호출
- 특히 한국어 강의 데이터 (웹검색 의존도 높음)

### 관련 코드
- `internal/app/service.go:442-446` (`collectUnified` — 1차 호출)
- `internal/infra/openrouter/client.go` (DashScope HTTP 클라이언트)

### 해결 방향
- [ ] 품질 인식 재시도 (quality-aware retry): TOC depth-1 개수가 임계값 미달 시 자동 재시도
- [ ] Best-of-N 선택: N회 호출 후 가장 풍부한 결과 선택
- [ ] 프롬프트 강화: 최소 예상 chapter 수를 프롬프트에 포함
- [ ] 2-pass 전략 개선: unified가 sparse하면 metadata만 취하고 TOC는 별도 호출로 재시도

---

## 3. enrichment 2nd-pass가 1st-pass 부족을 보완하지 못함

### 증상
- 1차 `collectUnified`에서 top-level 구조 자체가 부족하면 (예: Part가 3개만 반환), 2nd-pass `enrichTOCParts`가 **존재하는 part만** enrichment
- 누락된 Part/Section은 복구 불가

### 예시
```
기대: Part 1, Part 2, Part 3, Part 4  (4 parts)
1차 결과: Part 1, Part 2, Part 3       (3 parts — Part 4 누락)
2차 결과: Part 1(enriched), Part 2(enriched), Part 3(enriched)
→ Part 4는 영원히 누락
```

### 근본 원인
- `enrichTOCParts`는 **기존 depth-1 노드의 children만 보강**
- 누락된 depth-1 노드를 추가하는 메커니즘 없음
- `sparseParts`는 children이 비어있는 노드만 대상, 아예 없는 노드는 감지 불가

### 관련 코드
- `internal/app/service.go:291-320` (`enrichTOCParts`)

### 해결 방향
- [ ] 1차 결과 품질 게이트: depth-1 개수가 기대값 대비 부족하면 unified 재호출
- [ ] verified source에 예상 depth-1 개수 정보 추가하여 품질 판단 기준으로 활용
- [ ] 별도 TOC-only 호출로 fallback (unified 실패 시 metadata/TOC 분리 호출)

---

## 부록: 테스트 결과 타임라인

### 2026-03-04 NOCACHE 통합 테스트

| # | 케이스 | 명령 | 결과 | 비고 |
|---|--------|------|------|------|
| 1 | 인프런 시스템디자인 | `make run-inflearn-system EXTRA="--no-cache"` | depth-3=0, 3 parts/7 lectures | sparse |
| 2 | 리얼딜클래스 | `make run-realdeal EXTRA="--no-cache"` | depth-3=0, 5 parts/13 lectures | sparse |
| 3 | MCAT | `make run-mcat EXTRA="--no-cache"` | 7 Parts/35 Ch/105 Subsections | 양호 |
| 4 | Clean Code | `make run-cleancode EXTRA="--no-cache"` | 타임아웃 (반복 실패) | Issue #1 |

### 이전 커밋 (구현 완료)
- `8e6e059` — lecture truncation (IsLecture + truncateTOCDepth)
- `3ccb103` — NormalizeAuthor 통합 + 비교 스크립트 CLI 모드
