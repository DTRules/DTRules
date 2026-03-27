#!/usr/bin/env python3
"""
Extract EDD entities from monolithic TaxReturn_edd.xml into multi-file structure.

This script parses the monolithic EDD file and splits it into categorized files
based on entity purpose and type.
"""

import xml.etree.ElementTree as ET
import os
import sys
from collections import defaultdict
from datetime import datetime

# Define entity categories based on xls_file attribute and entity name
ENTITY_CATEGORIES = {
    'core': [
        'job', 'taxpayer', 'dependent', 'state_tax_result', 'state_period'
    ],
    'income': [
        'income', 'k1_income', 'farm_income', 'foreign_income',
        'foreign_earned_income', 'unreported_tips'
    ],
    'property': [
        'property', 'vehicle', 'property_sale', 'installment_sale',
        'like_kind_exchange', 'casualty_loss'
    ],
    'credits': [
        'credit', 'energy_improvement', 'elderly_disabled', 'clean_vehicle',
        'adoption', 'mortgage_credit_certificate', 'prior_year_amt'
    ],
    'deductions': [
        'deduction', 'noncash_contribution'
    ],
    'forms': [
        'ira_account', 'early_distribution', 'child_unearned_income',
        'estimated_payment', 'household_employee', 'nol_carryover',
        'partnership_audit', 'lihtc_recapture'
    ],
    'result': [
        'result'  # Special category - contains all constants and computed values
    ],
    'constants': [
        'constants'  # Federal tax constants and rates
    ]
}


def categorize_entity(entity_name):
    """Determine the category for an entity."""
    for category, entities in ENTITY_CATEGORIES.items():
        if entity_name in entities:
            return category
    # Default to 'other' if not categorized
    return 'other'


def parse_edd_file(filepath):
    """Parse the EDD XML file and extract entities."""
    try:
        tree = ET.parse(filepath)
        root = tree.getroot()

        entities_by_category = defaultdict(list)

        # Find all entity elements
        for entity in root.findall('entity'):
            entity_name = entity.get('name')
            if entity_name:
                category = categorize_entity(entity_name)
                entities_by_category[category].append(entity)
                print(f"  Found entity '{entity_name}' -> category '{category}'")

        return entities_by_category
    except Exception as e:
        print(f"ERROR parsing {filepath}: {e}")
        sys.exit(1)


def create_edd_xml(category, entities, output_dir):
    """Create an EDD XML file for a category with its entities."""

    # Category-specific numbering (following the pattern seen in state files)
    category_numbers = {
        'core': '01000',
        'income': '02000',
        'property': '03000',
        'deductions': '04000',
        'credits': '05000',
        'forms': '06000',
        'result': '07000',
        'constants': '08000',
        'other': '09000'
    }

    file_number = category_numbers.get(category, '99000')
    file_path_metadata = f"core/{file_number}_{category}_entities"

    # Create the output filename
    output_file = os.path.join(output_dir, f"TaxReturn_edd_{category}.xml")

    # Build the XML structure
    # Note: ElementTree handles xmlns differently - we'll set it directly in the output
    root = ET.Element('entity_data_dictionary')
    root.set('version', '2')

    # Add header comment
    comment_text = f"""
  Auto-generated file - extracted from TaxReturn_edd.xml
  Category: {category}
  Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

  This file contains {len(entities)} entit{'y' if len(entities) == 1 else 'ies'} categorized as '{category}'.
"""
    root.insert(0, ET.Comment(comment_text))

    # Add file_metadata with file_path
    file_metadata = ET.SubElement(root, 'file_metadata')
    file_path_elem = ET.SubElement(file_metadata, 'file_path')
    file_path_elem.text = file_path_metadata

    # Add all entities for this category
    for entity in entities:
        root.append(entity)

    # Write to file with pretty formatting
    tree = ET.ElementTree(root)
    ET.indent(tree, space='\t', level=0)

    try:
        # Convert to string to fix namespace handling
        from io import BytesIO
        temp = BytesIO()
        tree.write(temp, encoding='UTF-8', xml_declaration=False)
        xml_content = temp.getvalue().decode('UTF-8')

        # Fix the root element to include proper namespace
        xml_content = xml_content.replace(
            '<entity_data_dictionary version="2">',
            '<entity_data_dictionary version=\'2\' xmlns:xs=\'http://www.w3.org/2001/XMLSchema\'>'
        )

        with open(output_file, 'w', encoding='UTF-8') as f:
            f.write('<?xml version="1.0" encoding="UTF-8"?>\n')
            f.write(xml_content)
            f.write('\n')
        print(f"  Created: {output_file} ({len(entities)} entities)")
        return output_file
    except Exception as e:
        print(f"ERROR writing {output_file}: {e}")
        return None


def generate_extraction_report(entities_by_category, output_files, output_dir):
    """Generate a detailed extraction report."""
    report_file = os.path.join(output_dir, 'extraction_report.txt')

    total_entities = sum(len(entities) for entities in entities_by_category.values())

    with open(report_file, 'w') as f:
        f.write("=" * 80 + "\n")
        f.write("EDD EXTRACTION REPORT\n")
        f.write("=" * 80 + "\n\n")
        f.write(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write(f"Total entities extracted: {total_entities}\n\n")

        f.write("ENTITIES BY CATEGORY:\n")
        f.write("-" * 80 + "\n")

        for category in sorted(entities_by_category.keys()):
            entities = entities_by_category[category]
            f.write(f"\n{category.upper()} ({len(entities)} entities):\n")
            for entity in entities:
                entity_name = entity.get('name')
                comment = entity.get('comment', '')
                field_count = len(entity.findall('field'))
                f.write(f"  - {entity_name:30s} ({field_count:3d} fields) {comment}\n")

        f.write("\n" + "=" * 80 + "\n")
        f.write("OUTPUT FILES CREATED:\n")
        f.write("-" * 80 + "\n")

        for output_file in sorted(output_files):
            file_size = os.path.getsize(output_file)
            basename = os.path.basename(output_file)
            f.write(f"  {basename:40s} ({file_size:6d} bytes)\n")

        f.write("\n" + "=" * 80 + "\n")
        f.write("DIRECTORY STRUCTURE:\n")
        f.write("-" * 80 + "\n")
        f.write(f"\n{output_dir}/\n")
        for output_file in sorted(output_files):
            basename = os.path.basename(output_file)
            f.write(f"  ├── {basename}\n")
        f.write("  └── extraction_report.txt (this file)\n")

        f.write("\n" + "=" * 80 + "\n")
        f.write("NOTES:\n")
        f.write("-" * 80 + "\n")
        f.write("- Original file backed up as TaxReturn_edd.xml.original\n")
        f.write("- Each extracted file includes file_path metadata\n")
        f.write("- All entity structures preserved exactly as in original\n")
        f.write("- Files ready for use in multi-file EDD structure\n")
        f.write("\n" + "=" * 80 + "\n")

    print(f"\n  Created extraction report: {report_file}")
    return report_file


def main():
    """Main extraction process."""

    # Paths
    base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    xml_dir = os.path.join(base_dir, 'xml')
    input_file = os.path.join(xml_dir, 'TaxReturn_edd.xml')
    backup_file = os.path.join(xml_dir, 'TaxReturn_edd.xml.original')

    print("=" * 80)
    print("EDD EXTRACTION SCRIPT")
    print("=" * 80)
    print(f"\nInput file: {input_file}")
    print(f"Backup file: {backup_file}")
    print(f"Output directory: {xml_dir}")

    # Verify backup exists
    if not os.path.exists(backup_file):
        print(f"\nERROR: Backup file not found: {backup_file}")
        print("Please create backup before running extraction!")
        sys.exit(1)

    print(f"\n✓ Backup verified: {backup_file}")

    # Parse the EDD file
    print(f"\nParsing {input_file}...")
    entities_by_category = parse_edd_file(input_file)

    # Create output files
    print(f"\nCreating extracted files...")
    output_files = []

    for category in sorted(entities_by_category.keys()):
        entities = entities_by_category[category]
        output_file = create_edd_xml(category, entities, xml_dir)
        if output_file:
            output_files.append(output_file)

    # Generate report
    print(f"\nGenerating extraction report...")
    report_file = generate_extraction_report(entities_by_category, output_files, xml_dir)

    # Summary
    print("\n" + "=" * 80)
    print("EXTRACTION COMPLETE")
    print("=" * 80)
    print(f"\nTotal entities extracted: {sum(len(e) for e in entities_by_category.values())}")
    print(f"Files created: {len(output_files)}")
    print(f"Report: {report_file}")
    print("\nNext steps:")
    print("  1. Review the extraction_report.txt")
    print("  2. Verify the extracted XML files")
    print("  3. Update merge scripts to include new file structure")
    print("")


if __name__ == '__main__':
    main()
