#!/bin/bash
# Test script for validation and merge functionality
# This demonstrates common error cases and how the scripts detect them

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
XML_DIR="$PROJECT_DIR/xml"
TEST_DIR="/tmp/dtrules_extraction_test_$$"

echo "=== DTRules Multi-File XML Validation Test Suite ==="
echo ""
echo "Test directory: $TEST_DIR"
echo ""

# Create test directory
mkdir -p "$TEST_DIR/xml/states"

# Test 1: Create a well-formed minimal decision table file
echo "Test 1: Well-formed decision table"
cat > "$TEST_DIR/xml/states/TEST_dt.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Test_Table</table_name>
<xls_file>Test.xls</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>Test table</COMMENTS>
<TABLE_NUMBER>99999</TABLE_NUMBER>
<FILE_PATH>states/TEST/99999_Test_Table</FILE_PATH>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
EOF

# Test 2: Create a well-formed minimal EDD file
echo "Test 2: Well-formed entity definition"
cat > "$TEST_DIR/xml/states/TEST_edd.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version='2' xmlns:xs='http://www.w3.org/2001/XMLSchema'>
<entity name='test_entity' xls_file='Test.xls' access='rw' comment='Test entity'>
    <field name='test_field' type='string' subtype='' access='rw' input='' default_value='' comment='Test field'></field>
</entity>
</entity_data_dictionary>
EOF

# Test 3: Create malformed XML (missing closing tag)
echo "Test 3: Malformed XML (should be detected)"
cat > "$TEST_DIR/xml/states/BAD_dt.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Bad_Table</table_name>
<!-- Missing closing tags -->
EOF

# Test 4: Create duplicate table number
echo "Test 4: Duplicate table number"
cat > "$TEST_DIR/xml/states/DUP_dt.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Duplicate_Table</table_name>
<xls_file>Dup.xls</xls_file>
<attribute_fields>
<TABLE_NUMBER>99999</TABLE_NUMBER>
<FILE_PATH>states/DUP/99999_Duplicate</FILE_PATH>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
EOF

# Test 5: FILE_PATH mismatch
echo "Test 5: FILE_PATH mismatch (should warn)"
cat > "$TEST_DIR/xml/states/WRONG_dt.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Wrong_Path_Table</table_name>
<xls_file>Wrong.xls</xls_file>
<attribute_fields>
<TABLE_NUMBER>88888</TABLE_NUMBER>
<FILE_PATH>states/DIFFERENT/88888_Wrong</FILE_PATH>
</attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions></conditions>
<actions></actions>
</decision_table>
</decision_tables>
EOF

echo ""
echo "=== Running Validation on Test Files ==="
cd "$TEST_DIR"
python3 "$SCRIPT_DIR/validate_extraction.py" > "$TEST_DIR/validation_output.txt" 2>&1 || true

echo ""
echo "Validation output:"
cat "$TEST_DIR/validation_output.txt"

echo ""
echo "=== Running Merge on Test Files ==="
python3 "$SCRIPT_DIR/merge_files.py" --dry-run --verbose > "$TEST_DIR/merge_output.txt" 2>&1 || true

echo ""
echo "Merge output:"
cat "$TEST_DIR/merge_output.txt"

echo ""
echo "=== Expected Results ==="
echo "✅ Test 1 & 2 should pass (well-formed XML)"
echo "❌ Test 3 should fail (malformed XML detected)"
echo "❌ Test 4 should fail (duplicate table number)"
echo "⚠️  Test 5 should warn (FILE_PATH mismatch)"

echo ""
echo "=== Cleanup ==="
rm -rf "$TEST_DIR"
echo "Test directory removed"

echo ""
echo "=== Real Validation on Project Files ==="
echo "Running validation on actual TaxReturn XML files..."
cd "$PROJECT_DIR"
python3 scripts/validate_extraction.py 2>&1 | tail -50

echo ""
echo "=== Test Complete ==="
