#!/bin/bash
# DTRules Cross-Platform Test Runner
# Orchestrates tests across Java, Go, and ASM implementations

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
JAVA_PASS=0
JAVA_FAIL=0
GO_PASS=0
GO_FAIL=0
ASM_PASS=0
ASM_FAIL=0
COMPARISON_PASS=0
COMPARISON_FAIL=0

# Options
RUN_JAVA=true
RUN_GO=true
RUN_ASM=true
RUN_COMPARISON=true
VERBOSE=false
JUNIT_OUTPUT=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --java-only)
            RUN_GO=false
            RUN_ASM=false
            RUN_COMPARISON=false
            shift
            ;;
        --go-only)
            RUN_JAVA=false
            RUN_ASM=false
            RUN_COMPARISON=false
            shift
            ;;
        --asm-only)
            RUN_JAVA=false
            RUN_GO=false
            RUN_COMPARISON=false
            shift
            ;;
        --no-java)
            RUN_JAVA=false
            shift
            ;;
        --no-go)
            RUN_GO=false
            shift
            ;;
        --no-asm)
            RUN_ASM=false
            shift
            ;;
        --comparison-only)
            RUN_JAVA=false
            RUN_GO=false
            RUN_ASM=false
            RUN_COMPARISON=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --junit)
            JUNIT_OUTPUT="$2"
            shift 2
            ;;
        --help|-h)
            echo "DTRules Cross-Platform Test Runner"
            echo ""
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --java-only       Run only Java tests"
            echo "  --go-only         Run only Go tests"
            echo "  --asm-only        Run only ASM tests"
            echo "  --no-java         Skip Java tests"
            echo "  --no-go           Skip Go tests"
            echo "  --no-asm          Skip ASM tests"
            echo "  --comparison-only Run only comparison tests"
            echo "  --verbose, -v     Verbose output"
            echo "  --junit FILE      Output JUnit XML to FILE"
            echo "  --help, -h        Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}     DTRules Cross-Platform Test Suite     ${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""
echo "Project: $PROJECT_DIR"
echo "Date: $(date)"
echo ""

#------------------------------------------------------------------------------
# Java Tests
#------------------------------------------------------------------------------
if [ "$RUN_JAVA" = true ]; then
    echo -e "${YELLOW}=== Java Tests ===${NC}"
    echo ""

    # Check for Maven
    if command -v mvn &> /dev/null; then
        # Check for pom.xml
        if [ -f "$PROJECT_DIR/pom.xml" ]; then
            echo "Running Maven tests..."
            if $VERBOSE; then
                if mvn -f "$PROJECT_DIR/pom.xml" test 2>&1; then
                    JAVA_PASS=1
                    echo -e "${GREEN}Java tests: PASSED${NC}"
                else
                    JAVA_FAIL=1
                    echo -e "${RED}Java tests: FAILED${NC}"
                fi
            else
                if mvn -f "$PROJECT_DIR/pom.xml" test -q 2>&1; then
                    JAVA_PASS=1
                    echo -e "${GREEN}Java tests: PASSED${NC}"
                else
                    JAVA_FAIL=1
                    echo -e "${RED}Java tests: FAILED${NC}"
                fi
            fi
        else
            echo -e "${YELLOW}Warning: No pom.xml found, skipping Java tests${NC}"
        fi
    else
        echo -e "${YELLOW}Warning: Maven not found, skipping Java tests${NC}"
    fi
    echo ""
fi

#------------------------------------------------------------------------------
# Go Tests
#------------------------------------------------------------------------------
if [ "$RUN_GO" = true ]; then
    echo -e "${YELLOW}=== Go Tests ===${NC}"
    echo ""

    GO_DIR="$PROJECT_DIR/go"
    if [ -d "$GO_DIR" ]; then
        echo "Running Go tests..."
        cd "$GO_DIR"

        if $VERBOSE; then
            if go test -v ./... 2>&1; then
                GO_PASS=1
                echo -e "${GREEN}Go tests: PASSED${NC}"
            else
                GO_FAIL=1
                echo -e "${RED}Go tests: FAILED${NC}"
            fi
        else
            if go test ./... 2>&1; then
                GO_PASS=1
                echo -e "${GREEN}Go tests: PASSED${NC}"
            else
                GO_FAIL=1
                echo -e "${RED}Go tests: FAILED${NC}"
            fi
        fi

        cd "$PROJECT_DIR"
    else
        echo -e "${YELLOW}Warning: Go directory not found, skipping Go tests${NC}"
    fi
    echo ""
fi

#------------------------------------------------------------------------------
# ASM Tests
#------------------------------------------------------------------------------
if [ "$RUN_ASM" = true ]; then
    echo -e "${YELLOW}=== ASM Tests ===${NC}"
    echo ""

    ASM_DIR="$PROJECT_DIR/asm"
    if [ -d "$ASM_DIR" ]; then
        # Check for NASM
        if command -v nasm &> /dev/null; then
            echo "Running ASM tests..."

            if [ -f "$ASM_DIR/test/run_tests.sh" ]; then
                if $VERBOSE; then
                    if "$ASM_DIR/test/run_tests.sh" 2>&1; then
                        ASM_PASS=1
                        echo -e "${GREEN}ASM tests: PASSED${NC}"
                    else
                        ASM_FAIL=1
                        echo -e "${RED}ASM tests: FAILED${NC}"
                    fi
                else
                    if "$ASM_DIR/test/run_tests.sh" > /tmp/asm_tests.log 2>&1; then
                        ASM_PASS=1
                        echo -e "${GREEN}ASM tests: PASSED${NC}"
                    else
                        ASM_FAIL=1
                        echo -e "${RED}ASM tests: FAILED${NC}"
                        echo "See /tmp/asm_tests.log for details"
                    fi
                fi
            elif [ -f "$ASM_DIR/Makefile" ]; then
                if make -C "$ASM_DIR" test > /tmp/asm_tests.log 2>&1; then
                    ASM_PASS=1
                    echo -e "${GREEN}ASM tests: PASSED${NC}"
                else
                    ASM_FAIL=1
                    echo -e "${RED}ASM tests: FAILED${NC}"
                    echo "See /tmp/asm_tests.log for details"
                fi
            else
                echo -e "${YELLOW}Warning: No ASM test runner found${NC}"
            fi
        else
            echo -e "${YELLOW}Warning: NASM not found, skipping ASM tests${NC}"
        fi
    else
        echo -e "${YELLOW}Warning: ASM directory not found, skipping ASM tests${NC}"
    fi
    echo ""
fi

#------------------------------------------------------------------------------
# Comparison Tests (ASM vs Go)
#------------------------------------------------------------------------------
if [ "$RUN_COMPARISON" = true ]; then
    echo -e "${YELLOW}=== Comparison Tests (ASM vs Go) ===${NC}"
    echo ""

    ASM_DIR="$PROJECT_DIR/asm"
    if [ -d "$ASM_DIR/test/comparison" ]; then
        echo "Running comparison tests..."

        if make -C "$ASM_DIR" test-comparison > /tmp/comparison_tests.log 2>&1; then
            COMPARISON_PASS=1
            echo -e "${GREEN}Comparison tests: PASSED${NC}"
        else
            COMPARISON_FAIL=1
            echo -e "${RED}Comparison tests: FAILED${NC}"
            if $VERBOSE; then
                cat /tmp/comparison_tests.log
            else
                echo "See /tmp/comparison_tests.log for details"
            fi
        fi
    else
        echo -e "${YELLOW}Warning: Comparison test directory not found${NC}"
    fi
    echo ""
fi

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}                 Summary                    ${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

TOTAL_PASS=0
TOTAL_FAIL=0

if [ "$RUN_JAVA" = true ]; then
    if [ $JAVA_FAIL -gt 0 ]; then
        echo -e "Java:       ${RED}FAILED${NC}"
        TOTAL_FAIL=$((TOTAL_FAIL + 1))
    elif [ $JAVA_PASS -gt 0 ]; then
        echo -e "Java:       ${GREEN}PASSED${NC}"
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        echo -e "Java:       ${YELLOW}SKIPPED${NC}"
    fi
fi

if [ "$RUN_GO" = true ]; then
    if [ $GO_FAIL -gt 0 ]; then
        echo -e "Go:         ${RED}FAILED${NC}"
        TOTAL_FAIL=$((TOTAL_FAIL + 1))
    elif [ $GO_PASS -gt 0 ]; then
        echo -e "Go:         ${GREEN}PASSED${NC}"
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        echo -e "Go:         ${YELLOW}SKIPPED${NC}"
    fi
fi

if [ "$RUN_ASM" = true ]; then
    if [ $ASM_FAIL -gt 0 ]; then
        echo -e "ASM:        ${RED}FAILED${NC}"
        TOTAL_FAIL=$((TOTAL_FAIL + 1))
    elif [ $ASM_PASS -gt 0 ]; then
        echo -e "ASM:        ${GREEN}PASSED${NC}"
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        echo -e "ASM:        ${YELLOW}SKIPPED${NC}"
    fi
fi

if [ "$RUN_COMPARISON" = true ]; then
    if [ $COMPARISON_FAIL -gt 0 ]; then
        echo -e "Comparison: ${RED}FAILED${NC}"
        TOTAL_FAIL=$((TOTAL_FAIL + 1))
    elif [ $COMPARISON_PASS -gt 0 ]; then
        echo -e "Comparison: ${GREEN}PASSED${NC}"
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        echo -e "Comparison: ${YELLOW}SKIPPED${NC}"
    fi
fi

echo ""
echo -e "${BLUE}--------------------------------------------${NC}"
echo "Total: $TOTAL_PASS passed, $TOTAL_FAIL failed"
echo ""

# Generate JUnit XML if requested
if [ -n "$JUNIT_OUTPUT" ]; then
    echo "Generating JUnit XML report: $JUNIT_OUTPUT"
    cat > "$JUNIT_OUTPUT" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="DTRules" tests="$((TOTAL_PASS + TOTAL_FAIL))" failures="$TOTAL_FAIL">
  <testsuite name="Java" tests="$((JAVA_PASS + JAVA_FAIL))" failures="$JAVA_FAIL">
    <testcase name="Java Tests" classname="dtrules.java">
      $([ $JAVA_FAIL -gt 0 ] && echo '<failure message="Java tests failed"/>')
    </testcase>
  </testsuite>
  <testsuite name="Go" tests="$((GO_PASS + GO_FAIL))" failures="$GO_FAIL">
    <testcase name="Go Tests" classname="dtrules.go">
      $([ $GO_FAIL -gt 0 ] && echo '<failure message="Go tests failed"/>')
    </testcase>
  </testsuite>
  <testsuite name="ASM" tests="$((ASM_PASS + ASM_FAIL))" failures="$ASM_FAIL">
    <testcase name="ASM Tests" classname="dtrules.asm">
      $([ $ASM_FAIL -gt 0 ] && echo '<failure message="ASM tests failed"/>')
    </testcase>
  </testsuite>
  <testsuite name="Comparison" tests="$((COMPARISON_PASS + COMPARISON_FAIL))" failures="$COMPARISON_FAIL">
    <testcase name="Comparison Tests" classname="dtrules.comparison">
      $([ $COMPARISON_FAIL -gt 0 ] && echo '<failure message="Comparison tests failed"/>')
    </testcase>
  </testsuite>
</testsuites>
EOF
fi

# Exit with appropriate code
if [ $TOTAL_FAIL -gt 0 ]; then
    echo -e "${RED}OVERALL: FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}OVERALL: PASSED${NC}"
    exit 0
fi
