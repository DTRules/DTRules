#!/usr/bin/env python3
"""
Fix EL descriptions in DTRules decision table XML files.

Strategy:
1. If description has "(comment)" suffix, extract table name, add "perform" prefix
2. If description is a pure comment, move to comment field, clear description
3. Postfix is authoritative - description is for human readability

Usage:
    python fix_el_descriptions.py <xml_file_or_directory> [--dry-run]
"""

import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

# Pattern: TableName (comment) or TableName (comment with stuff)
TABLE_WITH_COMMENT = re.compile(r'^([A-Z][A-Za-z0-9_]+)\s*\((.+)\)\s*$')

# Pattern: Looks like a table name (CamelCase starting with capital)
TABLE_NAME = re.compile(r'^[A-Z][A-Za-z0-9_]+$')

def is_comment_not_el(desc: str, element_type: str) -> bool:
    """Check if description is a comment rather than EL."""
    lower = desc.lower()

    # "otherwise" is VALID for conditions (star conditions require it)
    if lower == 'otherwise' and element_type == 'condition':
        return False

    # Contains % (percentages in comments) or unicode comparison operators
    if '%' in desc or '≤' in desc or '≥' in desc:
        return True

    # Ends with parenthetical like (MFJ) or (max $300)
    if re.search(r'\([^)]+\)\s*$', desc):
        return True

    # Is a question
    if desc.endswith('?') or lower.startswith('is ') or lower.startswith('are '):
        return True

    # Multiple statements separated by comma (not EL)
    if ', set ' in lower or ', add ' in lower:
        return True

    # Common comment patterns
    comment_patterns = [
        'calculate ', 'computed', 'per irc', 'per pub',
        'up to', 'at least', 'at most', 'threshold',
        'brackets', 'exemptions', 'deductions and',
        'log ', 'skip ', 'does not', ' needed',
        'not applicable', 'no ', 'sum of all',
        'for all ', 'based on', ' rate', ' ceiling',
        'execute', 'construct', 'prorate', 'allocate',
        'process each', ' or ', 'lesser of', 'greater of',
        'maximum of', 'minimum of', 'phased out',
        'applies', 'qualifies', 'qualifying', 'qualified',
        'summary', 'showing', 'pass message', 'mismatch',
        'credit summary', 'increment', 'reduction',
        'otherwise',  # "otherwise" alone as action is comment
        'exists in', 'tax at ', 'taxed at', 'all component',
        'for each', 'flag to', 'excluded for', 'limited to',
        'fully ', 'exclusion', ' subtractions', 'appropriate',
        'and <=', 'and >=',  # double comparison
        'entity exists', 'credits greater', 'liability limitation',
        'kiddie tax to', 'greater than 0',
    ]

    for pattern in comment_patterns:
        if pattern in lower:
            return True

    return False

def fix_description(desc: str, element_type: str) -> tuple:
    """
    Fix a description. Returns (new_desc, comment_to_add).
    """
    if not desc or not desc.strip():
        return ("", None)

    desc = desc.strip()

    # Pattern: TableName (comment)
    match = TABLE_WITH_COMMENT.match(desc)
    if match:
        table_name = match.group(1)
        comment = match.group(2).strip()
        return (f"perform {table_name}", comment)

    # Pattern: Just a table name without "perform"
    if TABLE_NAME.match(desc) and element_type == "action":
        return (f"perform {desc}", None)

    # Pattern: Pure comment - clear description, move to comment
    if is_comment_not_el(desc, element_type):
        return ("", desc)

    # Otherwise return as-is (let EL compiler validate)
    return (desc, None)

def process_element(elem, desc_tag: str, comment_tag: str, element_type: str, stats: dict) -> bool:
    """Process a condition or action element. Returns True if modified."""
    desc_elem = elem.find(desc_tag)
    comment_elem = elem.find(comment_tag)

    if desc_elem is None or not desc_elem.text:
        return False

    old_desc = desc_elem.text.strip()
    if not old_desc:
        return False

    stats['processed'] += 1

    new_desc, new_comment = fix_description(old_desc, element_type)

    modified = False

    if new_desc != old_desc:
        desc_elem.text = new_desc
        stats['fixed'] += 1
        print(f"    FIX: '{old_desc[:50]}' -> '{new_desc}'")
        modified = True

    if new_comment:
        if comment_elem is None:
            comment_elem = ET.SubElement(elem, comment_tag)
            comment_elem.text = new_comment
        elif comment_elem.text:
            comment_elem.text = f"{comment_elem.text}; {new_comment}"
        else:
            comment_elem.text = new_comment
        stats['comments_added'] += 1
        modified = True

    return modified

def process_xml_file(filepath: Path, stats: dict, dry_run: bool = False) -> bool:
    """Process a single XML file. Returns True if file was modified."""
    print(f"\nProcessing: {filepath}")

    try:
        tree = ET.parse(filepath)
        root = tree.getroot()
    except ET.ParseError as e:
        print(f"  XML parse error: {e}", file=sys.stderr)
        return False

    modified = False

    for dt in root.iter('decision_table'):
        table_name = dt.find('table_name')
        if table_name is not None and table_name.text:
            print(f"  Table: {table_name.text}")

        for cond in dt.iter('condition_details'):
            if process_element(cond, 'condition_description', 'condition_comment', 'condition', stats):
                modified = True

        for action in dt.iter('action_details'):
            if process_element(action, 'action_description', 'action_comment', 'action', stats):
                modified = True

        for init in dt.iter('initial_action'):
            if process_element(init, 'action_description', 'action_comment', 'action', stats):
                modified = True

    if modified and not dry_run:
        tree.write(filepath, encoding='UTF-8', xml_declaration=True)
        print(f"  Saved changes to {filepath}")

    return modified

def find_xml_files(path: Path) -> list:
    """Find all decision table XML files."""
    if path.is_file():
        return [path] if path.suffix == '.xml' else []

    xml_files = []
    for f in path.rglob('*_dt.xml'):
        xml_files.append(f)
    return sorted(xml_files)

def main():
    if len(sys.argv) < 2:
        print("Usage: fix_el_descriptions.py <xml_file_or_directory> [--dry-run]")
        sys.exit(1)

    path = Path(sys.argv[1])
    dry_run = '--dry-run' in sys.argv

    if not path.exists():
        print(f"Error: {path} does not exist")
        sys.exit(1)

    xml_files = find_xml_files(path)

    if not xml_files:
        print(f"No *_dt.xml files found in {path}")
        sys.exit(1)

    print(f"Found {len(xml_files)} XML files to process")
    if dry_run:
        print("DRY RUN - no changes will be saved")

    stats = {
        'processed': 0,
        'fixed': 0,
        'comments_added': 0,
        'files_modified': 0
    }

    for xml_file in xml_files:
        if process_xml_file(xml_file, stats, dry_run):
            stats['files_modified'] += 1

    print(f"\n{'='*50}")
    print(f"Summary:")
    print(f"  Files processed: {len(xml_files)}")
    print(f"  Files modified: {stats['files_modified']}")
    print(f"  Elements processed: {stats['processed']}")
    print(f"  Descriptions fixed: {stats['fixed']}")
    print(f"  Comments extracted: {stats['comments_added']}")

if __name__ == '__main__':
    main()
