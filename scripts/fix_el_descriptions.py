#!/usr/bin/env python3
"""
Fix EL descriptions in DTRules decision table XML files.

Strategy: The postfix (DSL) field is authoritative. Description fields that
contain human-readable comments (not valid EL) should be cleared, with the
text optionally moved to the comment field.

Usage:
    python fix_el_descriptions.py <xml_file_or_directory> [--dry-run]
"""

import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Optional, Tuple

# Simple EL patterns that are valid
EL_PATTERNS = [
    # set X = Y
    r'^set\s+[\w.]+\s*=\s*.+$',
    # perform TableName
    r'^perform\s+\w+$',
    # X == Y, X != Y, X > Y, etc.
    r'^[\w.]+\s*(==|!=|>=|<=|>|<)\s*.+$',
    # X and Y, X or Y
    r'^.+\s+(and|or)\s+.+$',
    # not X
    r'^not\s+.+$',
    # add X to Y
    r'^add\s+.+\s+to\s+.+$',
    # Simple entity reference
    r'^[\w]+\.[\w.]+$',
    # Boolean literals
    r'^(true|false)$',
]

def is_valid_el(description: str) -> bool:
    """Check if a description looks like valid EL syntax."""
    if not description or not description.strip():
        return True  # Empty is valid (will use postfix)

    desc = description.strip()

    # Check against known EL patterns
    for pattern in EL_PATTERNS:
        if re.match(pattern, desc, re.IGNORECASE):
            return True

    return False

def looks_like_comment(description: str) -> bool:
    """Check if description looks like a human comment rather than EL."""
    if not description:
        return False

    desc = description.strip().lower()

    # Common comment patterns
    comment_indicators = [
        'is equal to',
        'is greater than',
        'is less than',
        'includes the',
        'add a new',
        'add the',
        'calculate',
        'evaluate',
        'otherwise',
        'when called',
        'perform when',
        'the client',
        'the result',
        'set the',
        'applies',
        '\'s ',  # possessive
    ]

    for indicator in comment_indicators:
        if indicator in desc:
            return True

    return False

def is_simple_value(token: str) -> bool:
    """Check if a token looks like a simple value (entity.attr, number, bool, string)."""
    # Entity reference
    if re.match(r'^[\w]+\.[\w.]+$', token):
        return True
    # Number
    if re.match(r'^-?\d+(\.\d+)?$', token):
        return True
    # Boolean
    if token.lower() in ('true', 'false'):
        return True
    # Quoted string
    if token.startswith('"') and token.endswith('"'):
        return True
    return False

def postfix_to_simple_el(postfix: str) -> Optional[str]:
    """Try to convert simple postfix patterns to EL. Very conservative."""
    if not postfix or not postfix.strip():
        return None

    # Skip complex expressions with code blocks, @, local, etc.
    if any(x in postfix for x in ['{', 'local@', 'cvi', 'cvs', 'forall', 'if']):
        return None

    tokens = postfix.strip().split()
    if not tokens:
        return None

    # Pattern: TableName performaliases -> perform TableName
    if len(tokens) == 2 and tokens[1] in ('performaliases', 'perform'):
        if re.match(r'^[A-Z]\w+$', tokens[0]):  # Table names are CamelCase
            return f"perform {tokens[0]}"

    # Pattern: simple_value /entity.attr xdef -> set entity.attr = simple_value
    if len(tokens) == 3 and tokens[2] == 'xdef' and tokens[1].startswith('/'):
        attr = tokens[1][1:]
        value = tokens[0]
        if is_simple_value(value) and re.match(r'^[\w.]+$', attr):
            return f"set {attr} = {value}"

    # Pattern: entity.attr value op -> entity.attr op value (simple comparison)
    binary_ops = {'gt': '>', 'ge': '>=', 'lt': '<', 'le': '<=', 'eq': '==', 'ne': '!='}
    if len(tokens) == 3 and tokens[2] in binary_ops:
        if is_simple_value(tokens[0]) and is_simple_value(tokens[1]):
            return f"{tokens[0]} {binary_ops[tokens[2]]} {tokens[1]}"

    return None

def process_element(elem, desc_tag: str, postfix_tag: str, comment_tag: str, stats: dict) -> bool:
    """Process a condition or action element. Returns True if modified."""
    desc_elem = elem.find(desc_tag)
    postfix_elem = elem.find(postfix_tag)
    comment_elem = elem.find(comment_tag)

    if postfix_elem is None:
        return False

    postfix = postfix_elem.text.strip() if postfix_elem.text else ""
    description = desc_elem.text.strip() if desc_elem is not None and desc_elem.text else ""

    if not postfix and not description:
        return False

    stats['processed'] += 1

    # If description is already valid EL, skip
    if is_valid_el(description):
        return False

    # Try to convert postfix to EL
    new_el = postfix_to_simple_el(postfix)

    modified = False
    old_desc = description[:50] + "..." if len(description) > 50 else description

    if new_el:
        # We have a valid EL conversion
        if desc_elem is None:
            desc_elem = ET.SubElement(elem, desc_tag)
        desc_elem.text = new_el
        stats['converted'] += 1
        print(f"    CONVERT: '{old_desc}' -> '{new_el}'")
        modified = True
    else:
        # Can't convert - clear description, move to comment if needed
        if description and looks_like_comment(description):
            # Move description to comment
            if comment_elem is None:
                comment_elem = ET.SubElement(elem, comment_tag)
                comment_elem.text = description
            elif not comment_elem.text:
                comment_elem.text = description
            else:
                # Append to existing comment
                comment_elem.text = f"{comment_elem.text}; {description}"
            print(f"    MOVE TO COMMENT: '{old_desc}'")
            stats['moved'] += 1

        # Clear description
        if desc_elem is not None and description:
            desc_elem.text = ""
            print(f"    CLEAR: '{old_desc}'")
            stats['cleared'] += 1
            modified = True

    return modified

def process_condition(cond_elem, stats: dict) -> bool:
    """Process a condition element."""
    return process_element(
        cond_elem,
        'condition_description',
        'condition_postfix',
        'condition_comment',
        stats
    )

def process_action(action_elem, stats: dict) -> bool:
    """Process an action element."""
    return process_element(
        action_elem,
        'action_description',
        'action_postfix',
        'action_comment',
        stats
    )

def process_initial_action(init_elem, stats: dict) -> bool:
    """Process an initial_action element."""
    return process_element(
        init_elem,
        'action_description',
        'action_postfix',
        'action_comment',
        stats
    )

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
            if process_condition(cond, stats):
                modified = True

        for action in dt.iter('action_details'):
            if process_action(action, stats):
                modified = True

        for init in dt.iter('initial_action'):
            if process_initial_action(init, stats):
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
        'converted': 0,
        'cleared': 0,
        'moved': 0,
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
    print(f"  Converted to EL: {stats['converted']}")
    print(f"  Cleared (invalid): {stats['cleared']}")
    print(f"  Moved to comment: {stats['moved']}")

if __name__ == '__main__':
    main()
