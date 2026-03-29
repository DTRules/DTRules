#!/usr/bin/env python3
"""
Visualize benchmark results from CSV file.
Generates comparison charts and summary tables.

Usage:
    python3 plot_results.py results/benchmark_YYYYMMDD_HHMMSS.csv
"""

import sys
import csv
from collections import defaultdict

def load_csv(filename):
    """Load benchmark results from CSV."""
    results = defaultdict(lambda: defaultdict(dict))

    with open(filename, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            impl = row['implementation']
            bench = row['benchmark']
            metric = row['metric']
            value = float(row['value'])
            unit = row['unit']

            results[bench][impl][metric] = (value, unit)

    return results


def print_comparison_table(results):
    """Print a formatted comparison table."""

    print()
    print("=" * 78)
    print("BENCHMARK COMPARISON")
    print("=" * 78)

    # Cold Start Latency
    if 'cold_start' in results:
        print()
        print("COLD START LATENCY (lower is better)")
        print("-" * 60)
        print(f"{'Implementation':<15} {'Min':>10} {'Mean':>10} {'P99':>10}")
        print("-" * 60)

        cold = results['cold_start']
        impls = sorted(cold.keys())

        for impl in impls:
            metrics = cold[impl]
            min_val = metrics.get('min', (0, 'ms'))[0]
            mean_val = metrics.get('mean', (0, 'ms'))[0]
            p99_val = metrics.get('p99', (0, 'ms'))[0]
            print(f"{impl:<15} {min_val:>9.2f}ms {mean_val:>9.2f}ms {p99_val:>9.2f}ms")

        # Calculate speedup
        if 'asm' in cold and len(impls) > 1:
            asm_mean = cold['asm'].get('mean', (1, 'ms'))[0]
            print()
            print("Speedup vs ASM (cold start):")
            for impl in impls:
                if impl != 'asm':
                    other_mean = cold[impl].get('mean', (1, 'ms'))[0]
                    if asm_mean > 0:
                        speedup = other_mean / asm_mean
                        print(f"  {impl}: ASM is {speedup:.1f}x faster")

    # Throughput
    if 'throughput' in results:
        print()
        print()
        print("THROUGHPUT (higher is better)")
        print("-" * 60)
        print(f"{'Implementation':<15} {'Ops/sec':>12} {'Mean Lat':>10} {'P99 Lat':>10}")
        print("-" * 60)

        tp = results['throughput']
        impls = sorted(tp.keys())

        for impl in impls:
            metrics = tp[impl]
            ops = metrics.get('ops_per_sec', (0, 'ops/s'))[0]
            mean = metrics.get('mean_latency', (0, 'ms'))[0]
            p99 = metrics.get('p99_latency', (0, 'ms'))[0]
            print(f"{impl:<15} {ops:>11.1f}/s {mean:>9.2f}ms {p99:>9.2f}ms")

        # Calculate speedup
        if 'asm' in tp and len(impls) > 1:
            asm_ops = tp['asm'].get('ops_per_sec', (1, 'ops/s'))[0]
            print()
            print("Throughput comparison vs ASM:")
            for impl in impls:
                if impl != 'asm':
                    other_ops = tp[impl].get('ops_per_sec', (1, 'ops/s'))[0]
                    if other_ops > 0:
                        ratio = asm_ops / other_ops
                        print(f"  {impl}: ASM is {ratio:.1f}x faster")

    # Memory
    if 'memory' in results:
        print()
        print()
        print("MEMORY USAGE (lower is better)")
        print("-" * 60)
        print(f"{'Implementation':<15} {'Peak RSS':>15}")
        print("-" * 60)

        mem = results['memory']
        impls = sorted(mem.keys())

        for impl in impls:
            metrics = mem[impl]
            rss = metrics.get('peak_rss', (0, 'KB'))[0]
            if rss > 1024:
                print(f"{impl:<15} {rss/1024:>14.1f} MB")
            else:
                print(f"{impl:<15} {rss:>14.0f} KB")

        # Calculate ratio
        if 'asm' in mem and len(impls) > 1:
            asm_mem = mem['asm'].get('peak_rss', (1, 'KB'))[0]
            print()
            print("Memory comparison vs ASM:")
            for impl in impls:
                if impl != 'asm':
                    other_mem = mem[impl].get('peak_rss', (1, 'KB'))[0]
                    if asm_mem > 0:
                        ratio = other_mem / asm_mem
                        print(f"  {impl}: uses {ratio:.1f}x more memory than ASM")

    # Binary Size
    if 'size' in results:
        print()
        print()
        print("BINARY SIZE")
        print("-" * 60)
        print(f"{'Implementation':<15} {'Size':>15}")
        print("-" * 60)

        size = results['size']
        impls = sorted(size.keys())

        for impl in impls:
            metrics = size[impl]
            binary = metrics.get('binary', (0, 'KB'))[0]
            if binary > 1024:
                print(f"{impl:<15} {binary/1024:>14.1f} MB")
            else:
                print(f"{impl:<15} {binary:>14.0f} KB")


def generate_ascii_chart(results, benchmark, metric, title):
    """Generate a simple ASCII bar chart."""
    if benchmark not in results:
        return

    data = results[benchmark]
    impls = sorted(data.keys())

    values = []
    for impl in impls:
        val = data[impl].get(metric, (0, ''))[0]
        values.append((impl, val))

    if not values:
        return

    max_val = max(v[1] for v in values) or 1
    bar_width = 40

    print()
    print(title)
    print("-" * 60)

    for impl, val in values:
        bar_len = int((val / max_val) * bar_width)
        bar = "█" * bar_len
        print(f"{impl:<8} {bar:<{bar_width}} {val:.2f}")


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 plot_results.py <benchmark.csv>")
        print()
        print("Example:")
        print("  python3 plot_results.py results/benchmark_20240115_103045.csv")
        sys.exit(1)

    filename = sys.argv[1]

    try:
        results = load_csv(filename)
    except FileNotFoundError:
        print(f"Error: File not found: {filename}")
        sys.exit(1)
    except Exception as e:
        print(f"Error reading file: {e}")
        sys.exit(1)

    print_comparison_table(results)

    # ASCII charts
    generate_ascii_chart(results, 'cold_start', 'mean',
                        "Cold Start Latency (ms, lower is better)")
    generate_ascii_chart(results, 'throughput', 'ops_per_sec',
                        "Throughput (ops/sec, higher is better)")
    generate_ascii_chart(results, 'memory', 'peak_rss',
                        "Memory Usage (KB, lower is better)")

    print()
    print("=" * 78)


if __name__ == '__main__':
    main()
