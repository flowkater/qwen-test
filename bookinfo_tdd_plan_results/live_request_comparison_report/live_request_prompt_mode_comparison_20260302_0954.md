# 실요청 프롬프트 모드 비교 리포트 (2026-03-02 09:54 KST)

## 목적
"JSON 강제 매핑 때문에 점수가 낮을 수 있다" 가설을 검증하기 위해 동일 모델/동일 케이스에서 프롬프트 모드를 바꿔 실측 비교.

## 실행 조건
- 모델: `qwen/qwen3.5-flash-02-23`
- search_mode: `auto` (plugins 실패 시 `:online` fallback)
- timeout: `120s`
- retry: `max_retries=2`, `retry_delay=2s`
- 케이스: `cleancode, mcat, inflearn, realdeal`
- 비교 기준: `docs/verified_source/json/*`의 `verification_rules`

## 프롬프트 모드 정의
- `strict-json`: 1단계에서 JSON 스키마 강제
- `soft-json`: JSON 우선 요청(강제 완화)
- `two-step-json`: 1단계 자유형 수집 + 2단계 JSON 매핑

## 실행 리포트
- strict-json: `live_request_comparison_report_20260302_095207.md`
- soft-json: `live_request_comparison_report_20260302_095318.md`
- two-step-json: `live_request_comparison_report_20260302_094930.md`

## 총점 비교
| prompt_mode | 총 통과 | 통과율 |
|---|---:|---:|
| strict-json | 7/53 | 13.2% |
| soft-json | 1/53 | 1.9% |
| two-step-json | 3/53 | 5.7% |

## 케이스별 소계
| 케이스 | strict-json | soft-json | two-step-json |
|---|---:|---:|---:|
| cleancode | 2/13 | 0/13 | 0/13 |
| mcat | 1/9 | 0/9 | 0/9 |
| inflearn | 1/14 | 0/14 | 1/14 |
| realdeal | 3/17 | 1/17 | 2/17 |

## 해석
1. 이번 실측에서는 `strict-json`이 가장 높은 점수를 기록.
2. 즉, "JSON 강제라서 점수가 낮다" 가설이 이번 샘플에서는 **직접적으로 입증되지는 않음**.
3. `soft-json`은 출력 자유도가 커져 오히려 검증 스키마와의 정합도가 더 낮아짐.
4. `two-step-json`은 일부 케이스(RealDeal)에서 개선 여지가 보이지만, 전체적으로는 strict-json보다 낮음.
5. 결론적으로 현재 지표(exact/구조 중심)에서는 프롬프트 완화만으로 성능 개선이 일관되지 않음.

## 후속 제안
- 지표를 2축으로 분리:
  - A) strict schema exact pass(현재)
  - B) 의미/근사 pass(문자열 정규화, 동의어/표기 변형 허용)
- `two-step-json` 2단계 매퍼를 provider 분리(예: 다른 모델)하여 재실험
- 케이스당 N회(예: 5회) 반복 실행 후 평균/분산 비교 (단일 샷 편차 완화)
