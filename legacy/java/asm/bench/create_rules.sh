#!/bin/bash
# Create benchmark rule files of varying complexity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RULES_DIR="$SCRIPT_DIR/rules"

mkdir -p "$RULES_DIR"

#------------------------------------------------------------------------------
# Simple: Basic arithmetic (minimal workload)
#------------------------------------------------------------------------------

cat > "$RULES_DIR/bench_simple.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench_simple">
    <decision_tables>
        <decision_table name="simple_math">
            <actions>
                <action name="compute">
                    3 4 add
                    10 3 sub
                    5 6 mul
                    20 4 div
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="compute"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
EOF

echo "Created: bench_simple.xml"

#------------------------------------------------------------------------------
# Medium: Stack operations and comparisons
#------------------------------------------------------------------------------

cat > "$RULES_DIR/bench_medium.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench_medium">
    <decision_tables>
        <decision_table name="stack_ops">
            <actions>
                <action name="stack_test">
                    1 2 3 4 5
                    dup swap rot over
                    10 20 lt
                    5 5 eq
                    true false and
                    false true or
                </action>
                <action name="math_test">
                    100 50 add
                    200 30 sub
                    7 8 mul
                    100 10 div
                    17 5 mod
                    42 neg
                    -100 abs
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="stack_test"/>
                    <action name="math_test"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
EOF

echo "Created: bench_medium.xml"

#------------------------------------------------------------------------------
# Large: Many operations (stress test)
#------------------------------------------------------------------------------

cat > "$RULES_DIR/bench_large.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench_large">
    <decision_tables>
        <decision_table name="stress_test">
            <actions>
                <action name="compute_1">
                    1 2 add 3 add 4 add 5 add
                    6 7 mul 8 mul
                    100 10 div 2 div
                    dup dup add add
                </action>
                <action name="compute_2">
                    10 20 30 40 50
                    swap rot swap rot
                    add add add add
                    2 mul 3 mul
                </action>
                <action name="compute_3">
                    1 1 eq
                    2 3 ne
                    5 10 lt
                    10 5 gt
                    5 5 le
                    5 5 ge
                </action>
                <action name="compute_4">
                    true true and
                    false true or
                    true not
                    true false xor
                </action>
                <action name="compute_5">
                    1000 999 sub
                    500 2 mul
                    1000000 1000 div
                    123456 1000 mod
                </action>
                <action name="compute_6">
                    -42 abs
                    42 neg neg
                    1 2 3 4 5 6 7 8 9 10
                    add add add add add add add add add
                </action>
                <action name="compute_7">
                    100 dup dup dup dup
                    add add add add
                    50 swap sub
                </action>
                <action name="compute_8">
                    1 2 over over
                    add add add
                    10 mul
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="compute_1"/>
                    <action name="compute_2"/>
                    <action name="compute_3"/>
                    <action name="compute_4"/>
                    <action name="compute_5"/>
                    <action name="compute_6"/>
                    <action name="compute_7"/>
                    <action name="compute_8"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
EOF

echo "Created: bench_large.xml"

#------------------------------------------------------------------------------
# Loop-heavy: Repeated iterations (when control flow is implemented)
#------------------------------------------------------------------------------

cat > "$RULES_DIR/bench_compute.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="bench_compute">
    <decision_tables>
        <decision_table name="heavy_compute">
            <actions>
                <action name="fibonacci_style">
                    <!-- Simulate fibonacci-like computation -->
                    1 1
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                    dup rot add
                </action>
                <action name="factorial_style">
                    <!-- Simulate factorial-like computation -->
                    1
                    2 mul
                    3 mul
                    4 mul
                    5 mul
                    6 mul
                    7 mul
                    8 mul
                    9 mul
                    10 mul
                </action>
                <action name="prime_check_style">
                    <!-- Simulate primality check operations -->
                    97
                    dup 2 mod 0 eq not
                    swap dup 3 mod 0 eq not
                    and
                    swap dup 5 mod 0 eq not
                    and
                    swap dup 7 mod 0 eq not
                    and
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="fibonacci_style"/>
                    <action name="factorial_style"/>
                    <action name="prime_check_style"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
EOF

echo "Created: bench_compute.xml"

echo ""
echo "Benchmark rules created in $RULES_DIR/"
