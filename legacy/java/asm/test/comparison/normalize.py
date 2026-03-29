#!/usr/bin/env python3
"""
Normalize output for comparison between Go and Assembly implementations.

This script handles known differences between implementations to allow
meaningful comparison of semantic output while ignoring formatting differences.

Usage:
    echo "output" | python3 normalize.py
    python3 normalize.py < file.txt
"""

import re
import sys


def normalize(text: str) -> str:
    """
    Normalize text output for cross-implementation comparison.

    Handles:
    - Timestamp removal
    - Floating point precision normalization
    - Whitespace normalization
    - Error message canonicalization
    """
    # Remove any ANSI color codes
    text = re.sub(r'\x1b\[[0-9;]*m', '', text)

    # Remove timestamps (various formats)
    # ISO format: 2024-01-15T10:30:45
    text = re.sub(r'\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?', '<TIMESTAMP>', text)
    # Unix timestamp
    text = re.sub(r'\b\d{10,13}\b', '<TIMESTAMP>', text)

    # Normalize floating point precision (keep 6 decimal places)
    def normalize_float(match):
        num = match.group(0)
        try:
            f = float(num)
            # Format with 6 decimal places
            return f'{f:.6f}'
        except ValueError:
            return num

    # Match floating point numbers
    text = re.sub(r'-?\d+\.\d+', normalize_float, text)

    # Normalize error messages to canonical form
    error_mappings = {
        r'stack underflow at op \w+': 'ERR_STACK_UNDERFLOW',
        r'stack underflow': 'ERR_STACK_UNDERFLOW',
        r'expected integer, got \w+': 'ERR_TYPE_MISMATCH',
        r'type mismatch': 'ERR_TYPE_MISMATCH',
        r'division by zero': 'ERR_DIV_ZERO',
        r'divide by zero': 'ERR_DIV_ZERO',
        r'out of memory': 'ERR_OUT_OF_MEMORY',
        r'memory allocation failed': 'ERR_OUT_OF_MEMORY',
        r'invalid opcode': 'ERR_INVALID_OPCODE',
        r'unknown opcode': 'ERR_INVALID_OPCODE',
        r'index out of bounds': 'ERR_INDEX_BOUNDS',
        r'array index out of range': 'ERR_INDEX_BOUNDS',
        r'name not found': 'ERR_NAME_NOT_FOUND',
        r'undefined name': 'ERR_NAME_NOT_FOUND',
        r'attribute not found': 'ERR_ATTR_NOT_FOUND',
        r'no such attribute': 'ERR_ATTR_NOT_FOUND',
    }

    for pattern, replacement in error_mappings.items():
        text = re.sub(pattern, replacement, text, flags=re.IGNORECASE)

    # Normalize line endings
    text = text.replace('\r\n', '\n')

    # Strip trailing whitespace from each line
    lines = [line.rstrip() for line in text.split('\n')]

    # Remove empty lines at start and end
    while lines and not lines[0]:
        lines.pop(0)
    while lines and not lines[-1]:
        lines.pop()

    return '\n'.join(lines)


def main():
    """Read from stdin and output normalized text."""
    input_text = sys.stdin.read()
    normalized = normalize(input_text)
    print(normalized)


if __name__ == '__main__':
    main()
