# DashScope Responses API 비교 리포트 (2026-03-02 19:48:10)

- 모델: `qwen3.5-flash`
- API: DashScope Responses (web_search + web_extractor)
- timeout: 180s

## cleancode
- 76.6s | searches:3 extracts:3 | tokens: in=125200 out=1156

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.author | Robert C. Martin | Robert C. Martin | ✅ |
| metadata.publisher | Addison-Wesley Professional | Addison-Wesley Professional (Pearson Education) | ✅ |
| metadata.isbn13 | 978-0132350884 | 9780132350884 | ✅ |
| total_chapters | 17 | 17 | ✅ |
| total_appendices | 3 | 3 | ✅ |
| "Meaningful Names" | 포함 | 포함 | ✅ |
| "Functions" | 포함 | 포함 | ✅ |
| "Comments" | 포함 | 포함 | ✅ |
| "Objects and Data Structures" | 포함 | 포함 | ✅ |
| "Classes" | 포함 | 포함 | ✅ |
| "Exceptions" | 포함 | 포함 | ✅ |
| "Boundaries" | 포함 | 포함 | ✅ |
| max_depth | >=3 | 1 | ❌ |

**12/13 통과**

## mcat
- FAILED: The read operation timed out

## inflearn
- 53.2s | searches:0 extracts:1 | tokens: in=11351 out=3040

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.instructor | mindlantern | mindlantern | ✅ |
| metadata.platform | Inflearn | Inflearn | ✅ |
| metadata.rating | 4.9 | 4.9 | ✅ |
| metadata.total_lectures | 24 | 24 | ✅ |
| total_sections | 4 | 4 | ✅ |
| sec "Why Learn System Design" | 포함 | 포함 | ✅ |
| sec "4 Key Goals of System Des" | 포함 | 포함 | ✅ |
| sec "Main System Components an" | 포함 | 포함 | ✅ |
| sec "Designing and Explaining " | 포함 | 포함 | ✅ |
| lec[3.3].title |  | API Gateway & Load Balancer & Service Discovery | ✅ |
| lec[3.3].duration |  | 16:01 | ✅ |
| lec[3.7].title |  | Message Queue & Event Broker | ✅ |
| lec[3.7].duration |  | 32:12 | ✅ |

**13/13 통과**

## realdeal
- 110.3s | searches:3 extracts:4 | tokens: in=180355 out=5461

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.platform | RealDealClass | Real Deal Class | ✅ |
| metadata.difficulty | (왕)초급 - 초중급 | 왕초보-초중급 | ❌ |
| metadata.duration | 6개월 | 6개월 | ✅ |
| total_chapters | 7 | 45 | ❌ |
| has_special | True | True | ✅ |
| "문장의 형태" | 포함 | 미포함 | ❌ |
| "동사의 종류" | 포함 | 포함 | ✅ |
| "동사의 시제" | 포함 | 포함 | ✅ |
| "수동태" | 포함 | 포함 | ✅ |
| "준동사 & 가짜 동사" | 포함 | 미포함 | ❌ |
| "긴 문장 만드는 원리" | 포함 | 미포함 | ❌ |
| "풍부한 문장 완성하기" | 포함 | 미포함 | ❌ |
| lec[1].title |  | 문법의 개념 | ✅ |
| lec[1].duration |  | 37:23 | ✅ |
| lec[27].title |  | 동사의 변형 – 준동사 – to-V 1 | ✅ |
| lec[27].duration |  | 42:20 | ✅ |

**10/16 통과**

## 전체 요약
- 총: **35/42**
- 통과율: **83.3%**