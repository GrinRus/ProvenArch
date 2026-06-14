from __future__ import annotations

import re

FAILURE_CLASS_PRECEDENCE = {
    "summary_missing": 0,
    "runtime_timeout": 1,
    "runtime_contract_failed": 2,
    "runner_unavailable": 3,
    "infra_signal_terminated": 4,
    "infra_incomplete_cycle": 5,
    "runtime_flow_failed": 6,
    "precheck_failed": 7,
    "none": 99,
}

EXPLICIT_RUNNER_UNAVAILABLE_PATTERNS = (
    r"\brunner[_ -]?unavailable\b",
)

GENERIC_RUNNER_UNAVAILABLE_PATTERNS = (
    r"\bmodel(?:\s+is)?\s+at\s+capacity\b",
    r"\brate[ -]?limit(?:ed)?\b",
    r"\btoo many requests\b",
    r"\b429\b",
)

CODEX_RUNTIME_NOISE_PATTERNS = (
    r"chatgpt\.com/backend-api/plugins/featured",
    r"\bcloudflare\b",
    r"failed to renew cache ttl",
    r"\bstate db\b",
    r"operation not permitted",
)


def failure_class_rank(value: str) -> int:
    return FAILURE_CLASS_PRECEDENCE.get((value or "").strip(), 98)


def terminal_process_failed_summary(summary_exists: bool, run_state: str, summary_written: str) -> bool:
    return bool(
        summary_exists
        and str(run_state or "").strip() == "process_failed"
        and str(summary_written or "").strip() == "yes"
    )


def terminal_success_summary(
    summary_exists: bool,
    run_status_exists: bool,
    result_value: str,
    api_status: str,
    run_state: str,
    process_exit: int,
) -> bool:
    return bool(
        summary_exists
        and run_status_exists
        and str(result_value or "") == "passed"
        and str(api_status or "") == "succeeded"
        and str(run_state or "").strip() == "completed"
        and process_exit == 0
    )


def should_ignore_stale_classified_failure(terminal_success: bool, classified_failure: str) -> bool:
    return bool(terminal_success and classified_failure not in {"", "none"})


def should_ignore_classified_incomplete_for_terminal_process(
    terminal_process_failure: bool,
    classified_failure: str,
    failure_reason: str,
) -> bool:
    return bool(
        terminal_process_failure
        and classified_failure == "infra_incomplete_cycle"
        and failure_reason != "infra_incomplete_cycle"
    )


def text_has_runner_unavailable_signal(text: str) -> bool:
    haystack = str(text or "")
    for pattern in EXPLICIT_RUNNER_UNAVAILABLE_PATTERNS:
        if re.search(pattern, haystack, flags=re.IGNORECASE):
            return True
    return text_has_raw_provider_runner_unavailable_signal(haystack)


def text_has_structured_runner_unavailable_signal(text: str) -> bool:
    haystack = str(text or "")
    for line in haystack.splitlines():
        if '"kind":"runtime_output"' in line or '"kind": "runtime_output"' in line:
            if text_has_raw_provider_runner_unavailable_signal(line):
                return True
            continue
        if text_has_runner_unavailable_signal(line):
            return True
    return False


def text_has_raw_provider_runner_unavailable_signal(text: str) -> bool:
    haystack = str(text or "")
    for line in haystack.splitlines():
        for pattern in GENERIC_RUNNER_UNAVAILABLE_PATTERNS:
            if re.search(pattern, line, flags=re.IGNORECASE):
                if any(re.search(noise_pattern, line, flags=re.IGNORECASE) for noise_pattern in CODEX_RUNTIME_NOISE_PATTERNS):
                    break
                return True
    return False


def text_has_runtime_contract_parse_signature(text: str) -> bool:
    haystack = str(text or "")
    return bool(
        re.search(r"parse runtime draft manifest", haystack, flags=re.IGNORECASE)
        and re.search(r"unknown field", haystack, flags=re.IGNORECASE)
    )


def text_has_collect_document_path_contract_signature(text: str) -> bool:
    haystack = str(text or "")
    patterns = (
        r"read shard document\b",
        r"manifest document path\b",
        r"documents\[[0-9]+\]\.path references missing document file\b",
        r"documents\[[0-9]+\]\.path references a directory\b",
        r"documents\[[0-9]+\]\.path stat document file\b",
    )
    return any(re.search(pattern, haystack, flags=re.IGNORECASE) for pattern in patterns)


def extract_focused_recovery_reason_tags(text: str) -> set[str]:
    haystack = str(text or "").lower()
    tags: set[str] = set()
    if (
        "collect manifest repair exhausted" in haystack
        or "manifest-only collect repair stalled" in haystack
        or "manifest-only collect repair did not produce valid collect artifacts" in haystack
    ):
        tags.add("collect_manifest_repair_exhausted")
    if (
        ("focused artifact repair exhausted" in haystack and "validator_verdict_repair" in haystack)
        or "verdict-only validator repair stalled" in haystack
        or "verdict-only validator repair did not produce a valid validator verdict contract" in haystack
    ):
        tags.add("validator_verdict_repair_exhausted")
    if (
        "validator verdict recovery write-set precheck failed" in haystack
        or "verdict-only validator repair wrote outside validator-verdict.json" in haystack
        or "validator repair wrote forbidden" in haystack
    ):
        tags.add("validator_verdict_repair_write_set_violation")
    if (
        ("focused artifact repair exhausted" in haystack and "draft_artifact_repair" in haystack)
        or "draft artifact repair stalled" in haystack
        or "draft artifact repair did not produce valid draft artifact contract" in haystack
    ):
        tags.add("draft_artifact_repair_exhausted")
    if (
        "draft recovery write_root precheck failed" in haystack
        or "draft recovery draft_final_root precheck failed" in haystack
        or "draft recovery wrote outside the draft artifact write set" in haystack
        or "draft repair wrote forbidden write_root files" in haystack
    ):
        tags.add("draft_artifact_repair_write_set_violation")
    return tags
