#!/usr/bin/env python3
"""
DashScope Responses API (web_search + web_extractor) vs Verified Source 비교 스크립트
핵심: OpenRouter :online은 검색 스니펫만, DashScope Responses API는 실제 페이지 크롤링
"""
from __future__ import annotations
import json, os, re, sys, time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

DASHSCOPE_BASE = "https://dashscope-intl.aliyuncs.com/api/v2/apps/protocols/compatible-mode/v1/responses"
DEFAULT_MODEL = "qwen3.5-flash-2026-02-23"
CASE_ORDER = ["cleancode", "mcat", "inflearn", "realdeal"]
CASE_TO_FIXTURE = {
    "cleancode": "cleancode-en-book-toc.json",
    "mcat": "mcat-en-book-toc.json",
    "inflearn": "inflearn-system-kr.json",
    "realdeal": "realdealclass-kr-lecture.json",
}
SYSTEM_PROMPT = """You are a precise metadata and table-of-contents/curriculum extraction engine.
CRITICAL RULES:
1. Use web search and web extractor to visit actual pages and extract accurate data.
2. Return ONLY valid JSON matching the user-requested schema. No markdown, no explanation, no code fences.
3. Preserve titles in the ORIGINAL language of the source resource.
4. If data is uncertain, leave placeholders empty ("", 0, [], false) instead of inventing.
5. Do NOT embed source citations or URLs inside JSON values. Keep values as clean data.
6. Extract as much detail as possible - visit the actual book/course page to get full TOC/curriculum."""

CASE_PROMPTS = {
    "cleancode": (
        'Find COMPLETE table of contents for "Clean Code: A Handbook of Agile Software Craftsmanship" '
        "by Robert C. Martin (English original edition, Addison-Wesley).\n"
        "Visit the book page to extract the FULL chapter list including appendices.\n"
        "IMPORTANT: Extract hierarchical structure — chapters AND their sections/subsections.\n"
        "max_depth_observed should be the deepest nesting level found (1=chapter only, 2=chapter>section, 3=chapter>section>subsection).\n"
        "Return JSON:\n"
        '{"metadata":{"author":"","publisher":"","isbn13":""},'
        '"table_of_contents":{"total_chapters":0,"total_appendices":0,'
        '"chapter_titles":[],"max_depth_observed":0}}'
    ),
    "mcat": (
        "Find COMPLETE table of contents for Kaplan MCAT Biology Review AND Biochemistry Review "
        "(latest edition, 2025-2026 or 2026-2027).\n"
        "Visit the Kaplan or Amazon page to extract the FULL chapter lists for BOTH books.\n"
        "Return JSON:\n"
        '{"metadata":{"publisher":"","isbn13_biology":"","isbn13_biochemistry":""},'
        '"table_of_contents":{"biology_chapters":0,"biochemistry_chapters":0,'
        '"biology_chapter_titles":[],"biochemistry_chapter_titles":[],"max_depth_observed":0}}'
    ),
    "inflearn": (
        '"시스템 디자인 첫걸음" 인프런(Inflearn) 강의의 상세 커리큘럼을 찾아줘.\n'
        "https://www.inflearn.com/course/시스템-디자인-첫걸음 페이지를 직접 방문해서 "
        "섹션별 강의 목록, 강의 수, 강의 시간을 추출해.\n"
        "Return JSON:\n"
        '{"metadata":{"instructor":"","platform":"","rating":0,"total_lectures":0},'
        '"curriculum":{"total_sections":0,"section_titles":[],"section_lecture_counts":[],'
        '"key_lectures":{"3.3":{"title":"","duration":""},"3.7":{"title":"","duration":""}}}}'
    ),
    "realdeal": (
        '"리얼딜 클라쓰 ZERO TO ONE" 영어 문법 강의의 상세 커리큘럼을 찾아줘.\n'
        "https://realdealclass.com/product/zero-to-one/ 페이지를 직접 방문해서 커리큘럼을 추출해.\n"
        'IMPORTANT: 개별 강의가 아닌 챕터(대분류) 단위로 묶어줘. 예: "동사의 종류", "동사의 시제" 등.\n'
        "total_chapters는 대분류 챕터 수 (개별 강의 수가 아님).\n"
        "chapter_titles는 대분류 챕터 제목 리스트.\n"
        "Return JSON:\n"
        '{"metadata":{"platform":"","difficulty":"","duration":""},'
        '"curriculum":{"total_chapters":0,"has_special_lessons":false,"special_lesson_count":0,'
        '"chapter_titles":[],"key_lectures":{"1":{"title":"","duration":""},"27":{"title":"","duration":""}}}}'
    ),
}


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def load_api_key(root: Path) -> str:
    key = os.environ.get("DASHSCOPE_API_KEY", "").strip()
    if key:
        return key
    env_file = root / ".env"
    if env_file.exists():
        for line in env_file.read_text().splitlines():
            text = line.strip()
            if text.startswith("#") or "=" not in text:
                continue
            k, v = text.split("=", 1)
            if k.strip() == "DASHSCOPE_API_KEY":
                return v.strip().strip('"').strip("'")
    return ""


def extract_json_from_text(text: str) -> dict[str, Any] | None:
    text = text.strip()
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass
    m = re.search(r"```(?:json)?\s*(\{[\s\S]*?\})\s*```", text)
    if m:
        try:
            return json.loads(m.group(1))
        except json.JSONDecodeError:
            pass
    start = text.find("{")
    if start >= 0:
        depth = 0
        for i in range(start, len(text)):
            if text[i] == "{":
                depth += 1
            elif text[i] == "}":
                depth -= 1
                if depth == 0:
                    try:
                        return json.loads(text[start : i + 1])
                    except json.JSONDecodeError:
                        break
    return None


def call_dashscope(
    api_key: str, model: str, case_id: str, timeout: int
) -> dict[str, Any]:
    payload = {
        "model": model,
        "input": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": CASE_PROMPTS[case_id]},
        ],
        "tools": [{"type": "web_search"}, {"type": "web_extractor"}],
        "temperature": 0.1,
    }
    req = Request(DASHSCOPE_BASE, data=json.dumps(payload).encode(), method="POST")
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")
    started = time.time()
    try:
        with urlopen(req, timeout=timeout) as resp:
            raw = json.loads(resp.read().decode())
            elapsed = time.time() - started
    except HTTPError as e:
        body = e.read().decode("utf-8", errors="ignore")
        return {
            "ok": False,
            "elapsed": time.time() - started,
            "error": f"HTTP {e.code}: {body[:300]}",
        }
    except Exception as e:
        return {"ok": False, "elapsed": time.time() - started, "error": str(e)}

    searches = extracts = 0
    response_text = ""
    for item in raw.get("output", []):
        t = item.get("type", "")
        if t == "web_search_call":
            searches += 1
        elif t == "web_extractor_call":
            extracts += 1
        elif t == "message":
            for c in item.get("content", []):
                if c.get("type") == "output_text":
                    response_text = c.get("text", "")
    parsed = extract_json_from_text(response_text) if response_text else None
    return {
        "ok": True,
        "elapsed": elapsed,
        "parsed": parsed,
        "response_text": response_text,
        "searches": searches,
        "extracts": extracts,
        "usage": raw.get("usage", {}),
    }


def normalize(text: Any) -> str:
    return re.sub(r"\s+", " ", str(text or "").strip().lower())


def flexible_match(expected: str, actual: str) -> bool:
    e, a = normalize(expected), normalize(actual)
    if not e or not a:
        return False
    if e == a or e in a or a in e:
        return True
    ec = re.sub(r"[\s\W_]+", "", e)
    ac = re.sub(r"[\s\W_]+", "", a)
    if ec and ac and (ec in ac or ac in ec):
        return True
    return False


TITLE_ALIASES = {
    "cleancode": {
        "Meaningful Names": ["의미있는이름"],
        "Functions": ["함수"],
        "Comments": ["주석"],
        "Objects and Data Structures": ["객체와 자료구조"],
        "Classes": ["클래스"],
        "Exceptions": ["오류 처리", "Error Handling"],
        "Boundaries": ["경계"],
    },
    "mcat": {
        "Cell Structure and Function": ["The cell", "Cell Biology"],
        "Human Anatomy and Physiology": ["The nervous system", "Homeostasis"],
        "Enzymes and Kinetics": ["Enzymes", "Enzyme kinetics", "Bioenergetics"],
    },
}
FIELD_ALIASES = {
    "inflearn": {
        "instructor": ["mindlantern", "성장랜턴"],
        "platform": ["Inflearn", "인프런"],
    },
    "realdeal": {
        "platform": ["RealDealClass", "리얼딜 클라쓰", "리얼딜클라쓰"],
        "difficulty": [
            "(왕)초급 - 초중급",
            "왕초보-초중급",
            "왕초보",
            "왕초급",
            "초급-초중급",
        ],
    },
}


def kw_match(titles, keyword, case_id):
    candidates = [keyword] + TITLE_ALIASES.get(case_id, {}).get(keyword, [])
    return any(flexible_match(c, str(t)) for t in titles for c in candidates)


def field_match(case_id, field, expected, actual):
    if flexible_match(str(expected), str(actual)):
        return True
    return any(
        flexible_match(a, str(actual))
        for a in FIELD_ALIASES.get(case_id, {}).get(field, [])
    )


def compare_case(case_id, fixture, actual):
    checks = []
    rules = fixture["verification_rules"]
    if case_id == "cleancode":
        mr = rules["metadata_exact_match"]
        am = actual.get("metadata", {})
        for f in mr["fields"]:
            checks.append(
                (
                    f"metadata.{f}",
                    mr[f],
                    am.get(f),
                    field_match(case_id, f, mr[f], am.get(f)),
                )
            )
        tr = rules["toc_structure_check"]
        at = actual.get("table_of_contents", {})
        checks.append(
            (
                "total_chapters",
                tr["total_chapters"],
                at.get("total_chapters"),
                str(tr["total_chapters"]) == str(at.get("total_chapters")),
            )
        )
        checks.append(
            (
                "total_appendices",
                tr["total_appendices"],
                at.get("total_appendices"),
                str(tr["total_appendices"]) == str(at.get("total_appendices")),
            )
        )
        titles = at.get("chapter_titles", []) or []
        for kw in tr["chapter_titles_must_contain"]:
            checks.append(
                (
                    f'"{kw}"',
                    "포함",
                    "포함" if kw_match(titles, kw, case_id) else "미포함",
                    kw_match(titles, kw, case_id),
                )
            )
        de = rules["toc_depth_check"]["max_depth_observed"]
        checks.append(
            (
                "max_depth",
                f">={de}",
                at.get("max_depth_observed"),
                int(at.get("max_depth_observed", 0)) >= int(de),
            )
        )
    elif case_id == "mcat":
        mr = rules["metadata_exact_match"]
        am = actual.get("metadata", {})
        for f in mr["fields"]:
            checks.append(
                (
                    f"metadata.{f}",
                    mr[f],
                    am.get(f),
                    field_match(case_id, f, mr[f], am.get(f)),
                )
            )
        tr = rules["toc_structure_check"]
        at = actual.get("table_of_contents", {})
        checks.append(
            (
                "bio_chapters",
                tr["biology_chapters"],
                at.get("biology_chapters"),
                str(tr["biology_chapters"]) == str(at.get("biology_chapters")),
            )
        )
        checks.append(
            (
                "biochem_chapters",
                tr["biochemistry_chapters"],
                at.get("biochemistry_chapters"),
                str(tr["biochemistry_chapters"])
                == str(at.get("biochemistry_chapters")),
            )
        )
        bt = at.get("biology_chapter_titles", []) or []
        for kw in ["Cell Structure and Function", "Human Anatomy and Physiology"]:
            checks.append(
                (
                    f'bio "{kw[:25]}"',
                    "포함",
                    "포함" if kw_match(bt, kw, case_id) else "미포함",
                    kw_match(bt, kw, case_id),
                )
            )
        bct = at.get("biochemistry_chapter_titles", []) or []
        checks.append(
            (
                'biochem "Enzymes"',
                "포함",
                "포함" if kw_match(bct, "Enzymes and Kinetics", case_id) else "미포함",
                kw_match(bct, "Enzymes and Kinetics", case_id),
            )
        )
    elif case_id == "inflearn":
        mr = rules["metadata_exact_match"]
        am = actual.get("metadata", {})
        for f in mr["fields"]:
            checks.append(
                (
                    f"metadata.{f}",
                    mr[f],
                    am.get(f),
                    field_match(case_id, f, mr[f], am.get(f)),
                )
            )
        cr = rules["curriculum_structure_check"]
        ac = actual.get("curriculum", {})
        checks.append(
            (
                "total_sections",
                cr["total_sections"],
                ac.get("total_sections"),
                str(cr["total_sections"]) == str(ac.get("total_sections")),
            )
        )
        st = ac.get("section_titles", []) or []
        for kw in cr["section_titles_must_contain"]:
            checks.append(
                (
                    f'sec "{kw[:25]}"',
                    "포함",
                    "포함" if any(flexible_match(kw, str(t)) for t in st) else "미포함",
                    any(flexible_match(kw, str(t)) for t in st),
                )
            )
        kl = ac.get("key_lectures", {}) or {}
        for key in ["3.3", "3.7"]:
            ai = kl.get(key, {}) or {}
            checks.append(
                (f"lec[{key}].title", "", ai.get("title", ""), bool(ai.get("title")))
            )
            checks.append(
                (
                    f"lec[{key}].duration",
                    "",
                    ai.get("duration", ""),
                    bool(ai.get("duration")),
                )
            )
    elif case_id == "realdeal":
        mr = rules["metadata_exact_match"]
        am = actual.get("metadata", {})
        for f in mr["fields"]:
            checks.append(
                (
                    f"metadata.{f}",
                    mr[f],
                    am.get(f),
                    field_match(case_id, f, mr[f], am.get(f)),
                )
            )
        cr = rules["curriculum_structure_check"]
        ac = actual.get("curriculum", {})
        checks.append(
            (
                "total_chapters",
                cr["total_chapters"],
                ac.get("total_chapters"),
                str(cr["total_chapters"]) == str(ac.get("total_chapters")),
            )
        )
        checks.append(
            (
                "has_special",
                cr["has_special_lessons"],
                ac.get("has_special_lessons"),
                str(cr["has_special_lessons"]) == str(ac.get("has_special_lessons")),
            )
        )
        ct = ac.get("chapter_titles", []) or []
        for kw in cr["chapter_titles_must_contain"]:
            checks.append(
                (
                    f'"{kw}"',
                    "포함",
                    "포함" if kw_match(ct, kw, case_id) else "미포함",
                    kw_match(ct, kw, case_id),
                )
            )
        kl = ac.get("key_lectures", {}) or {}
        for key in ["1", "27"]:
            ai = kl.get(key, {}) or {}
            checks.append(
                (f"lec[{key}].title", "", ai.get("title", ""), bool(ai.get("title")))
            )
            checks.append(
                (
                    f"lec[{key}].duration",
                    "",
                    ai.get("duration", ""),
                    bool(ai.get("duration")),
                )
            )
    return checks


CLI_CASES = {
    "cleancode": {
        "args": ["--isbn", "978-0132350884"],
        "fixture": "cleancode-en-book-toc.json",
    },
    "inflearn": {
        "args": [
            "인프런 시스템 디자인 첫걸음: 면접에서 돋보이는 백엔드 아키텍처 설계하기"
        ],
        "fixture": "inflearn-system-kr.json",
    },
    "realdeal": {
        "args": ["리얼딜클래스 제로투원"],
        "fixture": "realdealclass-kr-lecture.json",
    },
}


def flatten_toc_titles(nodes: list[dict], depth: int = 0) -> list[str]:
    """Recursively extract all title strings from hierarchical TOC."""
    titles = []
    for node in nodes or []:
        t = node.get("title", {})
        if isinstance(t, dict):
            for v in [t.get("original", ""), t.get("korean", "")]:
                if v:
                    titles.append(v)
        elif isinstance(t, str) and t:
            titles.append(t)
        titles.extend(flatten_toc_titles(node.get("children", []), depth + 1))
    return titles


def count_depth1(nodes: list[dict]) -> int:
    return len([n for n in (nodes or []) if n.get("depth", 0) == 1])


def compare_cli_case(case_id: str, fixture: dict, cli_output: dict) -> list[tuple]:
    """Compare Go CLI BookInfo JSON output against cli_verification rules."""
    checks = []
    rules = fixture["verification_rules"].get("cli_verification", {})
    if not rules:
        return [("cli_verification", "rules", "missing", False)]

    book = cli_output.get("book", {})
    toc = cli_output.get("table_of_contents", [])
    all_titles = flatten_toc_titles(toc)

    # Author check (keyword match)
    author_kws = rules.get("book_author_keywords", [])
    if not author_kws and rules.get("book_author"):
        author_kws = [rules["book_author"]]
    if author_kws:
        author_val = book.get("author", "")
        author_ok = any(flexible_match(kw, author_val) for kw in author_kws)
        checks.append(("author", str(author_kws[0]), author_val, author_ok))

    # Publisher check
    pub_kws = rules.get("book_publisher_keywords", [])
    if not pub_kws and rules.get("book_publisher"):
        pub_kws = [rules["book_publisher"]]
    if pub_kws:
        pub_val = book.get("publisher", "")
        pub_ok = any(flexible_match(kw, pub_val) for kw in pub_kws)
        checks.append(("publisher", str(pub_kws[0]), pub_val, pub_ok))

    # ISBN check
    if rules.get("book_isbn13"):
        isbn_val = book.get("isbn13", "")
        checks.append(
            (
                "isbn13",
                rules["book_isbn13"],
                isbn_val,
                flexible_match(rules["book_isbn13"], isbn_val),
            )
        )

    # TOC depth-1 count
    if rules.get("toc_min_depth1_count"):
        d1 = count_depth1(toc)
        checks.append(
            (
                "depth1_count",
                f">={rules['toc_min_depth1_count']}",
                d1,
                d1 >= rules["toc_min_depth1_count"],
            )
        )

    # TOC keyword presence
    for kw in rules.get("toc_title_keywords", []):
        found = any(flexible_match(kw, t) for t in all_titles)
        checks.append(
            (f'toc "{kw[:30]}"', "포함", "포함" if found else "미포함", found)
        )

    return checks


def run_cli_comparison() -> int:
    """Run Go CLI binary and compare output against verified fixtures."""
    import subprocess

    root = repo_root()
    binary = root / "bin" / "bookinfo"
    if not binary.exists():
        print("Building binary...", flush=True)
        subprocess.run(["make", "build"], cwd=root, check=True)

    fixture_dir = root / "docs" / "verified_source" / "json"
    output_dir = root / "bookinfo_tdd_plan_results" / "cli_comparison"
    output_dir.mkdir(parents=True, exist_ok=True)

    # Load .env for API key
    env = os.environ.copy()
    env_file = root / ".env"
    if env_file.exists():
        for line in env_file.read_text().splitlines():
            text = line.strip()
            if text.startswith("#") or "=" not in text:
                continue
            k, v = text.split("=", 1)
            env[k.strip()] = v.strip().strip('"').strip("'")
    env["BOOKINFO_HTTP_TIMEOUT_SEC"] = "180"

    results = []
    total_pass = total_checks = 0
    for case_id in ["cleancode", "inflearn", "realdeal"]:
        cfg = CLI_CASES[case_id]
        fixture = json.loads((fixture_dir / cfg["fixture"]).read_text())
        args = [str(binary)] + cfg["args"] + ["--format", "json", "--no-cache"]
        print(f"[cli] {case_id}: {' '.join(args)}", flush=True)

        started = time.time()
        try:
            proc = subprocess.run(
                args, capture_output=True, text=True, timeout=300, env=env
            )
            elapsed = time.time() - started
        except subprocess.TimeoutExpired:
            print(f"  -> TIMEOUT (300s)")
            results.append(
                {
                    "case_id": case_id,
                    "ok": False,
                    "error": "timeout",
                    "checks": [],
                    "passed": 0,
                    "total": 0,
                }
            )
            continue

        if proc.returncode != 0:
            print(f"  -> FAILED (exit {proc.returncode}): {proc.stderr[:200]}")
            results.append(
                {
                    "case_id": case_id,
                    "ok": False,
                    "error": proc.stderr[:300],
                    "checks": [],
                    "passed": 0,
                    "total": 0,
                }
            )
            continue

        # Parse JSON from stdout — CLI may print status lines before JSON
        cli_json = None
        for line in proc.stdout.strip().splitlines():
            line = line.strip()
            if line.startswith("{"):
                try:
                    cli_json = json.loads(line)
                    break
                except json.JSONDecodeError:
                    pass
        # Try the saved file instead
        if not cli_json:
            # Find the most recent output file
            import glob

            pattern = str(root / "*_20*.json")
            files = sorted(glob.glob(pattern), key=os.path.getmtime, reverse=True)
            for f in files:
                try:
                    cli_json = json.loads(Path(f).read_text())
                    break
                except (json.JSONDecodeError, IOError):
                    pass

        if not cli_json:
            print(f"  -> FAILED: no JSON output")
            results.append(
                {
                    "case_id": case_id,
                    "ok": False,
                    "error": "no JSON",
                    "checks": [],
                    "passed": 0,
                    "total": 0,
                }
            )
            continue

        checks = compare_cli_case(case_id, fixture, cli_json)
        passed = sum(1 for *_, ok in checks if ok)
        results.append(
            {
                "case_id": case_id,
                "ok": True,
                "elapsed": elapsed,
                "checks": [(n, str(e), str(a), ok) for n, e, a, ok in checks],
                "passed": passed,
                "total": len(checks),
            }
        )
        total_pass += passed
        total_checks += len(checks)
        print(f"  -> {passed}/{len(checks)} ({elapsed:.1f}s)")

    # Report
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    report_path = output_dir / f"cli_report_{ts}.md"
    lines = [
        f"# Go CLI vs Verified Source 비교 리포트 ({datetime.now().strftime('%Y-%m-%d %H:%M:%S')})",
        "",
    ]
    for r in results:
        lines.append(f"## {r['case_id']}")
        if not r["ok"]:
            lines.append(f"- FAILED: {r.get('error','')[:200]}")
            lines.append("")
            continue
        lines.append(f"- {r['elapsed']:.1f}s")
        lines.append("")
        lines.append("| 항목 | 기준 | 실측 | 결과 |")
        lines.append("|---|---|---|---|")
        for n, e, a, ok in r["checks"]:
            lines.append(f"| {n} | {e} | {a} | {'PASS' if ok else 'FAIL'} |")
        lines.append(f"\n**{r['passed']}/{r['total']} 통과**\n")
    lines.append("## 전체 요약")
    lines.append(f"- 총: **{total_pass}/{total_checks}**")
    if total_checks:
        lines.append(f"- 통과율: **{100*total_pass/total_checks:.1f}%**")
    report_path.write_text("\n".join(lines))
    print(f"\n{'='*50}")
    print(
        f"총: {total_pass}/{total_checks} ({100*total_pass/total_checks:.1f}%)"
        if total_checks
        else "N/A"
    )
    print(f"리포트: {report_path}")
    return 0


def main() -> int:
    root = repo_root()
    api_key = load_api_key(root)
    if not api_key:
        print("ERROR: DASHSCOPE_API_KEY not found", file=sys.stderr)
        return 2
    fixture_dir = root / "docs" / "verified_source" / "json"
    output_dir = root / "bookinfo_tdd_plan_results" / "dashscope_comparison"
    output_dir.mkdir(parents=True, exist_ok=True)
    model = os.environ.get("DASHSCOPE_MODEL", DEFAULT_MODEL)
    timeout = int(os.environ.get("DASHSCOPE_TIMEOUT", "300"))
    cases = CASE_ORDER[:]
    results = []
    total_pass = total_checks = 0

    for case_id in cases:
        fixture = json.loads((fixture_dir / CASE_TO_FIXTURE[case_id]).read_text())
        print(f"[dashscope] {case_id} ...", flush=True)
        resp = call_dashscope(api_key, model, case_id, timeout)
        result = {"case_id": case_id, **resp}
        if resp["ok"] and resp["parsed"]:
            checks = compare_case(case_id, fixture, resp["parsed"])
            passed = sum(1 for *_, ok in checks if ok)
            result["checks"] = [(n, str(e), str(a), ok) for n, e, a, ok in checks]
            result["passed"] = passed
            result["total"] = len(checks)
            total_pass += passed
            total_checks += len(checks)
            print(
                f"  -> {passed}/{len(checks)} ({resp['searches']} searches, {resp['extracts']} extracts, {resp['elapsed']:.1f}s)"
            )
        else:
            result["checks"] = []
            result["passed"] = 0
            result["total"] = 0
            print(f"  -> FAILED: {resp.get('error', 'no parsed JSON')[:100]}")
        results.append(result)

    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    report_path = output_dir / f"dashscope_report_{ts}.md"
    raw_path = output_dir / f"dashscope_raw_{ts}.json"
    raw_path.write_text(
        json.dumps(
            {"model": model, "results": results},
            ensure_ascii=False,
            indent=2,
            default=str,
        )
    )
    lines = [
        f"# DashScope Responses API 비교 리포트 ({datetime.now().strftime('%Y-%m-%d %H:%M:%S')})",
        "",
        f"- 모델: `{model}`",
        "- API: DashScope Responses (web_search + web_extractor)",
        f"- timeout: {timeout}s",
        "",
    ]
    for r in results:
        cid = r["case_id"]
        lines.append(f"## {cid}")
        if not r["ok"]:
            lines.append(f"- FAILED: {r.get('error','')[:200]}")
            lines.append("")
            continue
        lines.append(
            f"- {r['elapsed']:.1f}s | searches:{r['searches']} extracts:{r['extracts']} | tokens: in={r['usage'].get('input_tokens',0)} out={r['usage'].get('output_tokens',0)}"
        )
        lines.append("")
        lines.append("| 항목 | 기준 | 실측 | 결과 |")
        lines.append("|---|---|---|---|")
        for n, e, a, ok in r["checks"]:
            lines.append(f"| {n} | {e} | {a} | {'✅' if ok else '❌'} |")
        lines.append(f"\n**{r['passed']}/{r['total']} 통과**\n")
    lines.append("## 전체 요약")
    lines.append(f"- 총: **{total_pass}/{total_checks}**")
    if total_checks:
        lines.append(f"- 통과율: **{100*total_pass/total_checks:.1f}%**")
    report_path.write_text("\n".join(lines))
    print(f"\n{'='*50}")
    print(
        f"총: {total_pass}/{total_checks} ({100*total_pass/total_checks:.1f}%)"
        if total_checks
        else "N/A"
    )
    print(f"리포트: {report_path}")
    return 0


if __name__ == "__main__":
    if "--cli" in sys.argv:
        raise SystemExit(run_cli_comparison())
    raise SystemExit(main())
