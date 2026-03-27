#!/bin/bash

# merge-states.sh
# Merges federal core XML files with state-specific files for corporate tax
# Similar to TaxReturn merge script but for corporate tax (Form 1120)

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Corporate Tax State File Merge${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( dirname "$SCRIPT_DIR" )"
XML_DIR="$PROJECT_DIR/xml"
STATES_DIR="$XML_DIR/states"

# Check if directories exist
if [ ! -d "$XML_DIR" ]; then
    echo -e "${RED}Error: XML directory not found: $XML_DIR${NC}"
    exit 1
fi

if [ ! -d "$STATES_DIR" ]; then
    echo -e "${YELLOW}Warning: States directory not found: $STATES_DIR${NC}"
    echo -e "${YELLOW}Creating states directory...${NC}"
    mkdir -p "$STATES_DIR"
fi

# Function to merge XML files
merge_xml_files() {
    local output_file=$1
    local core_file=$2
    local file_pattern=$3
    local tag_name=$4

    echo -e "${YELLOW}Merging $tag_name files...${NC}"

    # Start with XML declaration and opening tag from core file
    head -n 100 "$core_file" | grep -E "^<\?xml|^<$tag_name" > "$output_file"

    # Add core content (skip declaration and opening/closing tags)
    sed '1,/^<'$tag_name'/d; /^<\/'$tag_name'/d' "$core_file" >> "$output_file"

    # Add state files (skip declaration and opening/closing tags)
    local count=0
    for state_file in "$STATES_DIR"/$file_pattern; do
        if [[ "$state_file" == *"TEMPLATE"* ]]; then continue; fi
        if [ -f "$state_file" ]; then
            local state=$(basename "$state_file" | cut -d'_' -f1)
            echo "  - Adding $state"
            echo "" >> "$output_file"
            echo "  <!-- State: $state -->" >> "$output_file"
            sed '1,/^<'$tag_name'/d; /^<\/'$tag_name'/d' "$state_file" >> "$output_file"
            ((count++))
        fi
    done

    # Add closing tag
    echo "" >> "$output_file"
    echo "</$tag_name>" >> "$output_file"

    echo -e "${GREEN}  ✓ Merged $count state files${NC}"
}

# Check for core files
CORE_EDD="$XML_DIR/CorporateTax_edd_core.xml"
CORE_DT="$XML_DIR/CorporateTax_dt_core.xml"

if [ ! -f "$CORE_EDD" ]; then
    echo -e "${RED}Error: Core EDD not found: $CORE_EDD${NC}"
    exit 1
fi

if [ ! -f "$CORE_DT" ]; then
    echo -e "${RED}Error: Core DT not found: $CORE_DT${NC}"
    exit 1
fi

# Merge EDD files
OUTPUT_EDD="$XML_DIR/CorporateTax_edd.xml"
merge_xml_files "$OUTPUT_EDD" "$CORE_EDD" "*_corp_edd.xml" "entity_dictionary"

# Merge DT files
OUTPUT_DT="$XML_DIR/CorporateTax_dt.xml"
merge_xml_files "$OUTPUT_DT" "$CORE_DT" "*_corp_dt.xml" "decision_tables"

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}Merge Complete!${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo "Output files:"
echo "  - $OUTPUT_EDD"
echo "  - $OUTPUT_DT"
echo ""
echo "Next steps:"
echo "  cd ../../go"
echo "  go test ./pkg/dtrules/... -run TestCorporateTax"
echo ""
