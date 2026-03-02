#!/usr/bin/env python3
"""
Verified Source vs OpenRouter 실요청 비교 스크립트

기본값:
- model: qwen/qwen3.5-flash-02-23
- search_mode: auto
- plugins: [{id:web, engine:native, max_results:5}]
"""

from __future__ import annotations

import argparse
import json
import os
import re
import signal
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

DEFAULT_MODEL = "qwen/qwen3.5-flash-02-23"
CASE_ORDER = ["cleancode", "mcat", "inflearn", "realdeal"]
CASE_TO_FIXTURE = {
    "cleancode": "cleancode-en-book-toc.json",
    "mcat": "mcat-en-book-toc.json",
    "inflearn": "inflearn-system-kr.json",
    "realdeal": "realdealclass-kr-lecture.json",
}
PROMPT_MODES = ["strict-json", "soft-json", "two-step-json"]
CASE_TO_QUERY = {
    "cleancode": 'Clean Code: A Handbook of Agile Software Craftsmanship by Robert C. Martin (English, original edition)',
    "mcat": "MCAT Biological & Biochemical Foundations - Kaplan Biology Review + Biochemistry Review (English, latest edition)",
    "inflearn": '"시스템 디자인 첫걸음" 인프런 강의 (강사: mindlantern)',
    "realdeal": '"리얼딜 클라쓰 ZERO TO ONE" 영어 문법 온라인 강의',
}
CASE_TO_SCHEMA = {
    "cleancode": (
        '{"metadata":{"author":"","publisher":"","isbn13":""},'
        '"table_of_contents":{"total_chapters":0,"total_appendices":0,'
        '"chapter_titles":[],"max_depth_observed":0}}'
    ),
    "mcat": (
        '{"metadata":{"publisher":"","isbn13_biology":"","isbn13_biochemistry":""},'
        '"table_of_contents":{"biology_chapters":0,"biochemistry_chapters":0,'
        '"biology_chapter_titles":[],"biochemistry_chapter_titles":[],"max_depth_observed":0}}'
    ),
    "inflearn": (
        '{"metadata":{"instructor":"","platform":"","rating":0,"total_lectures":0},'
        '"curriculum":{"total_sections":0,"section_titles":[],"section_lecture_counts":[],'
        'key_lectures":{"3.3":{"title":"","duration":""},"3.7":{"title":"","duration":""}}}}'
    ),
    "realdeal": (
        '{"metadata":{"platform":"","difficulty":"","duration":""},'
        '"curriculum":{"total_chapters":0,"has_special_lessons":false,"special_lesson_count":0,'
        '"chapter_titles":[],"key_lectures":{"1":{"title":"","duration":""},"27":{"title":"","duration":""}}}}'
    ),
}

OPENROUTER_EXTRACTION_SYSTEM_PROMPT = """You are a precise metadata and table-of-contents extraction engine.

CRITICAL RULES:
1. Use web evidence and return only high-confidence facts.
2. Follow the user-requested output schema exactly.
3. Preserve titles in the ORIGINAL language of the source resource.
   - English source -> English titles, Korean source -> Korean titles.
   - Do not translate unless explicitly requested.
4. Prefer canonical/original edition data over translated variants unless query language says otherwise.
5. If data is uncertain, leave placeholders empty instead of inventing values.
6. If JSON is requested, output valid JSON only (no markdown, no explanation, no code fences).
7. Do NOT embed source citations, URLs, or references inside JSON values. Keep values clean data only.
8. If the exact edition is uncertain but the book/lecture is clearly identified, still extract the TOC from available evidence rather than leaving it empty."""

OPENROUTER_EVIDENCE_SYSTEM_PROMPT = """You are a careful web research assistant for metadata and curriculum extraction.

Collect high-confidence evidence, keep the ORIGINAL language of titles, and clearly separate confirmed facts from uncertainty.
Do not invent unsupported claims. Prefer canonical/original edition data where applicable."""

OPENROUTER_MAPPER_SYSTEM_PROMPT = """You are a strict JSON mapper.

Map ONLY from the supplied evidence into the target schema.
- Return one valid JSON object only.
- No markdown, no explanation, no extra keys.
- Keep titles in the original language found in evidence.
- If evidence is missing, use empty placeholders ("", 0, [], false)."""


@dataclass
class CaseExecution:
    case_id: str
    fixture: str
    ok: bool
    elapsed_sec: float
    error: str | None
    checks: list[tuple[str, Any, Any, bool]]
    passed: int
    total: int
    web_search_requests: int | str
    mode_used: str
    actual: dict[str, Any] | None
    request_payload: dict[str, Any]
    response_usage: dict[str, Any] | None


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verified Source vs OpenRouter 실요청 비교 리포트 생성"
    )
    parser.add_argument("--model", default=DEFAULT_MODEL, help="OpenRouter model id")
    parser.add_argument(
        "--search-mode",
        default="auto",
        choices=["plugins", "online", "auto"],
        help="검색 사용 방식",
    )
    parser.add_argument(
        "--prompt-mode",
        default="strict-json",
        choices=PROMPT_MODES,
        help="프롬프트 방식: strict-json|soft-json|two-step-json",
    )
    parser.add_argument(
        "--engine",
        default="native",
        choices=["native", "exa"],
        help="web plugin engine",
    )
    parser.add_argument("--max-results", type=int, default=5, help="web plugin max_results")
    parser.add_argument(
        "--cases",
        default="all",
        help="실행 케이스(콤마구분). all 또는 cleancode,mcat,inflearn,realdeal",
    )
    parser.add_argument(
        "--output-dir",
        default="bookinfo_tdd_plan_results",
        help="리포트/원본 출력 디렉터리",
    )
    parser.add_argument("--timeout", type=int, default=120, help="요청 타임아웃(초)")
    parser.add_argument(
        "--max-retries",
        type=int,
        default=2,
        help="요청 실패 시 재시도 횟수(429/timeout 중심)",
    )
    parser.add_argument(
        "--retry-delay",
        type=float,
        default=2.0,
        help="재시도 기본 대기(초, 지수백오프 적용)",
    )
    parser.add_argument("--api-key-env", default="OPENROUTER_API_KEY", help="API 키 환경변수명")
    return parser.parse_args()


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def load_api_key(root: Path, env_name: str) -> str:
    env_key = os.environ.get(env_name, "").strip()
    if env_key:
        return env_key

    env_file = root / ".env"
    if env_file.exists():
        for line in env_file.read_text(encoding="utf-8").splitlines():
            text = line.strip()
            if not text or text.startswith("#") or "=" not in text:
                continue
            key, value = text.split("=", 1)
            if key.strip() == env_name:
                return value.strip().strip('"').strip("'")
    return ""


def resolve_cases(raw: str) -> list[str]:
    if raw.strip().lower() == "all":
        return CASE_ORDER[:]
    cases = [item.strip().lower() for item in raw.split(",") if item.strip()]
    unknown = [c for c in cases if c not in CASE_TO_FIXTURE]
    if unknown:
        raise ValueError(f"알 수 없는 cases: {', '.join(unknown)}")
    return cases


def extract_json(text: str) -> dict[str, Any]:
    body = text.strip()
    code_fence = re.search(r"```(?:json)?\s*(\{[\s\S]*\})\s*```", body)
    if code_fence:
        body = code_fence.group(1)

    if not body.startswith("{"):
        start = body.find("{")
        end = body.rfind("}")
        if start >= 0 and end > start:
            body = body[start : end + 1]

    parsed = json.loads(body)
    if not isinstance(parsed, dict):
        raise ValueError("응답 JSON 루트가 object가 아님")
    return parsed


def get_case_query(case_id: str) -> str:
    if case_id not in CASE_TO_QUERY:
        raise ValueError(case_id)
    return CASE_TO_QUERY[case_id]


def get_case_schema(case_id: str) -> str:
    if case_id not in CASE_TO_SCHEMA:
        raise ValueError(case_id)
    return CASE_TO_SCHEMA[case_id]


def make_prompt(case_id: str, prompt_mode: str) -> str:
    query = get_case_query(case_id)
    schema = get_case_schema(case_id)
    if prompt_mode == "strict-json":
        return (
            f'웹 검색을 사용해 쿼리 {query} 를 확인하고 아래 JSON 스키마로만 답하세요.\n'
            + schema
            + "\n설명문/마크다운 없이 JSON만 출력."
        )
    if prompt_mode == "soft-json":
        return (
            f'웹 검색으로 쿼리 {query} 정보를 최대한 수집하세요.\n'
            "가능하면 아래 스키마에 맞춰 JSON으로 답하되, 확신 없는 값은 빈 값으로 두세요.\n"
            + schema
            + "\nJSON 우선이며, 불가하면 마지막에 JSON 블록을 반드시 포함하세요."
        )
    if prompt_mode == "two-step-json":
        return (
            f'웹 검색으로 쿼리 {query} 정보를 먼저 자유형으로 정리하세요.\n'
            "아래 형식을 따르세요:\n"
            "1) 핵심 사실(메타데이터)\n2) 목차/커리큘럼 핵심 항목\n3) 숫자값(챕터수/강의수/평점/길이)\n"
            "4) 불확실한 항목\n"
            "5) 참고 URL 목록(있으면)\n"
            "지금 단계에서는 JSON으로 강제하지 않습니다."
        )
    raise ValueError(prompt_mode)


def request_once(
    api_key: str, payload: dict[str, Any], timeout_sec: int, expect_json: bool
) -> dict[str, Any]:
    class _RequestTimeout(Exception):
        pass

    def _handle_alarm(_signum: int, _frame: Any) -> None:
        raise _RequestTimeout()

    req = Request(
        "https://openrouter.ai/api/v1/chat/completions",
        data=json.dumps(payload).encode("utf-8"),
        method="POST",
    )
    req.add_header("Authorization", f"Bearer {api_key}")
    req.add_header("Content-Type", "application/json")

    started = time.time()
    prev_handler = signal.signal(signal.SIGALRM, _handle_alarm)
    signal.alarm(max(1, int(timeout_sec)))
    try:
        with urlopen(req, timeout=timeout_sec) as resp:
            raw_text = resp.read().decode("utf-8")
            elapsed = time.time() - started
            outer = json.loads(raw_text)
            if isinstance(outer, dict) and isinstance(outer.get("error"), dict):
                err_obj = outer["error"]
                code = err_obj.get("code", "unknown")
                msg = err_obj.get("message", "")
                return {
                    "ok": False,
                    "elapsed_sec": elapsed,
                    "error": f"UpstreamError {code}: {msg}",
                    "error_body": raw_text,
                    "request_payload": payload,
                }
            choices = outer.get("choices") if isinstance(outer, dict) else None
            if not isinstance(choices, list) or len(choices) == 0:
                return {
                    "ok": False,
                    "elapsed_sec": elapsed,
                    "error": "InvalidResponse missing choices",
                    "error_body": raw_text,
                    "request_payload": payload,
                }
            message = choices[0].get("message", {}) if isinstance(choices[0], dict) else {}
            content = message.get("content")
            if not isinstance(content, str):
                return {
                    "ok": False,
                    "elapsed_sec": elapsed,
                    "error": "InvalidResponse missing message.content",
                    "error_body": raw_text,
                    "request_payload": payload,
                }
            parsed = extract_json(content) if expect_json else None
            usage = outer.get("usage")
            web_requests = (
                usage.get("server_tool_use", {}).get("web_search_requests")
                if isinstance(usage, dict)
                else None
            )
            return {
                "ok": True,
                "elapsed_sec": elapsed,
                "parsed": parsed,
                "content": content,
                "outer_response": outer,
                "usage": usage,
                "web_search_requests": web_requests if web_requests is not None else "N/A",
                "request_payload": payload,
            }
    except _RequestTimeout:
        return {
            "ok": False,
            "elapsed_sec": time.time() - started,
            "error": f"TimeoutError request exceeded {timeout_sec}s",
            "request_payload": payload,
        }
    except HTTPError as error:
        body = error.read().decode("utf-8", errors="ignore")
        return {
            "ok": False,
            "elapsed_sec": time.time() - started,
            "error": f"HTTPError {error.code}",
            "error_body": body,
            "request_payload": payload,
        }
    except URLError as error:
        return {
            "ok": False,
            "elapsed_sec": time.time() - started,
            "error": f"URLError {error}",
            "request_payload": payload,
        }
    except Exception as error:  # noqa: BLE001
        return {
            "ok": False,
            "elapsed_sec": time.time() - started,
            "error": f"Exception {error}",
            "request_payload": payload,
        }
    finally:
        signal.alarm(0)
        signal.signal(signal.SIGALRM, prev_handler)


def request_with_retries(
    api_key: str,
    payload: dict[str, Any],
    timeout_sec: int,
    expect_json: bool,
    max_retries: int,
    retry_delay: float,
) -> dict[str, Any]:
    last: dict[str, Any] = {}
    for attempt in range(max(0, max_retries) + 1):
        last = request_once(api_key, payload, timeout_sec, expect_json=expect_json)
        if last.get("ok"):
            if attempt > 0:
                last["retry_attempts"] = attempt
            return last

        err = str(last.get("error", ""))
        retryable = err.startswith("HTTPError 429") or err.startswith("TimeoutError")
        if not retryable or attempt >= max_retries:
            if attempt > 0:
                last["retry_attempts"] = attempt
            return last

        sleep_sec = max(0.0, retry_delay) * (2**attempt)
        if sleep_sec > 0:
            time.sleep(sleep_sec)

    return last


def call_openrouter_search(
    api_key: str,
    model: str,
    search_mode: str,
    engine: str,
    max_results: int,
    messages: list[dict[str, str]],
    timeout_sec: int,
    expect_json: bool,
    max_retries: int,
    retry_delay: float,
) -> dict[str, Any]:
    plugins_payload = {
        "model": model,
        "temperature": 0,
        "max_tokens": 4000,
        "plugins": [{"id": "web", "engine": engine, "max_results": max_results}],
        "messages": messages,
    }
    online_model = model if model.endswith(":online") else f"{model}:online"
    online_payload = {
        "model": online_model,
        "temperature": 0,
        "max_tokens": 4000,
        "messages": messages,
    }

    if search_mode == "plugins":
        result = request_with_retries(
            api_key,
            plugins_payload,
            timeout_sec,
            expect_json=expect_json,
            max_retries=max_retries,
            retry_delay=retry_delay,
        )
        result["mode_used"] = "plugins"
        return result
    if search_mode == "online":
        result = request_with_retries(
            api_key,
            online_payload,
            timeout_sec,
            expect_json=expect_json,
            max_retries=max_retries,
            retry_delay=retry_delay,
        )
        result["mode_used"] = "online"
        return result

    first = request_with_retries(
        api_key,
        plugins_payload,
        timeout_sec,
        expect_json=expect_json,
        max_retries=max_retries,
        retry_delay=retry_delay,
    )
    first["mode_used"] = "plugins"
    if first.get("ok"):
        return first

    error_body = str(first.get("error_body", ""))
    error_text = str(first.get("error", ""))
    if "No endpoints found that support native web search" in error_body or (
        "HTTPError 404" in error_text and "native web search" in error_body
    ):
        second = request_with_retries(
            api_key,
            online_payload,
            timeout_sec,
            expect_json=expect_json,
            max_retries=max_retries,
            retry_delay=retry_delay,
        )
        second["mode_used"] = "online(fallback)"
        second["fallback_from"] = "plugins"
        second["fallback_reason"] = "native web search endpoint unavailable"
        return second
    return first


def map_evidence_to_json(
    api_key: str,
    model: str,
    case_id: str,
    evidence_text: str,
    timeout_sec: int,
    max_retries: int,
    retry_delay: float,
) -> dict[str, Any]:
    schema = get_case_schema(case_id)
    messages = [
        {
            "role": "system",
            "content": OPENROUTER_MAPPER_SYSTEM_PROMPT,
        },
        {
            "role": "user",
            "content": (
                "아래 evidence를 기반으로 스키마에 맞는 JSON만 출력하세요.\n"
                "모르는 값은 빈 문자열/0/[]/false를 사용하세요.\n"
                f"스키마:\n{schema}\n\n"
                f"evidence:\n{evidence_text}"
            ),
        },
    ]
    payload = {
        "model": model,
        "temperature": 0,
        "max_tokens": 4000,
        "messages": messages,
    }
    return request_with_retries(
        api_key,
        payload,
        timeout_sec,
        expect_json=True,
        max_retries=max_retries,
        retry_delay=retry_delay,
    )


def call_openrouter(
    api_key: str,
    model: str,
    search_mode: str,
    engine: str,
    max_results: int,
    case_id: str,
    prompt: str,
    prompt_mode: str,
    timeout_sec: int,
    max_retries: int,
    retry_delay: float,
) -> dict[str, Any]:
    if prompt_mode in ("strict-json", "soft-json"):
        messages = [
            {
                "role": "system",
                "content": OPENROUTER_EXTRACTION_SYSTEM_PROMPT,
            },
            {"role": "user", "content": prompt},
        ]
        result = call_openrouter_search(
            api_key=api_key,
            model=model,
            search_mode=search_mode,
            engine=engine,
            max_results=max_results,
            messages=messages,
            timeout_sec=timeout_sec,
            expect_json=True,
            max_retries=max_retries,
            retry_delay=retry_delay,
        )
        result["prompt_mode"] = prompt_mode
        return result

    raw_messages = [
        {
            "role": "system",
            "content": OPENROUTER_EVIDENCE_SYSTEM_PROMPT,
        },
        {"role": "user", "content": prompt},
    ]
    stage1 = call_openrouter_search(
        api_key=api_key,
        model=model,
        search_mode=search_mode,
        engine=engine,
        max_results=max_results,
        messages=raw_messages,
        timeout_sec=timeout_sec,
        expect_json=False,
        max_retries=max_retries,
        retry_delay=retry_delay,
    )
    if not stage1.get("ok"):
        stage1["prompt_mode"] = prompt_mode
        return stage1

    mapper = map_evidence_to_json(
        api_key=api_key,
        model=model,
        case_id=case_id,
        evidence_text=str(stage1.get("content", "")),
        timeout_sec=timeout_sec,
        max_retries=max_retries,
        retry_delay=retry_delay,
    )
    mapper["elapsed_sec"] = float(stage1.get("elapsed_sec", 0.0)) + float(
        mapper.get("elapsed_sec", 0.0)
    )
    mapper["web_search_requests"] = stage1.get("web_search_requests", "N/A")
    mapper["mode_used"] = str(stage1.get("mode_used", "unknown")) + " + mapper"
    mapper["request_payload"] = {
        "stage1_search_payload": stage1.get("request_payload", {}),
        "stage2_mapper_payload": mapper.get("request_payload", {}),
    }
    mapper["prompt_mode"] = prompt_mode
    mapper["stage1_raw_content"] = stage1.get("content")
    return mapper


def contains_keyword(text: str, keyword: str) -> bool:
    return keyword.lower() in text.lower()


def normalize_for_compare(text: Any) -> str:
    if text is None:
        return ""
    normalized = str(text).strip().lower()
    normalized = re.sub(r"\s+", " ", normalized)
    return normalized


def _normalize_compact(text: Any) -> str:
    normalized = normalize_for_compare(text)
    return re.sub(r"[\s\W_]+", "", normalized, flags=re.UNICODE)


def _variants_for_flexible_compare(text: Any) -> list[str]:
    base = str(text or "")
    variants: list[str] = [base]
    inner = re.findall(r"\(([^)]+)\)", base)
    variants.extend(inner)
    without_paren = re.sub(r"\([^)]*\)", " ", base).strip()
    if without_paren:
        variants.append(without_paren)
    return variants


def flexible_match(expected: str, actual: str) -> bool:
    exp_norm = normalize_for_compare(expected)
    act_norm = normalize_for_compare(actual)
    if not exp_norm or not act_norm:
        return False

    if exp_norm == act_norm:
        return True
    if exp_norm in act_norm or act_norm in exp_norm:
        return True

    exp_compact = _normalize_compact(expected)
    act_compact = _normalize_compact(actual)
    if exp_compact and act_compact and (exp_compact == act_compact or exp_compact in act_compact):
        return True

    exp_variants = _variants_for_flexible_compare(expected)
    act_variants = _variants_for_flexible_compare(actual)
    for exp_variant in exp_variants:
        exp_v_norm = normalize_for_compare(exp_variant)
        if not exp_v_norm:
            continue
        for act_variant in act_variants:
            act_v_norm = normalize_for_compare(act_variant)
            if not act_v_norm:
                continue
            if exp_v_norm == act_v_norm or exp_v_norm in act_v_norm or act_v_norm in exp_v_norm:
                return True
    return False


TITLE_ALIASES: dict[str, dict[str, list[str]]] = {
    "cleancode": {
        "Meaningful Names": ["의미있는이름", "의미 있는 이름", "의미있는 이름", "Meaningful Name"],
        "Functions": ["함수"],
        "Comments": ["주석"],
        "Objects and Data Structures": ["객체와자료구조", "객체와 자료구조", "객체와 자료 구조"],
        "Classes": ["클래스"],
        "Exceptions": ["오류처리", "오류 처리", "예외", "Error Handling"],
        "Boundaries": ["경계"],
    },
    "mcat": {
        "Cell Structure and Function": ["The cell", "The Cell", "Cell Biology", "Cell structure"],
        "Human Anatomy and Physiology": [
            "The nervous system", "The endocrine system", "The respiratory system",
            "The cardiovascular system", "The immune system", "The digestive system",
            "The musculoskeletal system", "Homeostasis",
        ],
        "Enzymes and Kinetics": ["Enzymes", "Enzyme kinetics", "Bioenergetics"],
    },
}


def contains_keyword_flexible(titles: list[str], keyword: str, case_id: str) -> bool:
    candidates = [keyword]
    candidates.extend(TITLE_ALIASES.get(case_id, {}).get(keyword, []))

    for title in titles:
        title_text = str(title)
        for candidate in candidates:
            if contains_keyword(title_text, candidate) or flexible_match(candidate, title_text):
                return True
    return False


def compare_case(case_id: str, fixture_obj: dict[str, Any], actual: dict[str, Any]) -> tuple[list[tuple[str, Any, Any, bool]], int, int]:
    checks: list[tuple[str, Any, Any, bool]] = []
    rules = fixture_obj["verification_rules"]

    if case_id == "cleancode":
        expected = rules["metadata_exact_match"]
        actual_meta = actual.get("metadata", {})
        for field in expected["fields"]:
            exp = expected[field]
            act = actual_meta.get(field)
            checks.append((f"metadata.{field}", exp, act, flexible_match(str(exp), str(act))))

        toc_expected = rules["toc_structure_check"]
        actual_toc = actual.get("table_of_contents", {})
        checks.append(("table_of_contents.total_chapters", toc_expected["total_chapters"], actual_toc.get("total_chapters"), str(toc_expected["total_chapters"]) == str(actual_toc.get("total_chapters"))))
        checks.append(("table_of_contents.total_appendices", toc_expected["total_appendices"], actual_toc.get("total_appendices"), str(toc_expected["total_appendices"]) == str(actual_toc.get("total_appendices"))))
        chapter_titles = actual_toc.get("chapter_titles", []) or []
        for keyword in toc_expected["chapter_titles_must_contain"]:
            ok = contains_keyword_flexible(chapter_titles, keyword, case_id)
            checks.append((f'chapter_titles contains "{keyword}"', "포함", "포함" if ok else "미포함", ok))
        depth_expected = rules["toc_depth_check"]["max_depth_observed"]
        depth_actual = actual_toc.get("max_depth_observed")
        checks.append(("table_of_contents.max_depth_observed", depth_expected, depth_actual, str(depth_expected) == str(depth_actual)))

    elif case_id == "mcat":
        expected = rules["metadata_exact_match"]
        actual_meta = actual.get("metadata", {})
        for field in expected["fields"]:
            exp = expected[field]
            act = actual_meta.get(field)
            checks.append((f"metadata.{field}", exp, act, flexible_match(str(exp), str(act))))

        toc_expected = rules["toc_structure_check"]
        actual_toc = actual.get("table_of_contents", {})
        checks.append(("table_of_contents.biology_chapters", toc_expected["biology_chapters"], actual_toc.get("biology_chapters"), str(toc_expected["biology_chapters"]) == str(actual_toc.get("biology_chapters"))))
        checks.append(("table_of_contents.biochemistry_chapters", toc_expected["biochemistry_chapters"], actual_toc.get("biochemistry_chapters"), str(toc_expected["biochemistry_chapters"]) == str(actual_toc.get("biochemistry_chapters"))))
        biology_titles = actual_toc.get("biology_chapter_titles", []) or []
        for keyword in ["Cell Structure and Function", "Human Anatomy and Physiology"]:
            ok = contains_keyword_flexible(biology_titles, keyword, case_id)
            checks.append((f'biology_chapter_titles contains "{keyword}"', "포함", "포함" if ok else "미포함", ok))
        biochem_titles = actual_toc.get("biochemistry_chapter_titles", []) or []
        target = "Enzymes and Kinetics"
        ok = contains_keyword_flexible(biochem_titles, target, case_id)
        checks.append((f'biochemistry_chapter_titles contains "{target}"', "포함", "포함" if ok else "미포함", ok))
        depth_expected = rules["toc_depth_check"]["max_depth_observed"]
        depth_actual = actual_toc.get("max_depth_observed")
        checks.append(("table_of_contents.max_depth_observed", depth_expected, depth_actual, str(depth_expected) == str(depth_actual)))

    elif case_id == "inflearn":
        expected = rules["metadata_exact_match"]
        actual_meta = actual.get("metadata", {})
        INFLEARN_FIELD_ALIASES = {
            "instructor": ["mindlantern", "성장랜턴", "성장 랜턴"],
        }
        for field in expected["fields"]:
            exp = expected[field]
            act = actual_meta.get(field)
            field_aliases = INFLEARN_FIELD_ALIASES.get(field, [])
            if field_aliases and act is not None:
                ok = flexible_match(str(exp), str(act)) or any(
                    flexible_match(alias, str(act)) for alias in field_aliases
                )
            else:
                ok = flexible_match(str(exp), str(act))
            checks.append((f"metadata.{field}", exp, act, ok))

        cur_expected = rules["curriculum_structure_check"]
        actual_cur = actual.get("curriculum", {})
        checks.append(("curriculum.total_sections", cur_expected["total_sections"], actual_cur.get("total_sections"), str(cur_expected["total_sections"]) == str(actual_cur.get("total_sections"))))
        checks.append(("curriculum.section_lecture_counts", cur_expected["section_lecture_counts"], actual_cur.get("section_lecture_counts"), str(cur_expected["section_lecture_counts"]) == str(actual_cur.get("section_lecture_counts"))))
        section_titles = actual_cur.get("section_titles", []) or []
        for keyword in cur_expected["section_titles_must_contain"]:
            ok = any(contains_keyword(str(title), keyword) for title in section_titles)
            checks.append((f'section_titles contains "{keyword}"', "포함", "포함" if ok else "미포함", ok))

        key_lectures = actual_cur.get("key_lectures", {}) or {}
        expected_keys = {
            "3.3": {"title": "API Gateway & Load Balancer & Service Discovery", "duration": "16:01"},
            "3.7": {"title": "Message Queue & Event Broker", "duration": "32:12"},
        }
        for key, data in expected_keys.items():
            actual_item = key_lectures.get(key, {}) or {}
            checks.append((f"key_lectures[{key}].title", data["title"], actual_item.get("title"), str(data["title"]) == str(actual_item.get("title"))))
            checks.append((f"key_lectures[{key}].duration", data["duration"], actual_item.get("duration"), str(data["duration"]) == str(actual_item.get("duration"))))

    elif case_id == "realdeal":
        expected = rules["metadata_exact_match"]
        actual_meta = actual.get("metadata", {})
        REALDEAL_FIELD_ALIASES = {
            "platform": ["RealDealClass", "리얼딜 클라쓰", "리얼딜클라쓰", "Real Deal Class"],
        }
        for field in expected["fields"]:
            exp = expected[field]
            act = actual_meta.get(field)
            field_aliases = REALDEAL_FIELD_ALIASES.get(field, [])
            if field_aliases and act is not None:
                ok = flexible_match(str(exp), str(act)) or any(
                    flexible_match(alias, str(act)) for alias in field_aliases
                )
            else:
                ok = flexible_match(str(exp), str(act))
            checks.append((f"metadata.{field}", exp, act, ok))

        cur_expected = rules["curriculum_structure_check"]
        actual_cur = actual.get("curriculum", {})
        checks.append(("curriculum.total_chapters", cur_expected["total_chapters"], actual_cur.get("total_chapters"), str(cur_expected["total_chapters"]) == str(actual_cur.get("total_chapters"))))
        checks.append(("curriculum.has_special_lessons", cur_expected["has_special_lessons"], actual_cur.get("has_special_lessons"), str(cur_expected["has_special_lessons"]) == str(actual_cur.get("has_special_lessons"))))
        checks.append(("curriculum.special_lesson_count", cur_expected["special_lesson_count"], actual_cur.get("special_lesson_count"), str(cur_expected["special_lesson_count"]) == str(actual_cur.get("special_lesson_count"))))
        chapter_titles = actual_cur.get("chapter_titles", []) or []
        for keyword in cur_expected["chapter_titles_must_contain"]:
            ok = contains_keyword_flexible(chapter_titles, keyword, case_id)
            checks.append((f'chapter_titles contains "{keyword}"', "포함", "포함" if ok else "미포함", ok))

        key_lectures = actual_cur.get("key_lectures", {}) or {}
        expected_keys = {"1": {"title": "문법의 개념", "duration": "37:23"}, "27": {"title": "to-V 1", "duration": "42:20"}}
        for key, data in expected_keys.items():
            actual_item = key_lectures.get(key, {}) or {}
            checks.append((f"key_lectures[{key}].title", data["title"], actual_item.get("title"), str(data["title"]) == str(actual_item.get("title"))))
            checks.append((f"key_lectures[{key}].duration", data["duration"], actual_item.get("duration"), str(data["duration"]) == str(actual_item.get("duration"))))

    passed = sum(1 for _, _, _, ok in checks if ok)
    return checks, passed, len(checks)


def write_outputs(
    root: Path,
    output_dir: Path,
    model: str,
    search_mode: str,
    prompt_mode: str,
    engine: str,
    max_results: int,
    cases: list[CaseExecution],
    raw_data: dict[str, Any],
) -> tuple[Path, Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    raw_path = output_dir / f"live_request_raw_{ts}.json"
    report_path = output_dir / f"live_request_comparison_report_{ts}.md"

    raw_path.write_text(json.dumps(raw_data, ensure_ascii=False, indent=2), encoding="utf-8")

    now_kst = datetime.now(timezone.utc).astimezone().strftime("%Y-%m-%d %H:%M:%S %Z")
    lines: list[str] = []
    lines.append(f"# Verified Source vs 실요청 실측 비교 리포트 ({now_kst})")
    lines.append("")
    lines.append(f"- 모델: `{model}`")
    lines.append(f"- search_mode: `{search_mode}`")
    lines.append(f"- prompt_mode: `{prompt_mode}`")
    lines.append(f"- timeout: `{raw_data.get('meta', {}).get('timeout_sec', 'N/A')}s`")
    lines.append(f"- retry: `max_retries={raw_data.get('meta', {}).get('max_retries', 'N/A')}, base_delay={raw_data.get('meta', {}).get('retry_delay_sec', 'N/A')}s`")
    lines.append("- API: OpenRouter 실요청")
    lines.append("- plugins: " + json.dumps([{"id": "web", "engine": engine, "max_results": max_results}], ensure_ascii=False))
    lines.append("- 비교 기준: `docs/verified_source/json/*`의 `verification_rules`")
    lines.append("- 주의: LLM 응답 비결정성으로 exact 비교는 매 실행 결과가 달라질 수 있음")
    lines.append("")

    total_pass = 0
    total_checks = 0

    lines.append("## 도구 사용 통계 (web_search_requests)")
    lines.append("")
    lines.append("| 케이스 | mode_used | web_search_requests | 비고 |")
    lines.append("|---|---|---:|---|")
    for case in cases:
        note = "요청 성공" if case.ok else f"실패 ({case.error})"
        lines.append(f"| {case.case_id} | {case.mode_used} | {case.web_search_requests} | {note} |")
    lines.append("")

    for case in cases:
        lines.append(f"## 케이스: {case.case_id} (기준: {case.fixture})")
        lines.append(f"- 요청 성공: {'예' if case.ok else '아니오'}")
        lines.append(f"- 응답 시간: {case.elapsed_sec:.2f}s")
        if not case.ok:
            lines.append(f"- 오류: `{case.error}`")
            lines.append("")
            continue

        lines.append("")
        lines.append("| 항목 | 기준값 | 실측값 | 결과 |")
        lines.append("|---|---|---|---|")
        for name, expected, actual, ok in case.checks:
            name_s = str(name).replace("|", "\\|")
            exp_s = str(expected).replace("|", "\\|")
            act_s = str(actual).replace("|", "\\|")
            lines.append(f"| {name_s} | {exp_s} | {act_s} | {'✅' if ok else '❌'} |")

        lines.append("")
        lines.append(f"- 소계: **{case.passed}/{case.total} 통과**")
        lines.append("")

        total_pass += case.passed
        total_checks += case.total

    lines.append("## 전체 요약")
    lines.append(f"- 총 통과: **{total_pass}/{total_checks}**")
    if total_checks > 0:
        lines.append(f"- 통과율: **{(100 * total_pass / total_checks):.1f}%**")
    else:
        lines.append("- 통과율: N/A")
    lines.append("")
    lines.append("## 산출물")
    lines.append(f"- 상세 리포트: `{report_path}`")
    lines.append(f"- 원본 응답(raw): `{raw_path}`")
    lines.append(f"- 실행 기준 root: `{root}`")

    report_path.write_text("\n".join(lines), encoding="utf-8")
    return report_path, raw_path


def main() -> int:
    args = parse_args()
    root = repo_root()
    fixture_dir = root / "docs" / "verified_source" / "json"
    output_dir = Path(args.output_dir)
    if not output_dir.is_absolute():
        output_dir = root / output_dir

    api_key = load_api_key(root, args.api_key_env)
    if not api_key:
        print(f"ERROR: API 키를 찾을 수 없습니다. {args.api_key_env} 또는 .env를 확인하세요.", file=sys.stderr)
        return 2

    cases = resolve_cases(args.cases)
    case_results: list[CaseExecution] = []
    raw_out: dict[str, Any] = {
        "meta": {
            "model": args.model,
            "search_mode": args.search_mode,
            "prompt_mode": args.prompt_mode,
            "timeout_sec": args.timeout,
            "max_retries": args.max_retries,
            "retry_delay_sec": args.retry_delay,
            "plugins": [{"id": "web", "engine": args.engine, "max_results": args.max_results}],
            "cases": cases,
            "requested_at": datetime.now(timezone.utc).isoformat(),
        },
        "cases": {},
    }

    for case_id in cases:
        fixture_name = CASE_TO_FIXTURE[case_id]
        fixture_obj = json.loads((fixture_dir / fixture_name).read_text(encoding="utf-8"))
        prompt = make_prompt(case_id, args.prompt_mode)

        print(f"[live] requesting {case_id} ...", flush=True)
        response = call_openrouter(
            api_key=api_key,
            model=args.model,
            search_mode=args.search_mode,
            engine=args.engine,
            max_results=args.max_results,
            case_id=case_id,
            prompt=prompt,
            prompt_mode=args.prompt_mode,
            timeout_sec=args.timeout,
            max_retries=args.max_retries,
            retry_delay=args.retry_delay,
        )

        raw_out["cases"][case_id] = response
        mode_used = str(response.get("mode_used", "unknown"))
        retry_attempts = response.get("retry_attempts")
        if isinstance(retry_attempts, int) and retry_attempts > 0:
            mode_used = f"{mode_used} (retry={retry_attempts})"

        if not response.get("ok"):
            case_results.append(
                CaseExecution(
                    case_id=case_id,
                    fixture=fixture_name,
                    ok=False,
                    elapsed_sec=float(response.get("elapsed_sec", 0.0)),
                    error=str(response.get("error", "unknown error")),
                    checks=[],
                    passed=0,
                    total=0,
                    web_search_requests=response.get("web_search_requests", "N/A"),
                    mode_used=mode_used,
                    actual=None,
                    request_payload=response.get("request_payload", {}),
                    response_usage=None,
                )
            )
            continue

        actual = response["parsed"]
        checks, passed, total = compare_case(case_id, fixture_obj, actual)
        case_results.append(
            CaseExecution(
                case_id=case_id,
                fixture=fixture_name,
                ok=True,
                elapsed_sec=float(response.get("elapsed_sec", 0.0)),
                error=None,
                checks=checks,
                passed=passed,
                total=total,
                web_search_requests=response.get("web_search_requests", "N/A"),
                mode_used=mode_used,
                actual=actual,
                request_payload=response.get("request_payload", {}),
                response_usage=response.get("usage"),
            )
        )

    report_path, raw_path = write_outputs(
        root=root,
        output_dir=output_dir,
        model=args.model,
        search_mode=args.search_mode,
        prompt_mode=args.prompt_mode,
        engine=args.engine,
        max_results=args.max_results,
        cases=case_results,
        raw_data=raw_out,
    )
    print(str(report_path))
    print(str(raw_path))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
