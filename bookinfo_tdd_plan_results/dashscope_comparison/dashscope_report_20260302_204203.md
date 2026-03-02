# DashScope Responses API 비교 리포트 (2026-03-02 20:42:03)

- 모델: `qwen3.5-flash`
- API: DashScope Responses (web_search + web_extractor)
- timeout: 300s

## cleancode
- 49.6s | searches:1 extracts:3 | tokens: in=84077 out=963

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
- 201.8s | searches:8 extracts:8 | tokens: in=696122 out=3659

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|
| metadata.publisher | Kaplan | Kaplan Test Prep | ✅ |
| metadata.isbn13_biology | 978-1506297408 | 9781506297408 | ✅ |
| metadata.isbn13_biochemistry | 978-1506297385 | 9781506297385 | ✅ |
| bio_chapters | 12 | 12 | ✅ |
| biochem_chapters | 12 | 12 | ✅ |
| bio "Cell Structure and Functi" | 포함 | 포함 | ✅ |
| bio "Human Anatomy and Physiol" | 포함 | 포함 | ✅ |
| biochem "Enzymes" | 포함 | 포함 | ✅ |

**8/8 통과**

## inflearn
- 32.0s | searches:0 extracts:1 | tokens: in=11345 out=2962

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
- 34.6s | searches:1 extracts:1 | tokens: in=16638 out=954

| 항목 | 기준 | 실측 | 결과 |
|---|---|---|---|

**0/0 통과**

## 전체 요약
- 총: **34/34**
- 통과율: **100.0%**