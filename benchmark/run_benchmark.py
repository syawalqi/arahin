#!/usr/bin/env python3
"""
ARAHIN Benchmark Runner
Runs 100 prompts × 3 modes (bare, pipeline, agent) = 300 LLM calls.
Parses output, compares against ground truth, computes P/R/F1.
"""

import json
import subprocess
import sys
import os
import re
import time
import argparse
import csv
from pathlib import Path
from math import radians, sin, cos, sqrt, atan2

# Config
ARAHIN_BIN = "/usr/local/bin/arahin"
DATASET_PATH = "/root/arahin/benchmark/msntb.json"
RESULTS_DIR = Path("/root/arahin/benchmark/results")
MODES = ["bare", "pipeline", "agent"]
TIMEOUT_PER_CALL = 120  # seconds
DELAY_BETWEEN_CALLS = 1  # seconds to avoid rate limiting

# Load environment
def load_env():
    env = os.environ.copy()
    try:
        with open("/etc/default/arahin-web") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    key, val = line.split("=", 1)
                    env[key.strip()] = val.strip().strip('"').strip("'")
    except FileNotFoundError:
        pass
    return env

ENV = load_env()

def parse_arahin_output(stdout: str) -> dict:
    """Parse arahin route output to extract waypoints, distance, time."""
    result = {
        "waypoints": [],
        "total_distance_km": 0,
        "total_time_min": 0,
        "raw_output": stdout,
        "error": None,
    }

    # Check for errors
    if "error" in stdout.lower() or "traceback" in stdout.lower():
        result["error"] = stdout[-500:] if len(stdout) > 500 else stdout
        return result

    # Extract waypoints: "START Name (lat, lng)" or "  • Name (lat, lng)" or "END Name (lat, lng)"
    waypoint_pattern = r'(?:START|•|END)\s+(.+?)\s+\((-?[\d.]+),\s*(-?[\d.]+)\)'
    matches = re.findall(waypoint_pattern, stdout)
    for name, lat, lng in matches:
        result["waypoints"].append({
            "name": name.strip(),
            "lat": float(lat),
            "lng": float(lng),
        })

    # Extract total distance and time: "Total: X.X km (X min)"
    total_pattern = r'Total:\s+([\d.]+)\s+km\s+\((\d+)\s+min\)'
    total_match = re.search(total_pattern, stdout)
    if total_match:
        result["total_distance_km"] = float(total_match.group(1))
        result["total_time_min"] = int(total_match.group(2))

    return result

def haversine(lat1, lon1, lat2, lon2):
    """Calculate distance in km between two points."""
    R = 6371.0
    lat1, lon1, lat2, lon2 = map(radians, [lat1, lon1, lat2, lon2])
    dlat = lat2 - lat1
    dlon = lon2 - lon1
    a = sin(dlat/2)**2 + cos(lat1) * cos(lat2) * sin(dlon/2)**2
    c = 2 * atan2(sqrt(a), sqrt(1-a))
    return R * c

def match_waypoint(extracted, ground_truth, threshold_km=1.0):
    """Check if an extracted waypoint matches a ground truth waypoint."""
    # Name match (case-insensitive contains)
    ext_name = extracted["name"].lower()
    gt_name = ground_truth["name"].lower()
    if gt_name in ext_name or ext_name in gt_name:
        return True

    # Coordinate match (within threshold_km)
    dist = haversine(extracted["lat"], extracted["lng"], ground_truth["lat"], ground_truth["lng"])
    return dist <= threshold_km

def score_single(extracted_wps, ground_truth_wps):
    """Score a single prompt's extraction against ground truth."""
    tp = 0
    matched_gt = set()
    matched_ext = set()

    for i, ext in enumerate(extracted_wps):
        for j, gt in enumerate(ground_truth_wps):
            if j not in matched_gt and match_waypoint(ext, gt):
                tp += 1
                matched_gt.add(j)
                matched_ext.add(i)
                break

    fp = len(extracted_wps) - tp
    fn = len(ground_truth_wps) - tp

    precision = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) > 0 else 0.0

    return {
        "tp": tp, "fp": fp, "fn": fn,
        "precision": round(precision, 4),
        "recall": round(recall, 4),
        "f1": round(f1, 4),
        "n_extracted": len(extracted_wps),
        "n_ground_truth": len(ground_truth_wps),
    }

def run_single(prompt: str, mode: str, timeout: int = TIMEOUT_PER_CALL) -> dict:
    """Run a single arahin call and return parsed result."""
    cmd = [ARAHIN_BIN, "route", "--mode", mode, prompt]
    try:
        proc = subprocess.run(
            cmd, capture_output=True, text=True,
            timeout=timeout, env=ENV
        )
        stdout = proc.stdout + proc.stderr
        result = parse_arahin_output(stdout)
        result["exit_code"] = proc.returncode
        return result
    except subprocess.TimeoutExpired:
        return {
            "waypoints": [], "total_distance_km": 0, "total_time_min": 0,
            "raw_output": f"TIMEOUT after {timeout}s", "error": "TIMEOUT",
            "exit_code": -1,
        }
    except Exception as e:
        return {
            "waypoints": [], "total_distance_km": 0, "total_time_min": 0,
            "raw_output": str(e), "error": str(e), "exit_code": -1,
        }

def main():
    parser = argparse.ArgumentParser(description="ARAHIN Benchmark Runner")
    parser.add_argument("--limit", type=int, default=0, help="Limit to N prompts (0=all)")
    parser.add_argument("--mode", choices=MODES + ["all"], default="all", help="Run specific mode or all")
    parser.add_argument("--resume", action="store_true", help="Resume from existing results")
    args = parser.parse_args()

    # Load dataset
    with open(DATASET_PATH, "r", encoding="utf-8") as f:
        dataset = json.load(f)

    if args.limit > 0:
        dataset = dataset[:args.limit]

    modes = MODES if args.mode == "all" else [args.mode]
    total_calls = len(dataset) * len(modes)

    print(f"ARAHIN Benchmark Runner")
    print(f"  Dataset: {len(dataset)} prompts")
    print(f"  Modes: {modes}")
    print(f"  Total calls: {total_calls}")
    print(f"  Timeout: {TIMEOUT_PER_CALL}s per call")
    print(f"  Delay: {DELAY_BETWEEN_CALLS}s between calls")
    print()

    RESULTS_DIR.mkdir(parents=True, exist_ok=True)

    # Load existing results if resuming
    all_results = {}
    if args.resume:
        for mode in modes:
            result_file = RESULTS_DIR / f"results_{mode}.json"
            if result_file.exists():
                with open(result_file) as f:
                    mode_results = json.load(f)
                for r in mode_results:
                    all_results[(r["id"], mode)] = r
                print(f"  Resumed {len(mode_results)} results for {mode}")

    # Run benchmark
    call_num = 0
    errors = 0
    for mode in modes:
        mode_results = []
        for entry in dataset:
            call_num += 1
            key = (entry["id"], mode)

            # Skip if already done (resume mode)
            if key in all_results:
                mode_results.append(all_results[key])
                continue

            prompt = entry["prompt"]
            print(f"[{call_num}/{total_calls}] {entry['id']} ({entry['difficulty']}) [{mode}]")

            result = run_single(prompt, mode)
            score = score_single(result["waypoints"], entry["ground_truth"])

            entry_result = {
                "id": entry["id"],
                "difficulty": entry["difficulty"],
                "prompt": prompt,
                "mode": mode,
                "n_ground_truth": len(entry["ground_truth"]),
                "n_extracted": result["waypoints"] is not None and len(result["waypoints"]) or 0,
                "waypoints": result["waypoints"],
                "total_distance_km": result["total_distance_km"],
                "total_time_min": result["total_time_min"],
                "error": result["error"],
                "exit_code": result["exit_code"],
                "score": score,
            }

            mode_results.append(entry_result)

            if result["error"]:
                errors += 1
                print(f"  ERROR: {result['error'][:80]}")
            else:
                print(f"  → {len(result['waypoints'])} waypoints, "
                      f"P={score['precision']:.2f} R={score['recall']:.2f} F1={score['f1']:.2f}")

            # Save after each prompt (for resume capability)
            with open(RESULTS_DIR / f"results_{mode}.json", "w") as f:
                json.dump(mode_results, f, ensure_ascii=False, indent=2)

            time.sleep(DELAY_BETWEEN_CALLS)

        all_results_for_mode = mode_results

    # Combine all results
    all_combined = []
    for mode in modes:
        result_file = RESULTS_DIR / f"results_{mode}.json"
        if result_file.exists():
            with open(result_file) as f:
                all_combined.extend(json.load(f))

    # Save combined results
    with open(RESULTS_DIR / "results_all.json", "w") as f:
        json.dump(all_combined, f, ensure_ascii=False, indent=2)

    # Print summary
    print()
    print(f"Benchmark complete: {call_num} calls, {errors} errors")
    print(f"Results saved to: {RESULTS_DIR}")

    # Quick summary per mode
    for mode in modes:
        mode_data = [r for r in all_combined if r["mode"] == mode]
        if mode_data:
            avg_p = sum(r["score"]["precision"] for r in mode_data) / len(mode_data)
            avg_r = sum(r["score"]["recall"] for r in mode_data) / len(mode_data)
            avg_f1 = sum(r["score"]["f1"] for r in mode_data) / len(mode_data)
            print(f"  {mode}: P={avg_p:.3f} R={avg_r:.3f} F1={avg_f1:.3f} (n={len(mode_data)})")

if __name__ == "__main__":
    main()
