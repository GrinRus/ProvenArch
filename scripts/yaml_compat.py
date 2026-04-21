#!/usr/bin/env python3
"""Small YAML compatibility loader for hermetic ACP scripts/tests.

Prefers PyYAML when available, but falls back to a minimal parser that
supports the repository's checked-in config subset: mappings, lists, nested
objects, integers, booleans, nulls, and plain/quoted strings.
"""

from __future__ import annotations

import ast
import re
from pathlib import Path
from typing import Any

_INLINE_KEY_RE = re.compile(r"^([A-Za-z0-9_.-]+):(.*)$")


def _strip_inline_comment(line: str) -> str:
    in_single = False
    in_double = False
    escaped = False
    for idx, ch in enumerate(line):
        if escaped:
            escaped = False
            continue
        if ch == "\\" and in_double:
            escaped = True
            continue
        if ch == "'" and not in_double:
            in_single = not in_single
            continue
        if ch == '"' and not in_single:
            in_double = not in_double
            continue
        if ch == "#" and not in_single and not in_double:
            if idx == 0 or line[idx - 1].isspace():
                return line[:idx].rstrip()
    return line.rstrip()


def _parse_scalar(raw: str) -> Any:
    text = raw.strip()
    if text == "":
        return ""
    lowered = text.lower()
    if lowered in {"null", "~"}:
        return None
    if lowered == "true":
        return True
    if lowered == "false":
        return False
    if re.fullmatch(r"-?\d+", text):
        return int(text)
    if (text.startswith('"') and text.endswith('"')) or (text.startswith("'") and text.endswith("'")):
        try:
            return ast.literal_eval(text)
        except Exception:
            return text[1:-1]
    return text


def _parse_inline_mapping_entry(text: str) -> tuple[str, Any | None, bool] | None:
    match = _INLINE_KEY_RE.match(text)
    if not match:
        return None
    key = match.group(1).strip()
    remainder = match.group(2)
    if not key:
        return None
    if remainder.strip() == "":
        return key, None, False
    return key, _parse_scalar(remainder.strip()), True


def _fallback_load_yaml(text: str) -> Any:
    tokens: list[tuple[int, str]] = []
    for lineno, raw_line in enumerate(text.splitlines(), start=1):
        if "\t" in raw_line:
            raise ValueError(f"tabs are not supported in fallback YAML parser (line {lineno})")
        stripped = _strip_inline_comment(raw_line)
        if stripped.strip() == "":
            continue
        indent = len(stripped) - len(stripped.lstrip(" "))
        tokens.append((indent, stripped.lstrip(" ")))

    index = 0

    def parse_child_block(parent_indent: int) -> Any:
        nonlocal index
        if index >= len(tokens):
            return {}
        next_indent, next_content = tokens[index]
        if next_indent < parent_indent:
            return {}
        if next_content.startswith("- "):
            return parse_list(next_indent)
        if next_indent == parent_indent:
            return {}
        return parse_block(next_indent)

    def parse_block(expected_indent: int) -> Any:
        nonlocal index
        if index >= len(tokens):
            return {}
        indent, content = tokens[index]
        if indent < expected_indent:
            return {}
        if content.startswith("- "):
            return parse_list(expected_indent)
        return parse_mapping(expected_indent)

    def parse_mapping(expected_indent: int) -> dict[str, Any]:
        nonlocal index
        result: dict[str, Any] = {}
        while index < len(tokens):
            indent, content = tokens[index]
            if indent < expected_indent:
                break
            if indent > expected_indent:
                raise ValueError(f"unexpected indentation before mapping entry: {content!r}")
            if content.startswith("- "):
                break
            parsed = _parse_inline_mapping_entry(content)
            if parsed is None:
                raise ValueError(f"invalid mapping entry: {content!r}")
            key, value, has_inline_value = parsed
            index += 1
            if has_inline_value:
                result[key] = value
                continue
            result[key] = parse_child_block(expected_indent)
        return result

    def parse_list(expected_indent: int) -> list[Any]:
        nonlocal index
        items: list[Any] = []
        while index < len(tokens):
            indent, content = tokens[index]
            if indent < expected_indent:
                break
            if indent > expected_indent:
                raise ValueError(f"unexpected indentation before list item: {content!r}")
            if not content.startswith("- "):
                break
            rest = content[2:].strip()
            index += 1
            if rest == "":
                items.append(parse_block(expected_indent + 2))
                continue
            parsed = _parse_inline_mapping_entry(rest)
            if parsed is None:
                items.append(_parse_scalar(rest))
                continue
            key, value, has_inline_value = parsed
            item: dict[str, Any] = {}
            if has_inline_value:
                item[key] = value
                if index < len(tokens) and tokens[index][0] >= expected_indent + 2:
                    extra = parse_block(tokens[index][0])
                    if isinstance(extra, dict):
                        item.update(extra)
            else:
                if index < len(tokens) and tokens[index][0] >= expected_indent + 2:
                    item[key] = parse_block(tokens[index][0])
                else:
                    item[key] = {}
            items.append(item)
        return items

    payload = parse_block(0)
    if index != len(tokens):
        raise ValueError("trailing unparsed YAML content")
    return payload


def load_yaml_text(text: str) -> Any:
    try:
        import yaml  # type: ignore
    except Exception:
        return _fallback_load_yaml(text)
    return yaml.safe_load(text)


def load_yaml_file(path: str | Path) -> Any:
    file_path = Path(path)
    return load_yaml_text(file_path.read_text(encoding="utf-8"))
