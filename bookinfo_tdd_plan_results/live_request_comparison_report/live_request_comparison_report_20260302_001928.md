# Verified Source vs 실요청 실측 비교 리포트 (2026-03-02 00:19:29 KST)

- 모델: `qwen/qwen3.5-flash`
- API: OpenRouter 실요청
- 비교 기준: `docs/verified_source/json/*`의 `verification_rules`
- 주의: LLM 응답은 비결정적이므로 exact 일치가 항상 보장되지는 않음

## 케이스: cleancode (기준: cleancode-en-book-toc.json)
- 요청 성공: 아니오
- 응답 시간: 0.53s
- 오류: `HTTPError 400`

## 케이스: mcat (기준: mcat-en-book-toc.json)
- 요청 성공: 아니오
- 응답 시간: 0.26s
- 오류: `HTTPError 400`

## 케이스: inflearn (기준: inflearn-system-kr.json)
- 요청 성공: 아니오
- 응답 시간: 0.04s
- 오류: `HTTPError 400`

## 케이스: realdeal (기준: realdealclass-kr-lecture.json)
- 요청 성공: 아니오
- 응답 시간: 0.04s
- 오류: `HTTPError 400`

## 전체 요약
- 총 통과: **0/0**
- 통과율: N/A

## 산출물
- 상세 리포트: `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_comparison_report_20260302_001928.md`
- 원본 응답(raw): `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_raw_20260302_001928.json`