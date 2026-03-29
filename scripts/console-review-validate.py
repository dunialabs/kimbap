#!/usr/bin/env python3

import argparse
import pathlib
import re
import sys


EXPECTED_VIEWS = ["dashboard", "actions", "approvals", "audit", "services"]
FINAL_STATUSES = {"closed", "blocked", "oracle_review"}
ROUND_RESULT_TYPES = {"pending", "fix", "no_op", "blocked"}


def require(pattern: str, text: str, label: str) -> str:
    match = re.search(pattern, text, re.MULTILINE)
    if not match:
        raise ValueError(f"missing required field: {label}")
    return match.group(1) if match.groups() else ""


def section_block(text: str, section: str) -> str:
    pattern = rf"^{re.escape(section)}:\s*$\n((?:^(?:\s{{2,}}).*$\n?)*)"
    match = re.search(pattern, text, re.MULTILINE)
    if not match:
        raise ValueError(f"missing required section: {section}")
    return match.group(1)


def nested_block(block: str, key: str, key_indent: int) -> str:
    child_indent = key_indent + 2
    pattern = rf"^\s{{{key_indent}}}{re.escape(key)}:\s*$\n((?:^\s{{{child_indent},}}.*$\n?)*)"
    match = re.search(pattern, block, re.MULTILINE)
    if not match:
        raise ValueError(f"missing required nested section: {key}")
    return match.group(1)


def list_items(block: str, key: str, key_indent: int) -> list[str]:
    inline_pattern = rf"^\s{{{key_indent}}}{re.escape(key)}:\s*\[\s*\]\s*$"
    if re.search(inline_pattern, block, re.MULTILINE):
        return []

    list_pattern = rf"^\s{{{key_indent}}}{re.escape(key)}:\s*$\n((?:^\s{{{key_indent + 2}}}-\s+.*$\n?)*)"
    match = re.search(list_pattern, block, re.MULTILINE)
    if not match:
        raise ValueError(f"missing required list: {key}")

    items_raw: list[str] = []
    for found in re.finditer(rf"^\s{{{key_indent + 2}}}-\s+(.*)$", match.group(1), re.MULTILINE):
        item_value = found.group(1)
        if not isinstance(item_value, str):
            raise ValueError(f"invalid list item for {key}")
        items_raw.append(item_value)
    return [item.strip().strip('"').strip("'") for item in items_raw]


def scalar_value(block: str, key: str, key_indent: int) -> str:
    pattern = rf"^\s{{{key_indent}}}{re.escape(key)}:\s*(.+)\s*$"
    match = re.search(pattern, block, re.MULTILINE)
    if not match:
        raise ValueError(f"missing required scalar: {key}")
    value = match.group(1).strip()
    return value.strip('"').strip("'")


def screenshot_views(screenshots: list[str]) -> set[str]:
    views: set[str] = set()
    for entry in screenshots:
        for token in entry.split("|"):
            token = token.strip()
            if token.startswith("view="):
                value = token.split("=", 1)[1].strip()
                if value:
                    views.add(value)
    return views


def validate_views(text: str, status: str) -> None:
    coverage = section_block(text, "coverage")
    required_order = list_items(coverage, "required_view_order", 2)
    if required_order != EXPECTED_VIEWS:
        raise ValueError("coverage.required_view_order must exactly match dashboard, actions, approvals, audit, services")

    inspected = list_items(coverage, "known_views_inspected", 2)
    if status in FINAL_STATUSES and inspected != EXPECTED_VIEWS:
        raise ValueError("coverage.known_views_inspected must contain all required views in fixed order for finalized rounds")


def next_id_for(round_id: str) -> str:
    if not re.fullmatch(r"R\d{3}", round_id):
        raise ValueError("round must use R### format")
    current = int(round_id[1:])
    if current < 1 or current > 100:
        raise ValueError("round out of range (R001-R100)")
    if current == 100:
        return "WRAPUP"
    return f"R{current + 1:03d}"


def validate_round_report(path: pathlib.Path) -> None:
    text = path.read_text(encoding="utf-8")

    round_id = require(r"^round:\s*(R\d{3})\s*$", text, "round")
    status = require(r"^status:\s*(queued|in_progress|oracle_review|closed|blocked)\s*$", text, "status")

    app = section_block(text, "app")
    launch_command = scalar_value(app, "launch_command", 2)
    if launch_command != "kimbap serve --console --port 8080":
        raise ValueError("app.launch_command must be 'kimbap serve --console --port 8080'")
    app_url = scalar_value(app, "url", 2)
    if app_url != "http://localhost:8080/console":
        raise ValueError("app.url must be 'http://localhost:8080/console'")
    launched_at = scalar_value(app, "launched_at", 2)

    _ = require(r"^\s*status:\s*(pending|pass|fail)\s*$", text, "oracle_review.status")
    _ = require(r"^\s*result:\s*(pending|pass|fail)\s*$", text, "regression_spot_check.result")
    next_round = require(r"^\s*id:\s*(R\d{3}|WRAPUP)\s*$", text, "next_round.id")

    round_result = section_block(text, "round_result")
    round_result_type = scalar_value(round_result, "type", 2)
    if round_result_type not in ROUND_RESULT_TYPES:
        raise ValueError("round_result.type must be one of: pending, fix, no_op, blocked")
    if status in FINAL_STATUSES and round_result_type == "pending":
        raise ValueError("round_result.type cannot be pending for finalized rounds")

    _ = list_items(text, "wording_changes", 0)
    screenshots = list_items(text, "screenshots", 0)
    changes = section_block(text, "changes")
    _ = list_items(changes, "files_changed", 2)
    _ = list_items(changes, "files_unchanged", 2)
    verification = section_block(text, "verification")
    verification_commands = list_items(verification, "commands", 2)
    if status in FINAL_STATUSES and len(verification_commands) == 0:
        raise ValueError("verification.commands must contain at least one command for finalized rounds")

    oracle_review = section_block(text, "oracle_review")
    oracle_status = scalar_value(oracle_review, "status", 2)
    regression = section_block(text, "regression_spot_check")
    regression_result = scalar_value(regression, "result", 2)

    if status in FINAL_STATUSES:
        if oracle_status == "pending":
            raise ValueError("oracle_review.status cannot be pending for finalized rounds")
        if regression_result == "pending":
            raise ValueError("regression_spot_check.result cannot be pending for finalized rounds")
        if launched_at == "":
            raise ValueError("app.launched_at cannot be empty for finalized rounds")
        if len(screenshots) == 0:
            raise ValueError("screenshots must contain at least one entry for finalized rounds")
        shot_views = screenshot_views(screenshots)
        missing = [v for v in EXPECTED_VIEWS if v not in shot_views]
        if missing:
            raise ValueError(f"screenshots must include evidence for each required view; missing: {', '.join(missing)}")

    validate_views(text, status)

    expected_next = next_id_for(round_id)
    if next_round != expected_next:
        raise ValueError(f"next_round.id must be {expected_next} for {round_id}, got {next_round}")


def validate_wrapup_report(path: pathlib.Path) -> None:
    text = path.read_text(encoding="utf-8")
    _ = require(r"^round:\s*WRAPUP\s*$", text, "round")
    _ = require(r"^status:\s*closed\s*$", text, "status")

    summary = scalar_value(text, "summary", 0)
    if summary == "":
        raise ValueError("summary cannot be empty for WRAPUP")

    final_checks = section_block(text, "final_checks")
    total_rounds = scalar_value(final_checks, "total_rounds", 2)
    completed_rounds = scalar_value(final_checks, "completed_rounds", 2)
    ledger_verified = scalar_value(final_checks, "ledger_verified", 2)
    oracle_verified = scalar_value(final_checks, "oracle_verified", 2)

    if total_rounds != "100":
        raise ValueError("final_checks.total_rounds must be 100")
    if completed_rounds != "100":
        raise ValueError("final_checks.completed_rounds must be 100")
    if ledger_verified != "true":
        raise ValueError("final_checks.ledger_verified must be true")
    if oracle_verified != "true":
        raise ValueError("final_checks.oracle_verified must be true")


def main() -> int:
    parser = argparse.ArgumentParser()
    _ = parser.add_argument("report", help="path to round report YAML")
    args = parser.parse_args()

    report_value = args.__dict__.get("report")
    if not isinstance(report_value, str) or not report_value:
        print("invalid report path argument", file=sys.stderr)
        return 1

    report_path = pathlib.Path(report_value).resolve()
    if not report_path.exists():
        print(f"report not found: {report_path}", file=sys.stderr)
        return 1

    try:
        text = report_path.read_text(encoding="utf-8")
        round_value = require(r"^round:\s*([A-Z0-9]+)\s*$", text, "round")
        if round_value == "WRAPUP":
            validate_wrapup_report(report_path)
        else:
            validate_round_report(report_path)
    except ValueError as exc:
        print(f"invalid round report: {exc}", file=sys.stderr)
        return 1

    print(f"round report valid: {report_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
