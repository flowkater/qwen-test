# DashScope Responses API 비교 리포트 (2026-03-02 20:10:45)

- 모델: `qwen3.5-flash`
- API: DashScope Responses (web_search + web_extractor)
- timeout: 300s

## cleancode
- 173.9s | searches:2 extracts:6 | tokens: in=304606 out=15737

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.author | Robert C. Martin | Robert C. Martin | ✅ |
| metadata.publisher | Addison-Wesley Professional | Addison-Wesley Professional | ✅ |
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
| max_depth | >=3 | 3 | ✅ |

**13/13 통과**

## mcat
- 191.5s | searches:9 extracts:6 | tokens: in=504884 out=3490

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.publisher | Kaplan | Kaplan Test Prep | ✅ |
| metadata.isbn13_biology | 978-1506297408 | 9781506297408 | ✅ |
| metadata.isbn13_biochemistry | 978-1506297385 | 9781506297385 | ✅ |
| bio_chapters | 12 | 12 | ✅ |
| biochem_chapters | 8 | 12 | ❌ |
| bio "Cell Structure and Functi" | 포함 | 포함 | ✅ |
| bio "Human Anatomy and Physiol" | 포함 | 포함 | ✅ |
| biochem "Enzymes" | 포함 | 포함 | ✅ |

**7/8 통과**

## inflearn
- 62.4s | searches:0 extracts:2 | tokens: in=31809 out=4958

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
- 43.4s | searches:1 extracts:1 | tokens: in=16641 out=529

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|

**0/0 통과**

## 전체 요약
- 총: **33/34**
- 통과율: **97.1%**