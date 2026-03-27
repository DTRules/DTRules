#!/usr/bin/env python3
"""
Merge script for multi-file XML structure.

Combines separate decision table and entity definition files back into
monolithic TaxReturn_dt.xml and TaxReturn_edd.xml files for backward
compatibility and testing.

Usage:
    python3 scripts/merge_files.py [--output-dir OUTPUT_DIR]

Options:
    --output-dir DIR    Output directory for merged files (default: xml/)
    --dry-run          Show what would be merged without writing files
    --verbose          Show detailed merge information

The script:
1. Finds all *_dt.xml and *_edd.xml files (excluding monolithic files)
2. Parses and extracts decision tables / entities
3. Sorts by TABLE_NUMBER to maintain consistent ordering
4. Generates merged TaxReturn_dt.xml and TaxReturn_edd.xml files
"""

import sys
import os
import argparse
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import List, Dict
from collections import defaultdict


def find_xml_files(base_path: Path, pattern: str, exclude: List[str]) -> List[Path]:
    """Find all XML files matching pattern, excluding specified files."""
    files = []
    for xml_file in base_path.rglob(pattern):
        if xml_file.is_file() and xml_file.name not in exclude:
            files.append(xml_file)
    return sorted(files)


def extract_decision_tables(xml_files: List[Path], verbose: bool = False) -> List[tuple]:
    """
    Extract decision tables from XML files.
    Returns list of (table_number, xml_element, source_file) tuples.
    """
    tables = []

    for xml_file in xml_files:
        try:
            tree = ET.parse(xml_file)
            root = tree.getroot()

            for dt in root.findall('.//decision_table'):
                # Find TABLE_NUMBER
                table_num_elem = dt.find('.//TABLE_NUMBER')
                if table_num_elem is not None:
                    table_num = table_num_elem.text.strip()
                    # Convert to int for sorting, but keep original string
                    try:
                        sort_key = int(table_num)
                    except ValueError:
                        # Handle non-numeric table numbers (like templates)
                        sort_key = 999999
                    tables.append((sort_key, table_num, dt, xml_file))
                    if verbose:
                        table_name = dt.find('table_name')
                        name = table_name.text if table_name is not None else "unknown"
                        print(f"  Found table {table_num} ({name}) in {xml_file.name}")
                else:
                    print(f"WARNING: decision_table without TABLE_NUMBER in {xml_file.name}", file=sys.stderr)

        except ET.ParseError as e:
            print(f"ERROR: XML parse error in {xml_file}: {e}", file=sys.stderr)
        except Exception as e:
            print(f"ERROR: Failed to process {xml_file}: {e}", file=sys.stderr)

    return tables


def extract_entities(xml_files: List[Path], verbose: bool = False) -> List[tuple]:
    """
    Extract entity definitions from EDD XML files.
    Returns list of (entity_name, xml_element, source_file) tuples.
    """
    entities = []
    seen_entities = set()

    for xml_file in xml_files:
        try:
            tree = ET.parse(xml_file)
            root = tree.getroot()

            for entity in root.findall('.//entity'):
                entity_name = entity.get('name')
                if entity_name:
                    if entity_name in seen_entities:
                        print(f"WARNING: Duplicate entity '{entity_name}' in {xml_file.name}", file=sys.stderr)
                    seen_entities.add(entity_name)
                    entities.append((entity_name, entity, xml_file))
                    if verbose:
                        field_count = len(entity.findall('field'))
                        print(f"  Found entity '{entity_name}' ({field_count} fields) in {xml_file.name}")

        except ET.ParseError as e:
            print(f"ERROR: XML parse error in {xml_file}: {e}", file=sys.stderr)
        except Exception as e:
            print(f"ERROR: Failed to process {xml_file}: {e}", file=sys.stderr)

    return entities


def create_merged_dt_xml(tables: List[tuple], output_file: Path, verbose: bool = False) -> None:
    """Create merged decision_tables XML file."""
    # Sort by table number
    tables.sort(key=lambda x: x[0])

    # Create root element
    root = ET.Element('decision_tables')
    root.text = "\n\n"

    # Add each table
    for sort_key, table_num, dt_elem, source_file in tables:
        # Add newline and comment before each table
        comment_text = f" Table {table_num} from {source_file.name} "
        comment = ET.Comment(comment_text)
        comment.tail = "\n"
        root.append(comment)

        # Clone the decision_table element
        dt_elem.tail = "\n\n"
        root.append(dt_elem)

    # Create tree and write
    tree = ET.ElementTree(root)
    ET.indent(tree, space="", level=0)

    # Write with XML declaration
    with open(output_file, 'wb') as f:
        tree.write(f, encoding='UTF-8', xml_declaration=True)

    if verbose:
        print(f"Wrote {len(tables)} tables to {output_file}")


def create_merged_edd_xml(entities: List[tuple], output_file: Path, verbose: bool = False) -> None:
    """Create merged entity_data_dictionary XML file."""
    # Sort entities alphabetically
    entities.sort(key=lambda x: x[0])

    # Create root element
    root = ET.Element('entity_data_dictionary')
    root.set('version', '2')
    root.set('xmlns:xs', 'http://www.w3.org/2001/XMLSchema')
    root.text = "\n\n"

    # Add each entity
    for entity_name, entity_elem, source_file in entities:
        # Add comment before each entity
        comment_text = f" Entity '{entity_name}' from {source_file.name} "
        comment = ET.Comment(comment_text)
        comment.tail = "\n"
        root.append(comment)

        # Clone the entity element
        entity_elem.tail = "\n\n"
        root.append(entity_elem)

    # Create tree and write
    tree = ET.ElementTree(root)
    ET.indent(tree, space="\t", level=0)

    # Write with XML declaration
    with open(output_file, 'wb') as f:
        tree.write(f, encoding='UTF-8', xml_declaration=True)

    if verbose:
        print(f"Wrote {len(entities)} entities to {output_file}")


def merge_decision_tables(xml_dir: Path, output_dir: Path, dry_run: bool = False, verbose: bool = False) -> None:
    """Merge all decision table files into TaxReturn_dt.xml."""
    print("=== Merging Decision Tables ===")

    # Find all DT files except the monolithic one
    dt_files = find_xml_files(xml_dir, "*_dt.xml", exclude=["TaxReturn_dt.xml"])
    print(f"Found {len(dt_files)} decision table files to merge")

    if verbose:
        for f in dt_files:
            print(f"  - {f.relative_to(xml_dir)}")

    # Extract tables
    tables = extract_decision_tables(dt_files, verbose)
    print(f"Extracted {len(tables)} decision tables")

    # Check for duplicates
    table_nums = [t[1] for t in tables]
    duplicates = [num for num in table_nums if table_nums.count(num) > 1]
    if duplicates:
        print(f"ERROR: Duplicate table numbers found: {set(duplicates)}", file=sys.stderr)
        sys.exit(1)

    # Write merged file
    output_file = output_dir / "TaxReturn_dt.xml"
    if dry_run:
        print(f"[DRY RUN] Would write {len(tables)} tables to {output_file}")
    else:
        create_merged_dt_xml(tables, output_file, verbose)
        print(f"✅ Created {output_file}")


def merge_entity_definitions(xml_dir: Path, output_dir: Path, dry_run: bool = False, verbose: bool = False) -> None:
    """Merge all entity definition files into TaxReturn_edd.xml."""
    print("\n=== Merging Entity Definitions ===")

    # Find all EDD files except the monolithic one
    edd_files = find_xml_files(xml_dir, "*_edd.xml", exclude=["TaxReturn_edd.xml"])
    print(f"Found {len(edd_files)} entity definition files to merge")

    if verbose:
        for f in edd_files:
            print(f"  - {f.relative_to(xml_dir)}")

    # Extract entities
    entities = extract_entities(edd_files, verbose)
    print(f"Extracted {len(entities)} entity definitions")

    # Write merged file
    output_file = output_dir / "TaxReturn_edd.xml"
    if dry_run:
        print(f"[DRY RUN] Would write {len(entities)} entities to {output_file}")
    else:
        create_merged_edd_xml(entities, output_file, verbose)
        print(f"✅ Created {output_file}")


def main():
    parser = argparse.ArgumentParser(
        description="Merge multi-file XML structure into monolithic files"
    )
    parser.add_argument(
        '--output-dir',
        type=Path,
        help='Output directory for merged files (default: xml/)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be merged without writing files'
    )
    parser.add_argument(
        '--verbose',
        action='store_true',
        help='Show detailed merge information'
    )

    args = parser.parse_args()

    # Determine directories
    script_dir = Path(__file__).parent
    project_dir = script_dir.parent
    xml_dir = project_dir / "xml"
    output_dir = args.output_dir if args.output_dir else xml_dir

    if not xml_dir.exists():
        print(f"ERROR: XML directory not found: {xml_dir}", file=sys.stderr)
        sys.exit(1)

    if not output_dir.exists():
        output_dir.mkdir(parents=True, exist_ok=True)

    print(f"XML directory: {xml_dir}")
    print(f"Output directory: {output_dir}")
    print()

    try:
        # Merge decision tables
        merge_decision_tables(xml_dir, output_dir, args.dry_run, args.verbose)

        # Merge entity definitions
        merge_entity_definitions(xml_dir, output_dir, args.dry_run, args.verbose)

        print("\n✅ Merge completed successfully")

    except Exception as e:
        print(f"\n❌ Merge failed: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
