# Verified Source vs 실요청 실측 비교 리포트 (2026-03-02 18:10:06 KST)

- 모델: `qwen/qwen3.5-flash-02-23`
- search_mode: `auto`
- prompt_mode: `strict-json`
- timeout: `90s`
- retry: `max_retries=1, base_delay=1.0s`
- API: OpenRouter 실요청
- plugins: [{"id": "web", "engine": "native", "max_results": 5}]
- 비교 기준: `docs/verified_source/json/*`의 `verification_rules`
- 주의: LLM 응답 비결정성으로 exact 비교는 매 실행 결과가 달라질 수 있음

## 도구 사용 통계 (web_search_requests)

| 케이스 | mode_used | web_search_requests | 비고 |
|---|---|---:|---|
| cleancode | online(fallback) | N/A | 요청 성공 |

## 케이스: cleancode (기준: cleancode-en-book-toc.json)
- 요청 성공: 예
- 응답 시간: 47.22s

| 항목 | 기준값 | 실측값 | 결과 |
|---|---|---|---|
| metadata.author | Robert C. Martin | Robert C. Martin | ✅ |
| metadata.publisher | Addison-Wesley Professional |  | ❌ |
| metadata.isbn13 | 978-0132350884 |  | ❌ |
| table_of_contents.total_chapters | 8 | 17 | ❌ |
| table_of_contents.total_appendices | 2 | 0 | ❌ |
| chapter_titles contains "Meaningful Names" | 포함 | 포함 | ✅ |
| chapter_titles contains "Functions" | 포함 | 포함 | ✅ |
| chapter_titles contains "Comments" | 포함 | 포함 | ✅ |
| chapter_titles contains "Objects and Data Structures" | 포함 | 포함 | ✅ |
| chapter_titles contains "Classes" | 포함 | 포함 | ✅ |
| chapter_titles contains "Exceptions" | 포함 | 포함 | ✅ |
| chapter_titles contains "Boundaries" | 포함 | 포함 | ✅ |
| table_of_contents.max_depth_observed | 3 | 1 | ❌ |

- 소계: **8/13 통과**

## 전체 요약
- 총 통과: **8/13**
- 통과율: **61.5%**

## 산출물
- 상세 리포트: `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_comparison_report_20260302_181006.md`
- 원본 응답(raw): `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_raw_20260302_181006.json`
- 실행 기준 root: `/Users/flowkater/workspace/side/qwen`