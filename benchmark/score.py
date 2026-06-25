#!/usr/bin/env python3
"""
ARAHIN Benchmark Scorer
Generates detailed analysis from benchmark results.
Output: CSV per-prompt scores, aggregate tables, error analysis.
"""

import json
import csv
import sys
from pathlib import Path
from collections import defaultdict

RESULTS_DIR = Path("/root/arahin/benchmark/results")
OUTPUT_DIR = RESULTS_DIR

def load_results():
    """Load combined results."""
    results_file = RESULTS_DIR / "results_all.json"
    if not results_file.exists():
        print(f"ERROR: {results_file} not found. Run run_benchmark.py first.")
        sys.exit(1)
    with open(results_file, "r", encoding="utf-8") as f:
        return json.load(f)

def aggregate_by_mode(results):
    """Compute aggregate P/R/F1 per mode."""
    by_mode = defaultdict(list)
    for r in results:
        by_mode[r["mode"]].append(r)

    agg = {}
    for mode, data in by_mode.items():
        n = len(data)
        avg_p = sum(r["score"]["precision"] for r in data) / n
        avg_r = sum(r["score"]["recall"] for r in data) / n
        avg_f1 = sum(r["score"]["f1"] for r in data) / n
        total_tp = sum(r["score"]["tp"] for r in data)
        total_fp = sum(r["score"]["fp"] for r in data)
        total_fn = sum(r["score"]["fn"] for r in data)
        n_errors = sum(1 for r in data if r.get("error"))

        agg[mode] = {
            "n": n,
            "precision": round(avg_p, 4),
            "recall": round(avg_r, 4),
            "f1": round(avg_f1, 4),
            "total_tp": total_tp,
            "total_fp": total_fp,
            "total_fn": total_fn,
            "n_errors": n_errors,
        }
    return agg

def aggregate_by_difficulty(results):
    """Compute aggregate P/R/F1 per mode × difficulty."""
    by_key = defaultdict(list)
    for r in results:
        key = (r["mode"], r["difficulty"])
        by_key[key].append(r)

    agg = {}
    for (mode, diff), data in by_key.items():
        n = len(data)
        avg_p = sum(r["score"]["precision"] for r in data) / n
        avg_r = sum(r["score"]["recall"] for r in data) / n
        avg_f1 = sum(r["score"]["f1"] for r in data) / n
        n_errors = sum(1 for r in data if r.get("error"))

        agg[(mode, diff)] = {
            "n": n,
            "precision": round(avg_p, 4),
            "recall": round(avg_r, 4),
            "f1": round(avg_f1, 4),
            "n_errors": n_errors,
        }
    return agg

def error_analysis(results):
    """Categorize error patterns."""
    patterns = {
        "timeout": 0,
        "no_waypoints": 0,
        "partial_extraction": 0,
        "over_extraction": 0,
        "hallucination": 0,
        "correct": 0,
    }

    error_details = []
    for r in results:
        if r.get("error") == "TIMEOUT":
            patterns["timeout"] += 1
            error_details.append({"id": r["id"], "mode": r["mode"], "type": "timeout", "detail": "TIMEOUT"})
        elif r.get("error"):
            patterns["timeout"] += 1  # other errors
            error_details.append({"id": r["id"], "mode": r["mode"], "type": "error", "detail": r["error"][:100]})
        elif len(r["waypoints"]) == 0:
            patterns["no_waypoints"] += 1
            error_details.append({"id": r["id"], "mode": r["mode"], "type": "no_waypoints", "detail": "0 waypoints extracted"})
        elif r["score"]["fn"] > 0 and r["score"]["fp"] > 0:
            patterns["partial_extraction"] += 1
        elif r["score"]["fp"] > 0:
            patterns["over_extraction"] += 1
        elif r["score"]["f1"] == 1.0:
            patterns["correct"] += 1
        elif r["score"]["fn"] > 0:
            patterns["partial_extraction"] += 1

    return patterns, error_details

def generate_csv(results):
    """Write per-prompt scores to CSV."""
    csv_path = OUTPUT_DIR / "scores_per_prompt.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "id", "difficulty", "mode", "prompt", "n_ground_truth", "n_extracted",
            "tp", "fp", "fn", "precision", "recall", "f1",
            "total_distance_km", "total_time_min", "error"
        ])
        for r in results:
            writer.writerow([
                r["id"], r["difficulty"], r["mode"], r["prompt"],
                r["n_ground_truth"], r.get("n_extracted", len(r.get("waypoints", []))),
                r["score"]["tp"], r["score"]["fp"], r["score"]["fn"],
                r["score"]["precision"], r["score"]["recall"], r["score"]["f1"],
                r.get("total_distance_km", 0), r.get("total_time_min", 0),
                r.get("error", ""),
            ])
    print(f"CSV saved: {csv_path}")

def main():
    results = load_results()
    print(f"Loaded {len(results)} results\n")

    # 1. Aggregate by mode
    mode_agg = aggregate_by_mode(results)
    print("=" * 70)
    print("RQ1: AGGREGATE RESULTS BY MODE")
    print("=" * 70)
    print(f"{'Mode':<12} {'N':>5} {'Precision':>10} {'Recall':>10} {'F1':>10} {'Errors':>7}")
    print("-" * 70)
    for mode in ["bare", "pipeline", "agent"]:
        if mode in mode_agg:
            a = mode_agg[mode]
            print(f"{mode:<12} {a['n']:>5} {a['precision']:>10.4f} {a['recall']:>10.4f} {a['f1']:>10.4f} {a['n_errors']:>7}")
    print()

    # 2. Aggregate by difficulty
    diff_agg = aggregate_by_difficulty(results)
    print("=" * 70)
    print("RQ1: RESULTS BY MODE × DIFFICULTY")
    print("=" * 70)
    print(f"{'Mode':<12} {'Difficulty':<10} {'N':>5} {'Precision':>10} {'Recall':>10} {'F1':>10}")
    print("-" * 70)
    for mode in ["bare", "pipeline", "agent"]:
        for diff in ["easy", "medium", "hard"]:
            key = (mode, diff)
            if key in diff_agg:
                a = diff_agg[key]
                print(f"{mode:<12} {diff:<10} {a['n']:>5} {a['precision']:>10.4f} {a['recall']:>10.4f} {a['f1']:>10.4f}")
        print()

    # 3. Error analysis
    patterns, error_details = error_analysis(results)
    print("=" * 70)
    print("ERROR ANALYSIS")
    print("=" * 70)
    for pattern, count in sorted(patterns.items(), key=lambda x: -x[1]):
        if count > 0:
            print(f"  {pattern}: {count}")
    print()

    # 4. Generate CSV
    generate_csv(results)

    # 5. Save summary JSON
    summary = {
        "total_results": len(results),
        "modes": mode_agg,
        "by_difficulty": {f"{k[0]}_{k[1]}": v for k, v in diff_agg.items()},
        "error_patterns": patterns,
    }
    summary_path = OUTPUT_DIR / "summary.json"
    with open(summary_path, "w") as f:
        json.dump(summary, f, indent=2)
    print(f"Summary saved: {summary_path}")

    # 6. Print worst cases for debugging
    print()
    print("=" * 70)
    print("WORST CASES (F1 < 0.5, first 10)")
    print("=" * 70)
    worst = sorted(results, key=lambda r: r["score"]["f1"])[:10]
    for r in worst:
        print(f"  {r['id']} [{r['mode']}] F1={r['score']['f1']:.2f}: {r['prompt'][:60]}")

if __name__ == "__main__":
    main()
