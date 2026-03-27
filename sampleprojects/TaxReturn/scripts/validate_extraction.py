#!/usr/bin/env python3
"""
Validation script for multi-file XML extraction.

Validates that decision tables and entity definitions have been correctly
extracted from monolithic TaxReturn_dt.xml and TaxReturn_edd.xml files
into separate per-state and per-function files.

Usage:
    python3 scripts/validate_extraction.py

Exit codes:
    0 - All validations passed
    1 - Validation errors found
"""

import sys
import os
import xml.etree.ElementTree as ET
from pathlib import Path
from collections import defaultdict
from typing import Dict, List, Set, Tuple

class ValidationResult:
    def __init__(self):
        self.errors: List[str] = []
        self.warnings: List[str] = []
        self.info: List[str] = []

    def add_error(self, msg: str):
        self.errors.append(f"ERROR: {msg}")

    def add_warning(self, msg: str):
        self.warnings.append(f"WARNING: {msg}")

    def add_info(self, msg: str):
        self.info.append(f"INFO: {msg}")

    def has_errors(self) -> bool:
        return len(self.errors) > 0

    def print_results(self):
        for msg in self.info:
            print(msg)
        for msg in self.warnings:
            print(msg)
        for msg in self.errors:
            print(msg, file=sys.stderr)

        if self.has_errors():
            print(f"\n❌ Validation FAILED with {len(self.errors)} errors", file=sys.stderr)
        else:
            print(f"\n✅ Validation PASSED")

        if self.warnings:
            print(f"⚠️  {len(self.warnings)} warnings found")


def find_xml_files(base_path: Path, pattern: str) -> List[Path]:
    """Find all XML files matching pattern in directory tree."""
    files = []
    for xml_file in base_path.rglob(pattern):
        if xml_file.is_file():
            files.append(xml_file)
    return sorted(files)


def parse_decision_tables(xml_file: Path) -> Dict[str, Dict]:
    """Parse decision tables from XML file and return dict keyed by TABLE_NUMBER."""
    tables = {}

    try:
        tree = ET.parse(xml_file)
        root = tree.getroot()

        for dt in root.findall('.//decision_table'):
            table_name_elem = dt.find('table_name')
            table_num_elem = dt.find('.//TABLE_NUMBER')
            file_path_elem = dt.find('.//FILE_PATH')

            if table_name_elem is not None and table_num_elem is not None:
                table_name = table_name_elem.text.strip()
                table_num = table_num_elem.text.strip()
                file_path = file_path_elem.text.strip() if file_path_elem is not None else None

                tables[table_num] = {
                    'name': table_name,
                    'number': table_num,
                    'file_path': file_path,
                    'source_file': xml_file
                }
    except ET.ParseError as e:
        raise ValueError(f"XML parse error in {xml_file}: {e}")
    except Exception as e:
        raise ValueError(f"Error processing {xml_file}: {e}")

    return tables


def parse_entities(xml_file: Path) -> Dict[str, Dict]:
    """Parse entity definitions from EDD XML file."""
    entities = {}

    try:
        tree = ET.parse(xml_file)
        root = tree.getroot()

        for entity in root.findall('.//entity'):
            name_attr = entity.get('name')
            if name_attr:
                field_count = len(entity.findall('field'))
                entities[name_attr] = {
                    'name': name_attr,
                    'field_count': field_count,
                    'source_file': xml_file
                }
    except ET.ParseError as e:
        raise ValueError(f"XML parse error in {xml_file}: {e}")
    except Exception as e:
        raise ValueError(f"Error processing {xml_file}: {e}")

    return entities


def validate_decision_tables(xml_dir: Path, result: ValidationResult) -> None:
    """Validate decision table extraction."""
    result.add_info("=== Validating Decision Tables ===")

    # Parse monolithic file
    monolithic_file = xml_dir / "TaxReturn_dt.xml"
    if not monolithic_file.exists():
        result.add_error(f"Monolithic file not found: {monolithic_file}")
        return

    result.add_info(f"Loading monolithic file: {monolithic_file}")
    try:
        monolithic_tables = parse_decision_tables(monolithic_file)
        result.add_info(f"Found {len(monolithic_tables)} tables in monolithic file")
    except ValueError as e:
        result.add_error(str(e))
        result.add_info("Skipping monolithic comparison due to parse error")
        monolithic_tables = {}

    # Parse all extracted files
    extracted_files = []
    extracted_files.extend(find_xml_files(xml_dir, "*_dt.xml"))
    extracted_files = [f for f in extracted_files if f.name != "TaxReturn_dt.xml"]

    result.add_info(f"Found {len(extracted_files)} extracted DT files")

    all_extracted_tables = {}
    for xml_file in extracted_files:
        try:
            tables = parse_decision_tables(xml_file)
            result.add_info(f"  {xml_file.name}: {len(tables)} tables")
            all_extracted_tables.update(tables)
        except ValueError as e:
            result.add_error(str(e))

    result.add_info(f"Total extracted tables: {len(all_extracted_tables)}")

    # Compare table numbers (only if monolithic file was parseable)
    if monolithic_tables:
        monolithic_nums = set(monolithic_tables.keys())
        extracted_nums = set(all_extracted_tables.keys())

        missing = monolithic_nums - extracted_nums
        extra = extracted_nums - monolithic_nums

        if missing:
            result.add_error(f"Missing {len(missing)} tables in extracted files: {sorted(missing)}")

        if extra:
            result.add_warning(f"Found {len(extra)} tables in extracted files not in monolithic: {sorted(extra)}")

    # Check for duplicates
    table_sources = defaultdict(list)
    for table_num, info in all_extracted_tables.items():
        table_sources[table_num].append(info['source_file'].name)

    for table_num, sources in table_sources.items():
        if len(sources) > 1:
            result.add_error(f"Table {table_num} appears in multiple files: {sources}")

    # Validate FILE_PATH consistency
    result.add_info("\n=== Validating FILE_PATH Consistency ===")
    for table_num, info in all_extracted_tables.items():
        file_path = info.get('file_path')
        source_file = info['source_file']

        if file_path:
            # Check if state file FILE_PATH matches directory structure
            if source_file.parent.name == 'states':
                state_code = source_file.name.split('_')[0]
                expected_prefix = f"states/{state_code}/"
                if not file_path.startswith(expected_prefix):
                    result.add_warning(
                        f"Table {table_num} in {source_file.name} has FILE_PATH '{file_path}' "
                        f"but expected prefix '{expected_prefix}'"
                    )


def validate_entity_definitions(xml_dir: Path, result: ValidationResult) -> None:
    """Validate entity definition extraction."""
    result.add_info("\n=== Validating Entity Definitions ===")

    # Parse monolithic file
    monolithic_file = xml_dir / "TaxReturn_edd.xml"
    if not monolithic_file.exists():
        result.add_error(f"Monolithic file not found: {monolithic_file}")
        return

    result.add_info(f"Loading monolithic file: {monolithic_file}")
    try:
        monolithic_entities = parse_entities(monolithic_file)
        result.add_info(f"Found {len(monolithic_entities)} entities in monolithic file")
    except ValueError as e:
        result.add_error(str(e))
        result.add_info("Skipping monolithic comparison due to parse error")
        monolithic_entities = {}

    # Parse all extracted files
    extracted_files = []
    extracted_files.extend(find_xml_files(xml_dir, "*_edd.xml"))
    extracted_files = [f for f in extracted_files if f.name != "TaxReturn_edd.xml"]

    result.add_info(f"Found {len(extracted_files)} extracted EDD files")

    all_extracted_entities = {}
    for xml_file in extracted_files:
        try:
            entities = parse_entities(xml_file)
            result.add_info(f"  {xml_file.name}: {len(entities)} entities")
            all_extracted_entities.update(entities)
        except ValueError as e:
            result.add_error(str(e))

    result.add_info(f"Total extracted entities: {len(all_extracted_entities)}")

    # Compare entity names (only if monolithic file was parseable)
    if monolithic_entities:
        monolithic_names = set(monolithic_entities.keys())
        extracted_names = set(all_extracted_entities.keys())

        missing = monolithic_names - extracted_names
        extra = extracted_names - monolithic_names

        if missing:
            result.add_error(f"Missing {len(missing)} entities in extracted files: {sorted(missing)}")

        if extra:
            result.add_warning(f"Found {len(extra)} entities in extracted files not in monolithic: {sorted(extra)}")

        # Check for entity field count differences (might indicate incomplete extraction)
        for entity_name in monolithic_names & extracted_names:
            mono_fields = monolithic_entities[entity_name]['field_count']
            ext_fields = all_extracted_entities[entity_name]['field_count']
            if mono_fields != ext_fields:
                result.add_warning(
                    f"Entity '{entity_name}' has {mono_fields} fields in monolithic "
                    f"but {ext_fields} fields in extracted file "
                    f"{all_extracted_entities[entity_name]['source_file'].name}"
                )


def validate_xml_well_formed(xml_dir: Path, result: ValidationResult) -> None:
    """Validate that all XML files are well-formed."""
    result.add_info("\n=== Validating XML Well-Formedness ===")

    all_xml_files = find_xml_files(xml_dir, "*.xml")

    for xml_file in all_xml_files:
        try:
            ET.parse(xml_file)
        except ET.ParseError as e:
            result.add_error(f"XML parse error in {xml_file.name}: {e}")
        except Exception as e:
            result.add_error(f"Error reading {xml_file.name}: {e}")

    result.add_info(f"Checked {len(all_xml_files)} XML files for well-formedness")


def main():
    # Determine project root
    script_dir = Path(__file__).parent
    project_dir = script_dir.parent
    xml_dir = project_dir / "xml"

    if not xml_dir.exists():
        print(f"ERROR: XML directory not found: {xml_dir}", file=sys.stderr)
        sys.exit(1)

    result = ValidationResult()

    try:
        # Run all validations
        validate_xml_well_formed(xml_dir, result)
        validate_decision_tables(xml_dir, result)
        validate_entity_definitions(xml_dir, result)

    except Exception as e:
        result.add_error(f"Validation failed with exception: {e}")
        import traceback
        traceback.print_exc()

    # Print results
    result.print_results()

    # Exit with appropriate code
    sys.exit(1 if result.has_errors() else 0)


if __name__ == "__main__":
    main()
