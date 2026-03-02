# 실요청 비교 리포트 실행 가이드

## 목적
`docs/verified_source/json/*` 기준값과 OpenRouter 실요청 결과를 비교해 한글 리포트를 생성합니다.

## 실행 스크립트
- `scripts/live_compare_openrouter.py`

## 사전 준비
1. OpenRouter API 키 설정
   - 환경변수: `OPENROUTER_API_KEY=...`
   - 또는 루트 `.env`에 `OPENROUTER_API_KEY=...`

## 기본 실행
```bash
python3 scripts/live_compare_openrouter.py
```

기본값:
- model: `qwen/qwen3.5-flash-02-23`
- search_mode: `auto` (plugins 우선, 미지원 시 `:online` 자동 fallback)
- prompt_mode: `strict-json`
- plugins: `[{\"id\":\"web\",\"engine\":\"native\",\"max_results\":5}]`
- cases: `cleancode,mcat,inflearn,realdeal`
- output-dir: `bookinfo_tdd_plan_results`
- timeout: `120s`
- retry: `max_retries=2`, `retry_delay=2s` (429/timeout 중심)

## 옵션 예시
```bash
python3 scripts/live_compare_openrouter.py \
  --model qwen/qwen3.5-flash-02-23 \
  --search-mode auto \
  --prompt-mode two-step-json \
  --engine native \
  --max-results 5 \
  --max-retries 2 \
  --retry-delay 2 \
  --cases cleancode,mcat \
  --output-dir bookinfo_tdd_plan_results
```

## prompt_mode 설명
- `strict-json`: 처음부터 JSON 스키마 강제
- `soft-json`: JSON 우선 요청(완화된 강제)
- `two-step-json`: 1차 자유형 수집 → 2차 JSON 매핑

## 산출물
- 비교 리포트: `live_request_comparison_report_<timestamp>.md`
- 원본 응답: `live_request_raw_<timestamp>.json`

리포트에는 아래 정보가 포함됩니다.
- 실행 파라미터(모델/플러그인)
- 케이스별 `web_search_requests` 통계
- 검증 항목별 pass/fail 표
- 총 통과율

## 자주 발생하는 오류
1. `API 키를 찾을 수 없습니다`
   - `OPENROUTER_API_KEY` 값 확인
2. `HTTPError 400` 모델 오류
   - 모델 ID 오타 여부 확인
3. `HTTPError 404` native web search 미지원
   - `--search-mode online` 또는 `--search-mode auto` 사용
4. 파싱 실패
   - 모델 응답이 JSON 외 텍스트를 포함한 케이스로, 재실행하거나 프롬프트를 더 엄격히 조정
5. `TimeoutError request exceeded ...`
   - `--timeout` 값을 늘려 재시도
6. `HTTPError 429`
   - 잠시 후 재시도하거나 `--max-retries`, `--retry-delay`를 늘려 실행
