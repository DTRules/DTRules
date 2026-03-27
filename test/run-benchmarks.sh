#!/bin/bash
# DTRules Cross-Runtime Performance Benchmark Suite
# Runs benchmarks across all implementations and generates comparison charts

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/benchmark-results}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Create output directory
mkdir -p "$OUTPUT_DIR"

log_info "DTRules Cross-Runtime Performance Benchmark"
log_info "Output directory: $OUTPUT_DIR"
log_info "Timestamp: $TIMESTAMP"
echo

# =============================================================================
# Go Benchmarks (Go, NativeASM, x86-64-ASM runtimes)
# =============================================================================

run_go_benchmarks() {
    log_info "Running Go benchmarks (go, native-asm, x86-64-asm)..."

    cd "$ROOT_DIR/go"

    # Run standard Go benchmarks
    go test -bench=. -benchmem -count=3 \
        ./pkg/dtrules/runtime/... \
        2>&1 | tee "$OUTPUT_DIR/go_benchmarks_raw_$TIMESTAMP.txt"

    # Run export benchmarks
    RUN_BENCHMARKS=1 BENCHMARK_OUTPUT="$OUTPUT_DIR/go_benchmarks_$TIMESTAMP.json" \
        go test -v -run TestRunBenchmarksForExport -count=1 \
        ./pkg/dtrules/runtime/... \
        2>&1 | tee -a "$OUTPUT_DIR/go_benchmarks_raw_$TIMESTAMP.txt"

    log_success "Go benchmarks complete"
}

# =============================================================================
# Java Benchmarks
# =============================================================================

run_java_benchmarks() {
    log_info "Checking for Java benchmarks..."

    cd "$ROOT_DIR"

    # Check if Maven is available
    if command -v mvn &> /dev/null; then
        MVN_CMD="mvn"
    elif [ -f "/tmp/apache-maven-3.9.6/bin/mvn" ]; then
        MVN_CMD="/tmp/apache-maven-3.9.6/bin/mvn"
    else
        log_warning "Maven not found. Skipping Java benchmarks."
        log_warning "Install Maven and add JMH benchmarks to get real Java measurements."
        log_warning "Do NOT use fabricated numbers."
        return
    fi

    # Check if JMH benchmark class exists
    JMH_SRC="$ROOT_DIR/sampleprojects/CHIP/src/main/java/com/dtrules/samples/chipeligibility/BenchmarkChip.java"
    if [ ! -f "$JMH_SRC" ]; then
        log_warning "No JMH benchmark class found at $JMH_SRC"
        log_warning "Skipping Java benchmarks. To add real Java benchmarks:"
        log_warning "  1. Add JMH dependency to pom.xml"
        log_warning "  2. Create a JMH benchmark class that measures actual operations"
        log_warning "  3. Output results in JSON format to $OUTPUT_DIR/"
        return
    fi

    # Run actual JMH benchmarks and export results
    log_info "Running Java JMH benchmarks..."
    $MVN_CMD -pl sampleprojects/CHIP exec:java \
        -Dexec.mainClass="com.dtrules.samples.chipeligibility.BenchmarkChip" \
        -Dexec.args="--output $OUTPUT_DIR/java_benchmarks_$TIMESTAMP.json" \
        2>&1 | tee "$OUTPUT_DIR/java_test_$TIMESTAMP.txt"

    if [ -f "$OUTPUT_DIR/java_benchmarks_$TIMESTAMP.json" ]; then
        log_success "Java benchmarks complete"
    else
        log_warning "Java benchmark output not found. No Java results will be included."
    fi
}

# =============================================================================
# Merge Results and Generate Charts
# =============================================================================

merge_results() {
    log_info "Merging benchmark results..."

    # Use Python to merge JSON files and generate charts
    python3 << PYMERGE
import json
import os
from pathlib import Path

output_dir = "$OUTPUT_DIR"
timestamp = "$TIMESTAMP"

# Find all benchmark JSON files
go_file = Path(output_dir) / f"go_benchmarks_{timestamp}.json"
java_file = Path(output_dir) / f"java_benchmarks_{timestamp}.json"

all_results = []

if go_file.exists():
    with open(go_file) as f:
        all_results.extend(json.load(f))
    print(f"Loaded {len(all_results)} Go benchmark results")

if java_file.exists():
    with open(java_file) as f:
        java_results = json.load(f)
        all_results.extend(java_results)
    print(f"Loaded {len(java_results)} Java benchmark results")

# Save merged results
merged_file = Path(output_dir) / f"all_benchmarks_{timestamp}.json"
with open(merged_file, 'w') as f:
    json.dump(all_results, f, indent=2)

print(f"Merged {len(all_results)} total results to {merged_file}")
PYMERGE

    log_success "Results merged"
}

generate_charts() {
    log_info "Generating performance charts..."

    # Generate HTML chart using embedded JavaScript
    python3 << 'PYCHARTS'
import json
import os
from pathlib import Path

output_dir = os.environ.get('OUTPUT_DIR', 'benchmark-results')
timestamp = os.environ.get('TIMESTAMP', 'latest')

# Find the merged results file
results_file = None
for f in Path(output_dir).glob('all_benchmarks_*.json'):
    results_file = f
    break

if not results_file or not results_file.exists():
    # Try go benchmarks
    for f in Path(output_dir).glob('go_benchmarks_*.json'):
        results_file = f
        break

if not results_file or not results_file.exists():
    print("No benchmark results found")
    exit(1)

with open(results_file) as f:
    results = json.load(f)

# Group by category and operation
data = {}
runtimes = set()
for r in results:
    cat = r['category']
    op = r['operation']
    rt = r['runtime']
    runtimes.add(rt)

    key = f"{cat}/{op}"
    if key not in data:
        data[key] = {}
    data[key][rt] = r['ns_per_op']

# Sort runtimes for consistent ordering
runtime_order = ['java', 'go', 'native-asm', 'x86-64-asm']
runtimes = [r for r in runtime_order if r in runtimes]

# Generate HTML with Chart.js
html = f'''<!DOCTYPE html>
<html>
<head>
    <title>DTRules Runtime Performance Comparison</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {{ font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }}
        h1 {{ color: #333; text-align: center; }}
        h2 {{ color: #666; margin-top: 30px; }}
        .chart-container {{
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .summary {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }}
        .summary-card {{
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .summary-card h3 {{ margin: 0 0 10px 0; color: #333; }}
        .summary-card .value {{ font-size: 24px; font-weight: bold; color: #007bff; }}
        table {{ border-collapse: collapse; width: 100%; margin: 20px 0; }}
        th, td {{ border: 1px solid #ddd; padding: 12px; text-align: right; }}
        th {{ background: #f8f9fa; font-weight: bold; }}
        td:first-child {{ text-align: left; }}
        tr:nth-child(even) {{ background: #f8f9fa; }}
        .fastest {{ background: #d4edda !important; font-weight: bold; }}
    </style>
</head>
<body>
    <h1>DTRules Runtime Performance Comparison</h1>
    <p style="text-align:center;color:#666;">Generated: {timestamp}</p>

    <div class="summary">
        <div class="summary-card">
            <h3>Runtimes Tested</h3>
            <div class="value">{len(runtimes)}</div>
            <div>{', '.join(runtimes)}</div>
        </div>
        <div class="summary-card">
            <h3>Benchmarks Run</h3>
            <div class="value">{len(data)}</div>
        </div>
        <div class="summary-card">
            <h3>Total Measurements</h3>
            <div class="value">{len(results)}</div>
        </div>
    </div>
'''

# Generate data for JavaScript
categories = {}
for key, values in data.items():
    cat, op = key.split('/')
    if cat not in categories:
        categories[cat] = {'labels': [], 'datasets': {rt: [] for rt in runtimes}}
    categories[cat]['labels'].append(op)
    for rt in runtimes:
        categories[cat]['datasets'][rt].append(values.get(rt, 0))

# Colors for each runtime
colors = {
    'java': 'rgba(255, 99, 132, 0.8)',
    'go': 'rgba(54, 162, 235, 0.8)',
    'native-asm': 'rgba(75, 192, 192, 0.8)',
    'x86-64-asm': 'rgba(255, 206, 86, 0.8)'
}

# Generate charts for each category
for cat, cat_data in categories.items():
    html += f'''
    <h2>{cat.title()} Operations</h2>
    <div class="chart-container">
        <canvas id="chart_{cat}"></canvas>
    </div>
'''

# Add detailed table
html += '''
    <h2>Detailed Results (nanoseconds per operation)</h2>
    <table>
        <tr>
            <th>Category</th>
            <th>Operation</th>
'''
for rt in runtimes:
    html += f'            <th>{rt}</th>\n'
html += '        </tr>\n'

for key in sorted(data.keys()):
    cat, op = key.split('/')
    values = data[key]
    min_val = min(v for v in values.values() if v > 0)

    html += f'        <tr>\n            <td>{cat}</td>\n            <td>{op}</td>\n'
    for rt in runtimes:
        val = values.get(rt, 0)
        cls = ' class="fastest"' if val == min_val and val > 0 else ''
        html += f'            <td{cls}>{val:.1f}</td>\n'
    html += '        </tr>\n'

html += '    </table>\n'

# Add Chart.js script
html += '''
    <script>
'''

for cat, cat_data in categories.items():
    datasets_js = []
    for rt in runtimes:
        datasets_js.append(f'''{{
            label: '{rt}',
            data: {cat_data['datasets'][rt]},
            backgroundColor: '{colors.get(rt, "rgba(128,128,128,0.8)")}'
        }}''')

    html += f'''
        new Chart(document.getElementById('chart_{cat}'), {{
            type: 'bar',
            data: {{
                labels: {cat_data['labels']},
                datasets: [{', '.join(datasets_js)}]
            }},
            options: {{
                responsive: true,
                plugins: {{
                    title: {{
                        display: true,
                        text: '{cat.title()} Operations Performance (lower is better)'
                    }},
                    legend: {{
                        position: 'top'
                    }}
                }},
                scales: {{
                    y: {{
                        beginAtZero: true,
                        title: {{
                            display: true,
                            text: 'Nanoseconds per operation'
                        }}
                    }}
                }}
            }}
        }});
'''

html += '''
    </script>
</body>
</html>
'''

# Write HTML file
html_file = Path(output_dir) / f'benchmark_report_{timestamp}.html'
with open(html_file, 'w') as f:
    f.write(html)

print(f"Generated chart: {html_file}")

# Also generate a simple text summary
summary_file = Path(output_dir) / f'benchmark_summary_{timestamp}.txt'
with open(summary_file, 'w') as f:
    f.write("DTRules Runtime Performance Summary\n")
    f.write("=" * 60 + "\n\n")

    for cat in sorted(categories.keys()):
        f.write(f"\n{cat.upper()}\n")
        f.write("-" * 40 + "\n")
        f.write(f"{'Operation':<20}")
        for rt in runtimes:
            f.write(f"{rt:>15}")
        f.write("\n")

        for i, op in enumerate(categories[cat]['labels']):
            f.write(f"{op:<20}")
            for rt in runtimes:
                val = categories[cat]['datasets'][rt][i]
                f.write(f"{val:>12.1f} ns")
            f.write("\n")

print(f"Generated summary: {summary_file}")
PYCHARTS

    log_success "Charts generated"
}

# =============================================================================
# Main
# =============================================================================

main() {
    echo "========================================"
    echo "DTRules Performance Benchmark Suite"
    echo "========================================"
    echo

    # Export variables for Python scripts
    export OUTPUT_DIR
    export TIMESTAMP

    # Run benchmarks
    run_go_benchmarks
    echo
    run_java_benchmarks
    echo

    # Generate reports
    merge_results
    echo
    generate_charts
    echo

    log_success "Benchmark suite complete!"
    echo
    log_info "Results saved to: $OUTPUT_DIR"
    log_info "View report: $OUTPUT_DIR/benchmark_report_$TIMESTAMP.html"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --go-only)
            run_go_benchmarks
            exit 0
            ;;
        --java-only)
            run_java_benchmarks
            exit 0
            ;;
        --charts-only)
            export OUTPUT_DIR TIMESTAMP
            generate_charts
            exit 0
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "  --go-only      Run only Go benchmarks"
            echo "  --java-only    Run only Java benchmarks"
            echo "  --charts-only  Generate charts from existing data"
            echo "  -o, --output   Output directory (default: benchmark-results)"
            echo "  -h, --help     Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
