#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

import pandas as pd


WORKBOOK_COLUMNS = {
    "day": "账单周期",
    "model": "模型名称",
    "input_tokens": "Input 用量(tokens)",
    "uncached_tokens": "Uncached input(tokens)",
    "cached_tokens": "Cached reads input(tokens)",
    "cache_write_5m_tokens": "Cached writes:5m input(tokens)",
    "cache_write_1h_tokens": "Cached writes:1h input(tokens)",
    "output_tokens": "Output 用量(tokens)",
    "uncached_price_per_m": "Uncached input(/Mt)(¥)",
    "cached_price_per_m": "Cached reads input(/Mt)(¥)",
    "cache_write_5m_price_per_m": "Cached writes:5m input(/Mt)(¥)",
    "cache_write_1h_price_per_m": "Cached writes:1h input(/Mt)(¥)",
    "output_price_per_m": "Output 单价(/Mt)(¥)",
    "used_amount": "总价(¥)",
}

LOCAL_COLUMNS = {
    "day": "day",
    "model": "model",
    "input_tokens": "input_tokens",
    "cached_tokens": "cached_tokens",
    "cache_creation_tokens": "cache_creation_tokens",
    "output_tokens": "output_tokens",
    "input_amount": "input_amount",
    "cached_amount": "cached_amount",
    "cache_creation_amount": "cache_creation_amount",
    "output_amount": "output_amount",
    "used_amount": "used_amount",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Normalize PPIO workbook billing data and compare it with local aggregates.",
    )
    parser.add_argument("--workbook", required=True, help="Path to the PPIO .xlsx workbook")
    parser.add_argument(
        "--local-csv",
        help="Optional local aggregate CSV, such as tmp_ppio_logs_apr.csv",
    )
    parser.add_argument(
        "--start",
        required=True,
        help="Inclusive start date in YYYY-MM-DD",
    )
    parser.add_argument(
        "--end",
        required=True,
        help="Inclusive end date in YYYY-MM-DD",
    )
    parser.add_argument(
        "--output-dir",
        default="tmp/ppio_reconcile",
        help="Directory for normalized CSV outputs",
    )
    parser.add_argument(
        "--emit-sql",
        help="Optional path for a SQL file that loads workbook truth into a temp table",
    )
    return parser.parse_args()


def _date_series(series: pd.Series) -> pd.Series:
    return pd.to_datetime(series, errors="coerce").dt.date


def _float_series(df: pd.DataFrame, column: str) -> pd.Series:
    if column not in df.columns:
        return pd.Series([0.0] * len(df), index=df.index, dtype="float64")
    return pd.to_numeric(df[column], errors="coerce").fillna(0.0)


def _int_series(df: pd.DataFrame, column: str) -> pd.Series:
    if column not in df.columns:
        return pd.Series([0] * len(df), index=df.index, dtype="int64")
    return pd.to_numeric(df[column], errors="coerce").fillna(0).round().astype("int64")


def load_workbook_truth(path: Path, start: str, end: str) -> pd.DataFrame:
    raw = pd.read_excel(path)
    raw["day"] = _date_series(raw[WORKBOOK_COLUMNS["day"]])
    start_day = pd.to_datetime(start).date()
    end_day = pd.to_datetime(end).date()
    raw = raw[(raw["day"] >= start_day) & (raw["day"] <= end_day)].copy()
    raw["model"] = raw[WORKBOOK_COLUMNS["model"]].astype(str)

    raw["book_input_tokens"] = _int_series(raw, WORKBOOK_COLUMNS["input_tokens"])
    raw["book_uncached_tokens"] = _int_series(raw, WORKBOOK_COLUMNS["uncached_tokens"])
    raw["book_cached_tokens"] = _int_series(raw, WORKBOOK_COLUMNS["cached_tokens"])
    raw["book_cache_write_5m_tokens"] = _int_series(
        raw, WORKBOOK_COLUMNS["cache_write_5m_tokens"]
    )
    raw["book_cache_write_1h_tokens"] = _int_series(
        raw, WORKBOOK_COLUMNS["cache_write_1h_tokens"]
    )
    raw["book_cache_creation_tokens"] = (
        raw["book_cache_write_5m_tokens"] + raw["book_cache_write_1h_tokens"]
    )
    raw["book_output_tokens"] = _int_series(raw, WORKBOOK_COLUMNS["output_tokens"])
    raw["book_used_amount"] = _float_series(raw, WORKBOOK_COLUMNS["used_amount"])

    raw["book_input_amount"] = (
        raw["book_uncached_tokens"]
        * _float_series(raw, WORKBOOK_COLUMNS["uncached_price_per_m"])
        / 1_000_000.0
    )
    raw["book_cached_amount"] = (
        raw["book_cached_tokens"]
        * _float_series(raw, WORKBOOK_COLUMNS["cached_price_per_m"])
        / 1_000_000.0
    )
    raw["book_cache_creation_amount"] = (
        raw["book_cache_write_5m_tokens"]
        * _float_series(raw, WORKBOOK_COLUMNS["cache_write_5m_price_per_m"])
        / 1_000_000.0
        + raw["book_cache_write_1h_tokens"]
        * _float_series(raw, WORKBOOK_COLUMNS["cache_write_1h_price_per_m"])
        / 1_000_000.0
    )
    raw["book_output_amount"] = (
        raw["book_output_tokens"]
        * _float_series(raw, WORKBOOK_COLUMNS["output_price_per_m"])
        / 1_000_000.0
    )

    grouped = raw.groupby(["day", "model"], as_index=False).agg(
        {
            "book_input_tokens": "sum",
            "book_uncached_tokens": "sum",
            "book_cached_tokens": "sum",
            "book_cache_creation_tokens": "sum",
            "book_output_tokens": "sum",
            "book_input_amount": "sum",
            "book_cached_amount": "sum",
            "book_cache_creation_amount": "sum",
            "book_output_amount": "sum",
            "book_used_amount": "sum",
        }
    )
    return grouped.sort_values(["day", "book_used_amount"], ascending=[True, False])


def load_local_aggregate(path: Path, start: str, end: str) -> pd.DataFrame:
    raw = pd.read_csv(path)
    raw["day"] = _date_series(raw[LOCAL_COLUMNS["day"]])
    start_day = pd.to_datetime(start).date()
    end_day = pd.to_datetime(end).date()
    raw = raw[(raw["day"] >= start_day) & (raw["day"] <= end_day)].copy()
    raw["model"] = raw[LOCAL_COLUMNS["model"]].astype(str)

    grouped = raw.groupby(["day", "model"], as_index=False).agg(
        {
            LOCAL_COLUMNS["input_tokens"]: "sum",
            LOCAL_COLUMNS["cached_tokens"]: "sum",
            LOCAL_COLUMNS["cache_creation_tokens"]: "sum",
            LOCAL_COLUMNS["output_tokens"]: "sum",
            LOCAL_COLUMNS["input_amount"]: "sum",
            LOCAL_COLUMNS["cached_amount"]: "sum",
            LOCAL_COLUMNS["cache_creation_amount"]: "sum",
            LOCAL_COLUMNS["output_amount"]: "sum",
            LOCAL_COLUMNS["used_amount"]: "sum",
        }
    )
    grouped = grouped.rename(
        columns={
            LOCAL_COLUMNS["input_tokens"]: "local_input_tokens",
            LOCAL_COLUMNS["cached_tokens"]: "local_cached_tokens",
            LOCAL_COLUMNS["cache_creation_tokens"]: "local_cache_creation_tokens",
            LOCAL_COLUMNS["output_tokens"]: "local_output_tokens",
            LOCAL_COLUMNS["input_amount"]: "local_input_amount",
            LOCAL_COLUMNS["cached_amount"]: "local_cached_amount",
            LOCAL_COLUMNS["cache_creation_amount"]: "local_cache_creation_amount",
            LOCAL_COLUMNS["output_amount"]: "local_output_amount",
            LOCAL_COLUMNS["used_amount"]: "local_used_amount",
        }
    )
    return grouped.sort_values(["day", "local_used_amount"], ascending=[True, False])


def build_diff(truth: pd.DataFrame, local: pd.DataFrame) -> pd.DataFrame:
    merged = truth.merge(local, on=["day", "model"], how="outer").fillna(0)
    for field in [
        "input_tokens",
        "cached_tokens",
        "cache_creation_tokens",
        "output_tokens",
        "input_amount",
        "cached_amount",
        "cache_creation_amount",
        "output_amount",
        "used_amount",
    ]:
        merged[f"delta_{field}"] = merged[f"local_{field}"] - merged[f"book_{field}"]
    merged["abs_delta_used_amount"] = merged["delta_used_amount"].abs()
    merged["ratio_used_amount"] = merged.apply(
        lambda row: row["local_used_amount"] / row["book_used_amount"]
        if row["book_used_amount"]
        else 0,
        axis=1,
    )
    return merged.sort_values(["day", "abs_delta_used_amount"], ascending=[True, False])


def write_sql_truth_table(truth: pd.DataFrame, path: Path) -> None:
    rows = [
        "-- Generated by scripts/ppio_billing_reconcile.py",
        "DROP TABLE IF EXISTS tmp_ppio_truth;",
        "CREATE TEMP TABLE tmp_ppio_truth (",
        "  day date NOT NULL,",
        "  model text NOT NULL,",
        "  input_tokens bigint NOT NULL,",
        "  uncached_tokens bigint NOT NULL,",
        "  cached_tokens bigint NOT NULL,",
        "  cache_creation_tokens bigint NOT NULL,",
        "  output_tokens bigint NOT NULL,",
        "  input_amount numeric NOT NULL,",
        "  cached_amount numeric NOT NULL,",
        "  cache_creation_amount numeric NOT NULL,",
        "  output_amount numeric NOT NULL,",
        "  used_amount numeric NOT NULL,",
        "  PRIMARY KEY (day, model)",
        ");",
        "",
        "INSERT INTO tmp_ppio_truth (",
        "  day, model, input_tokens, uncached_tokens, cached_tokens, cache_creation_tokens,",
        "  output_tokens, input_amount, cached_amount, cache_creation_amount, output_amount, used_amount",
        ") VALUES",
    ]

    value_lines = []
    for row in truth.itertuples(index=False):
        model = str(row.model).replace("'", "''")
        value_lines.append(
            "  ('{day}', '{model}', {input_tokens}, {uncached_tokens}, {cached_tokens}, "
            "{cache_creation_tokens}, {output_tokens}, {input_amount:.8f}, {cached_amount:.8f}, "
            "{cache_creation_amount:.8f}, {output_amount:.8f}, {used_amount:.8f})".format(
                day=row.day,
                model=model,
                input_tokens=int(row.book_input_tokens),
                uncached_tokens=int(row.book_uncached_tokens),
                cached_tokens=int(row.book_cached_tokens),
                cache_creation_tokens=int(row.book_cache_creation_tokens),
                output_tokens=int(row.book_output_tokens),
                input_amount=float(row.book_input_amount),
                cached_amount=float(row.book_cached_amount),
                cache_creation_amount=float(row.book_cache_creation_amount),
                output_amount=float(row.book_output_amount),
                used_amount=float(row.book_used_amount),
            )
        )
    rows.append(",\n".join(value_lines) + ";")
    path.write_text("\n".join(rows) + "\n", encoding="utf-8")


def print_summary(truth: pd.DataFrame, diff: pd.DataFrame | None) -> None:
    print("Workbook rows:", len(truth))
    print("Workbook total:", round(float(truth["book_used_amount"].sum()), 4))
    if diff is None:
        return

    print("Local total:", round(float(diff["local_used_amount"].sum()), 4))
    print("Delta total:", round(float(diff["delta_used_amount"].sum()), 4))
    early = diff[
        (diff["day"] >= pd.to_datetime("2026-04-01").date())
        & (diff["day"] <= pd.to_datetime("2026-04-07").date())
    ]
    if not early.empty:
        print("Early-window delta:", round(float(early["delta_used_amount"].sum()), 4))
    print("\nTop amount deltas:")
    preview = diff.nlargest(15, "abs_delta_used_amount")[
        [
            "day",
            "model",
            "book_used_amount",
            "local_used_amount",
            "delta_used_amount",
            "ratio_used_amount",
        ]
    ]
    print(preview.to_string(index=False))


def main() -> None:
    args = parse_args()
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    truth = load_workbook_truth(Path(args.workbook), args.start, args.end)
    truth_path = output_dir / "ppio_truth_by_day_model.csv"
    truth.to_csv(truth_path, index=False)

    diff = None
    if args.local_csv:
        local = load_local_aggregate(Path(args.local_csv), args.start, args.end)
        local_path = output_dir / "local_by_day_model.csv"
        local.to_csv(local_path, index=False)
        diff = build_diff(truth, local)
        diff_path = output_dir / "ppio_vs_local_by_day_model.csv"
        diff.to_csv(diff_path, index=False)

    if args.emit_sql:
        write_sql_truth_table(truth, Path(args.emit_sql))

    print_summary(truth, diff)
    print("\nWrote:", truth_path)
    if args.local_csv:
        print("Wrote:", output_dir / "local_by_day_model.csv")
        print("Wrote:", output_dir / "ppio_vs_local_by_day_model.csv")
    if args.emit_sql:
        print("Wrote:", Path(args.emit_sql))


if __name__ == "__main__":
    main()
