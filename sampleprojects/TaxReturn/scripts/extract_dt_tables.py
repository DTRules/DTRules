#!/usr/bin/env python3
"""
extract_dt_tables.py - Extract decision tables from monolithic TaxReturn_dt.xml

This script extracts decision tables from the monolithic TaxReturn_dt.xml file
into separate files organized by FILE_PATH metadata. Tables are categorized into:
- Core tables (no state-specific logic)
- State-specific tables (by state code)
- Form-specific tables (by form number)

Usage:
    python3 extract_dt_tables.py

Input:
    xml/TaxReturn_dt.xml (original monolithic file)

Output:
    xml/TaxReturn_dt_core.xml (core tables)
    xml/states/XX_dt.xml (state-specific tables)
    /tmp/extraction_report.txt (detailed report)
"""

import re
import sys
from pathlib import Path
from xml.etree import ElementTree as ET
from collections import defaultdict

# State codes for identification
STATE_CODES = [
    'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'DC', 'FL',
    'GA', 'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME',
    'MD', 'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH',
    'NJ', 'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI',
    'SC', 'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY'
]

class DecisionTableExtractor:
    def __init__(self, input_file, output_dir):
        self.input_file = Path(input_file)
        self.output_dir = Path(output_dir)
        self.tables_by_category = defaultdict(list)
        self.stats = defaultdict(int)

    def parse_tables(self):
        """Parse the monolithic XML file and extract tables."""
        print(f"Parsing {self.input_file}...")

        # Read the file as text to preserve formatting
        with open(self.input_file, 'r', encoding='utf-8') as f:
            content = f.read()

        # Split into individual decision_table blocks
        tables = []
        current_pos = 0

        while True:
            start = content.find('<decision_table>', current_pos)
            if start == -1:
                break

            end = content.find('</decision_table>', start)
            if end == -1:
                break

            end += len('</decision_table>')
            table_xml = content[start:end]
            tables.append(table_xml)
            current_pos = end

        print(f"Found {len(tables)} decision tables")
        self.stats['total_tables'] = len(tables)

        return tables

    def categorize_table(self, table_xml):
        """Categorize a table based on its name and number."""
        # Extract table name and number
        name_match = re.search(r'<table_name>([^<]+)</table_name>', table_xml)

        # Try both formats for table number
        number_match = re.search(r'<TABLE_NUMBER>(\d+)</TABLE_NUMBER>', table_xml)
        if not number_match:
            number_match = re.search(r'<table_number>(\d+)</table_number>', table_xml)

        if not name_match:
            return 'unknown', None

        table_name = name_match.group(1)

        # If no table number found, still try to categorize by name
        if not number_match:
            table_number = 0
        else:
            table_number = int(number_match.group(1))

        # Check for state-specific tables by name pattern
        for state_code in STATE_CODES:
            if f'_{state_code}_' in table_name or table_name.endswith(f'_{state_code}'):
                return f'state_{state_code}', state_code

        # Check by table number range
        # 40000+ are state tables
        if table_number >= 40000:
            # Try to determine state from context
            # Tables in 40000 range: dispatcher and state calculations
            if 'Dispatch_State' in table_name or 'Allocate_Income_By_State' in table_name:
                return 'core', None

            # Try to extract state from nearby content or action calls
            # Look for execute calls or state references
            for state_code in STATE_CODES:
                if f'Calculate_{state_code}_Tax' in table_xml or f'_{state_code}_' in table_xml:
                    return f'state_{state_code}', state_code

            # If we can't determine state, it's a core table
            return 'core', None

        # Everything else is core
        return 'core', None

    def add_file_path(self, table_xml, category, state_code):
        """Add FILE_PATH metadata to a table if it doesn't exist."""
        # Check if FILE_PATH already exists
        if '<FILE_PATH>' in table_xml:
            return table_xml

        # Extract table number and name for FILE_PATH
        number_match = re.search(r'<TABLE_NUMBER>(\d+)</TABLE_NUMBER>', table_xml)
        if not number_match:
            number_match = re.search(r'<table_number>(\d+)</table_number>', table_xml)

        name_match = re.search(r'<table_name>([^<]+)</table_name>', table_xml)

        if not name_match:
            return table_xml

        table_name = name_match.group(1)
        table_number = number_match.group(1) if number_match else '0000'

        # Construct FILE_PATH
        if state_code:
            file_path = f'states/{state_code}/{table_number}_{table_name}'
        else:
            file_path = f'core/{table_number}_{table_name}'

        # Insert FILE_PATH after TABLE_NUMBER or table_number
        file_path_tag = f'<FILE_PATH>{file_path}</FILE_PATH>'

        # Find position to insert (after TABLE_NUMBER or table_number closing tag)
        number_end = table_xml.find('</TABLE_NUMBER>')
        if number_end == -1:
            number_end = table_xml.find('</table_number>')
            if number_end == -1:
                # No table number field, insert after attribute_fields if it exists
                attr_end = table_xml.find('</attribute_fields>')
                if attr_end != -1:
                    insert_pos = attr_end + len('</attribute_fields>')
                    new_table_xml = (
                        table_xml[:insert_pos] +
                        '\n<FILE_PATH>' + file_path + '</FILE_PATH>' +
                        table_xml[insert_pos:]
                    )
                    return new_table_xml
                # Otherwise insert after table_description
                desc_end = table_xml.find('</table_description>')
                if desc_end != -1:
                    insert_pos = desc_end + len('</table_description>')
                    new_table_xml = (
                        table_xml[:insert_pos] +
                        '\n<FILE_PATH>' + file_path + '</FILE_PATH>' +
                        table_xml[insert_pos:]
                    )
                    return new_table_xml
                return table_xml
            else:
                tag = '</table_number>'
        else:
            tag = '</TABLE_NUMBER>'

        # Insert with proper formatting
        insert_pos = number_end + len(tag)
        new_table_xml = (
            table_xml[:insert_pos] +
            '\n' + file_path_tag +
            table_xml[insert_pos:]
        )

        return new_table_xml

    def extract_tables(self):
        """Extract and categorize all tables."""
        tables = self.parse_tables()

        for table_xml in tables:
            category, state_code = self.categorize_table(table_xml)

            # Add FILE_PATH metadata
            table_xml = self.add_file_path(table_xml, category, state_code)

            self.tables_by_category[category].append(table_xml)
            self.stats[f'category_{category}'] += 1

        print(f"\nCategorization complete:")
        print(f"  Core tables: {self.stats['category_core']}")
        for state_code in STATE_CODES:
            count = self.stats.get(f'category_state_{state_code}', 0)
            if count > 0:
                print(f"  {state_code} tables: {count}")

        unknown = self.stats.get('category_unknown', 0)
        if unknown > 0:
            print(f"  Unknown: {unknown}")

    def write_output_files(self):
        """Write extracted tables to output files."""
        print("\nWriting output files...")

        # Create output directory structure
        states_dir = self.output_dir / 'states'
        states_dir.mkdir(parents=True, exist_ok=True)

        # Write core tables
        if 'core' in self.tables_by_category:
            core_file = self.output_dir / 'TaxReturn_dt_core.xml'
            self._write_xml_file(core_file, self.tables_by_category['core'],
                               'Core tax calculation tables (federal, non-state-specific)')
            print(f"  ✓ {core_file}: {len(self.tables_by_category['core'])} tables")

        # Write state tables
        for state_code in STATE_CODES:
            category = f'state_{state_code}'
            if category in self.tables_by_category:
                state_file = states_dir / f'{state_code}_dt.xml'
                self._write_xml_file(
                    state_file,
                    self.tables_by_category[category],
                    f'State-specific decision tables for {state_code}'
                )
                print(f"  ✓ {state_file}: {len(self.tables_by_category[category])} tables")

        # Write unknown tables if any
        if 'unknown' in self.tables_by_category:
            unknown_file = self.output_dir / 'TaxReturn_dt_unknown.xml'
            self._write_xml_file(unknown_file, self.tables_by_category['unknown'],
                               'Tables that could not be categorized')
            print(f"  ⚠ {unknown_file}: {len(self.tables_by_category['unknown'])} tables")

    def _write_xml_file(self, filepath, tables, description):
        """Write a collection of tables to an XML file."""
        with open(filepath, 'w', encoding='utf-8') as f:
            # Write XML header
            f.write('<?xml version="1.0" encoding="UTF-8"?>\n')
            f.write('<!--\n')
            f.write(f'  {description}\n')
            f.write('\n')
            f.write('  AUTOMATICALLY GENERATED - DO NOT EDIT MANUALLY\n')
            f.write(f'  Generated from {self.input_file.name}\n')
            f.write('  Use extract_dt_tables.py to regenerate this file\n')
            f.write('-->\n')
            f.write('<decision_tables>\n\n')

            # Write tables
            for table_xml in tables:
                f.write(table_xml)
                f.write('\n\n')

            # Close root element
            f.write('</decision_tables>\n')

    def generate_report(self, report_file):
        """Generate detailed extraction report."""
        print(f"\nGenerating report: {report_file}")

        with open(report_file, 'w') as f:
            f.write("=" * 80 + "\n")
            f.write("Decision Table Extraction Report\n")
            f.write("=" * 80 + "\n\n")

            f.write(f"Input file: {self.input_file}\n")
            f.write(f"Output directory: {self.output_dir}\n")
            f.write(f"Total tables processed: {self.stats['total_tables']}\n\n")

            f.write("-" * 80 + "\n")
            f.write("Tables by Category\n")
            f.write("-" * 80 + "\n\n")

            # Core tables
            core_count = self.stats.get('category_core', 0)
            f.write(f"Core tables: {core_count}\n")
            f.write(f"  Output: TaxReturn_dt_core.xml\n\n")

            # State tables
            f.write("State-specific tables:\n")
            for state_code in STATE_CODES:
                count = self.stats.get(f'category_state_{state_code}', 0)
                if count > 0:
                    f.write(f"  {state_code}: {count} tables -> states/{state_code}_dt.xml\n")

            unknown = self.stats.get('category_unknown', 0)
            if unknown > 0:
                f.write(f"\nUnknown/uncategorized: {unknown} tables\n")
                f.write("  Output: TaxReturn_dt_unknown.xml\n")

            f.write("\n" + "=" * 80 + "\n")
            f.write("Directory Structure Created\n")
            f.write("=" * 80 + "\n\n")

            f.write("xml/\n")
            f.write("├── TaxReturn_dt_core.xml (core tables)\n")
            f.write("└── states/\n")

            for state_code in STATE_CODES:
                count = self.stats.get(f'category_state_{state_code}', 0)
                if count > 0:
                    f.write(f"    ├── {state_code}_dt.xml ({count} tables)\n")

            f.write("\n")

        print(f"✓ Report written to {report_file}")

def main():
    # Determine paths
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    xml_dir = project_root / 'xml'
    input_file = xml_dir / 'TaxReturn_dt.xml'
    output_dir = xml_dir
    report_file = Path('/tmp/extraction_report.txt')

    # Check input file exists
    if not input_file.exists():
        print(f"Error: Input file not found: {input_file}")
        sys.exit(1)

    # Create extractor and run
    extractor = DecisionTableExtractor(input_file, output_dir)

    print("=" * 80)
    print("Decision Table Extractor")
    print("=" * 80)
    print()

    extractor.extract_tables()
    extractor.write_output_files()
    extractor.generate_report(report_file)

    print("\n" + "=" * 80)
    print("Extraction Complete!")
    print("=" * 80)
    print(f"\nBackup: {input_file}.original")
    print(f"Report: {report_file}")
    print("\nNext steps:")
    print("  1. Review extraction report")
    print("  2. Verify generated files")
    print("  3. Test with: cd go && go test ./pkg/dtrules/...")
    print()

if __name__ == '__main__':
    main()
