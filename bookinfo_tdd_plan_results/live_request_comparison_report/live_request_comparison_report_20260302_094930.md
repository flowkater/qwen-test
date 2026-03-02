# Verified Source vs 실요청 실측 비교 리포트 (2026-03-02 09:49:30 KST)

- 모델: `qwen/qwen3.5-flash-02-23`
- search_mode: `auto`
- prompt_mode: `two-step-json`
- timeout: `120s`
- retry: `max_retries=2, base_delay=2.0s`
- API: OpenRouter 실요청
- plugins: [{"id": "web", "engine": "native", "max_results": 5}]
- 비교 기준: `docs/verified_source/json/*`의 `verification_rules`
- 주의: LLM 응답 비결정성으로 exact 비교는 매 실행 결과가 달라질 수 있음

## 도구 사용 통계 (web_search_requests)

| 케이스 | mode_used | web_search_requests | 비고 |
|---|---|---:|---|
| cleancode | online(fallback) + mapper | N/A | 요청 성공 |
| mcat | online(fallback) + mapper | N/A | 요청 성공 |
| inflearn | online(fallback) + mapper | N/A | 요청 성공 |
| realdeal | online(fallback) + mapper (retry=1) | N/A | 요청 성공 |

## 케이스: cleancode (기준: cleancode-en-book-toc.json)
- 요청 성공: 예
- 응답 시간: 40.00s

| 항목 | 기준값 | 실측값 | 결과 |
|---|---|---|---|
| metadata.author | Robert C. Martin | 로버트 C. 마틴 (Robert C. Martin) | ❌ |
| metadata.publisher | Addison-Wesley Professional |  | ❌ |
| metadata.isbn13 | 978-0132350884 |  | ❌ |
| table_of_contents.total_chapters | 8 | 17 | ❌ |
| table_of_contents.total_appendices | 2 | 3 | ❌ |
| chapter_titles contains "Meaningful Names" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Functions" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Comments" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Objects and Data Structures" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Classes" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Exceptions" | 포함 | 미포함 | ❌ |
| chapter_titles contains "Boundaries" | 포함 | 미포함 | ❌ |
| table_of_contents.max_depth_observed | 3 | 1 | ❌ |

- 소계: **0/13 통과**

## 케이스: mcat (기준: mcat-en-book-toc.json)
- 요청 성공: 예
- 응답 시간: 54.06s

| 항목 | 기준값 | 실측값 | 결과 |
|---|---|---|---|
| metadata.publisher | Kaplan Test Prep |  | ❌ |
| metadata.isbn13_biology | 978-1506297408 |  | ❌ |
| metadata.isbn13_biochemistry | 978-1506297385 |  | ❌ |
| table_of_contents.biology_chapters | 8 | 56 | ❌ |
| table_of_contents.biochemistry_chapters | 8 | 4 | ❌ |
| biology_chapter_titles contains "Cell Structure and Function" | 포함 | 미포함 | ❌ |
| biology_chapter_titles contains "Human Anatomy and Physiology" | 포함 | 미포함 | ❌ |
| biochemistry_chapter_titles contains "Enzymes and Kinetics" | 포함 | 미포함 | ❌ |
| table_of_contents.max_depth_observed | 3 | 0 | ❌ |

- 소계: **0/9 통과**

## 케이스: inflearn (기준: inflearn-system-kr.json)
- 요청 성공: 예
- 응답 시간: 35.72s

| 항목 | 기준값 | 실측값 | 결과 |
|---|---|---|---|
| metadata.instructor | mindlantern |  | ❌ |
| metadata.platform | Inflearn | Inflearn | ✅ |
| metadata.rating | 4.9 | 0 | ❌ |
| metadata.total_lectures | 24 | 0 | ❌ |
| curriculum.total_sections | 4 | 0 | ❌ |
| curriculum.section_lecture_counts | [4, 6, 9, 5] | [] | ❌ |
| section_titles contains "Why Learn System Design" | 포함 | 미포함 | ❌ |
| section_titles contains "4 Key Goals of System Design" | 포함 | 미포함 | ❌ |
| section_titles contains "Main System Components and Trade-offs" | 포함 | 미포함 | ❌ |
| section_titles contains "Designing and Explaining My Own Architecture" | 포함 | 미포함 | ❌ |
| key_lectures[3.3].title | API Gateway & Load Balancer & Service Discovery |  | ❌ |
| key_lectures[3.3].duration | 16:01 |  | ❌ |
| key_lectures[3.7].title | Message Queue & Event Broker |  | ❌ |
| key_lectures[3.7].duration | 32:12 |  | ❌ |

- 소계: **1/14 통과**

## 케이스: realdeal (기준: realdealclass-kr-lecture.json)
- 요청 성공: 예
- 응답 시간: 52.67s

| 항목 | 기준값 | 실측값 | 결과 |
|---|---|---|---|
| metadata.platform | RealDealClass | realdealclass.com | ❌ |
| metadata.difficulty | (왕)초급 - 초중급 |  | ❌ |
| metadata.duration | 6개월 |  | ❌ |
| curriculum.total_chapters | 7 | 0 | ❌ |
| curriculum.has_special_lessons | True | False | ❌ |
| curriculum.special_lesson_count | 5 | 0 | ❌ |
| chapter_titles contains "문장의 형태" | 포함 | 포함 | ✅ |
| chapter_titles contains "동사의 종류" | 포함 | 미포함 | ❌ |
| chapter_titles contains "동사의 시제" | 포함 | 미포함 | ❌ |
| chapter_titles contains "수동태" | 포함 | 미포함 | ❌ |
| chapter_titles contains "준동사 & 가짜 동사" | 포함 | 미포함 | ❌ |
| chapter_titles contains "긴 문장 만드는 원리" | 포함 | 미포함 | ❌ |
| chapter_titles contains "풍부한 문장 완성하기" | 포함 | 미포함 | ❌ |
| key_lectures[1].title | 문법의 개념 | 문장의 형태 (문법의 개념) | ❌ |
| key_lectures[1].duration | 37:23 | 37:23 | ✅ |
| key_lectures[27].title | to-V 1 |  | ❌ |
| key_lectures[27].duration | 42:20 |  | ❌ |

- 소계: **2/17 통과**

## 전체 요약
- 총 통과: **3/53**
- 통과율: **5.7%**

## 산출물
- 상세 리포트: `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_comparison_report_20260302_094930.md`
- 원본 응답(raw): `/Users/flowkater/workspace/side/qwen/bookinfo_tdd_plan_results/live_request_raw_20260302_094930.json`
- 실행 기준 root: `/Users/flowkater/workspace/side/qwen`