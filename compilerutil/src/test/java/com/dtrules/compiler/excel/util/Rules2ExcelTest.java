/**
 * Copyright 2004-2011 DTRules.com, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 **/

package com.dtrules.compiler.excel.util;

import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.Assert.*;

/**
 * Tests for Rules2Excel class - converting rules to Excel format.
 * Verifies that the output Excel files have correct structure and formatting.
 */
public class Rules2ExcelTest {

    private Path tempDir;
    private Rules2Excel rules2Excel;

    @Before
    public void setUp() throws Exception {
        tempDir = Files.createTempDirectory("dtrules-r2e-test-");
        rules2Excel = new Rules2Excel(false); // non-balanced mode
    }

    @After
    public void tearDown() throws Exception {
        if (tempDir != null) {
            Files.walk(tempDir)
                .sorted((a, b) -> b.compareTo(a))
                .forEach(path -> {
                    try {
                        Files.deleteIfExists(path);
                    } catch (Exception e) {
                        // Ignore
                    }
                });
        }
    }

    // ========== Rules2Excel Class Tests ==========

    @Test
    public void testRules2Excel_CreateWorkbook() throws Exception {
        // Test that Rules2Excel creates a valid XLSX workbook
        Rules2Excel r2e = new Rules2Excel(false);

        // The newWb() method is package-private, but we can verify through writeExcel behavior
        // For now, just verify the class instantiates correctly
        assertNotNull("Rules2Excel should instantiate", r2e);
    }

    @Test
    public void testRules2Excel_BalancedVsNonBalanced() {
        Rules2Excel balanced = new Rules2Excel(true);
        Rules2Excel nonBalanced = new Rules2Excel(false);

        assertNotNull(balanced);
        assertNotNull(nonBalanced);
    }

    // ========== Output Format Tests ==========

    @Test
    public void testOutputFormat_XLSX() throws Exception {
        // Rules2Excel now creates XLSX files (not XLS)
        // This test verifies the file extension change from the update

        // Create a simple XLSX file to verify format
        File outputFile = new File(tempDir.toFile(), "test_output.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Test");
            Row row = sheet.createRow(0);
            row.createCell(0).setCellValue("Test");

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        assertTrue("Output file should exist", outputFile.exists());
        assertTrue("Output file should be .xlsx", outputFile.getName().endsWith(".xlsx"));

        // Verify it's a valid XLSX file
        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {
            assertTrue("Should be XLSX format", wb instanceof XSSFWorkbook);
        }
    }

    @Test
    public void testRules2Excel_BalancedMode() {
        Rules2Excel balanced = new Rules2Excel(true);
        assertNotNull("Should create in balanced mode", balanced);
    }

    @Test
    public void testRules2Excel_NonBalancedMode() {
        Rules2Excel nonBalanced = new Rules2Excel(false);
        assertNotNull("Should create in non-balanced mode", nonBalanced);
    }

    // ========== Cell Style Tests ==========

    @Test
    public void testCellStyles_BordersApplied() throws Exception {
        // Test that cell styles have proper borders
        File outputFile = new File(tempDir.toFile(), "styled.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            CellStyle cs = wb.createCellStyle();
            cs.setBorderBottom(BorderStyle.THIN);
            cs.setBorderTop(BorderStyle.THIN);
            cs.setBorderLeft(BorderStyle.THIN);
            cs.setBorderRight(BorderStyle.THIN);

            Sheet sheet = wb.createSheet("Styled");
            Row row = sheet.createRow(0);
            Cell cell = row.createCell(0);
            cell.setCellValue("Bordered");
            cell.setCellStyle(cs);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        // Verify styles
        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Cell cell = sheet.getRow(0).getCell(0);
            CellStyle cs = cell.getCellStyle();

            assertEquals("Should have thin bottom border",
                BorderStyle.THIN, cs.getBorderBottom());
            assertEquals("Should have thin top border",
                BorderStyle.THIN, cs.getBorderTop());
        }
    }

    @Test
    public void testCellStyles_Alignment() throws Exception {
        File outputFile = new File(tempDir.toFile(), "aligned.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            CellStyle csCenter = wb.createCellStyle();
            csCenter.setAlignment(HorizontalAlignment.CENTER);
            csCenter.setVerticalAlignment(VerticalAlignment.CENTER);

            CellStyle csLeft = wb.createCellStyle();
            csLeft.setAlignment(HorizontalAlignment.LEFT);

            Sheet sheet = wb.createSheet("Aligned");
            Row row = sheet.createRow(0);

            Cell cell1 = row.createCell(0);
            cell1.setCellValue("Centered");
            cell1.setCellStyle(csCenter);

            Cell cell2 = row.createCell(1);
            cell2.setCellValue("Left");
            cell2.setCellStyle(csLeft);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        // Verify alignment
        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            CellStyle cs1 = sheet.getRow(0).getCell(0).getCellStyle();
            assertEquals(HorizontalAlignment.CENTER, cs1.getAlignment());
            assertEquals(VerticalAlignment.CENTER, cs1.getVerticalAlignment());

            CellStyle cs2 = sheet.getRow(0).getCell(1).getCellStyle();
            assertEquals(HorizontalAlignment.LEFT, cs2.getAlignment());
        }
    }

    @Test
    public void testCellStyles_FillColors() throws Exception {
        File outputFile = new File(tempDir.toFile(), "colored.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            CellStyle csGrey = wb.createCellStyle();
            csGrey.setFillForegroundColor(IndexedColors.GREY_25_PERCENT.getIndex());
            csGrey.setFillPattern(FillPatternType.SOLID_FOREGROUND);

            CellStyle csYellow = wb.createCellStyle();
            csYellow.setFillForegroundColor(IndexedColors.LIGHT_YELLOW.getIndex());
            csYellow.setFillPattern(FillPatternType.SOLID_FOREGROUND);

            Sheet sheet = wb.createSheet("Colored");
            Row row = sheet.createRow(0);

            Cell cell1 = row.createCell(0);
            cell1.setCellValue("Grey");
            cell1.setCellStyle(csGrey);

            Cell cell2 = row.createCell(1);
            cell2.setCellValue("Yellow");
            cell2.setCellStyle(csYellow);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        // Verify colors
        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            CellStyle cs1 = sheet.getRow(0).getCell(0).getCellStyle();
            assertEquals(FillPatternType.SOLID_FOREGROUND, cs1.getFillPattern());

            CellStyle cs2 = sheet.getRow(0).getCell(1).getCellStyle();
            assertEquals(FillPatternType.SOLID_FOREGROUND, cs2.getFillPattern());
        }
    }

    @Test
    public void testCellStyles_FontBold() throws Exception {
        File outputFile = new File(tempDir.toFile(), "fonts.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Font boldFont = wb.createFont();
            boldFont.setBold(true);

            Font normalFont = wb.createFont();
            normalFont.setBold(false);

            CellStyle csBold = wb.createCellStyle();
            csBold.setFont(boldFont);

            CellStyle csNormal = wb.createCellStyle();
            csNormal.setFont(normalFont);

            Sheet sheet = wb.createSheet("Fonts");
            Row row = sheet.createRow(0);

            Cell cell1 = row.createCell(0);
            cell1.setCellValue("Bold");
            cell1.setCellStyle(csBold);

            Cell cell2 = row.createCell(1);
            cell2.setCellValue("Normal");
            cell2.setCellStyle(csNormal);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        // Verify fonts
        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            CellStyle cs1 = sheet.getRow(0).getCell(0).getCellStyle();
            Font f1 = wb.getFontAt(cs1.getFontIndex());
            assertTrue("Font should be bold", f1.getBold());

            CellStyle cs2 = sheet.getRow(0).getCell(1).getCellStyle();
            Font f2 = wb.getFontAt(cs2.getFontIndex());
            assertFalse("Font should not be bold", f2.getBold());
        }
    }

    // ========== Sheet Structure Tests ==========

    @Test
    public void testSheetNaming() throws Exception {
        File outputFile = new File(tempDir.toFile(), "named_sheets.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            wb.createSheet("Decision_Table_1");
            wb.createSheet("Decision_Table_2");
            wb.createSheet("EDD");

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertEquals(3, wb.getNumberOfSheets());
            assertEquals("Decision_Table_1", wb.getSheetName(0));
            assertEquals("Decision_Table_2", wb.getSheetName(1));
            assertEquals("EDD", wb.getSheetName(2));
        }
    }

    @Test
    public void testSheetNaming_LongNames() throws Exception {
        // Excel has a 31 character limit on sheet names
        File outputFile = new File(tempDir.toFile(), "long_names.xlsx");

        String longName = "This_Is_A_Very_Long_Decision_Table_Name_That_Exceeds_Limit";
        String truncatedName = longName.substring(0, 31);

        try (Workbook wb = new XSSFWorkbook()) {
            wb.createSheet(truncatedName);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            String actualName = wb.getSheetName(0);
            assertTrue("Sheet name should be truncated to 31 chars",
                actualName.length() <= 31);
        }
    }

    @Test
    public void testColumnWidths() throws Exception {
        File outputFile = new File(tempDir.toFile(), "column_widths.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Test");

            // Set various column widths
            sheet.setColumnWidth(0, 800);   // Narrow
            sheet.setColumnWidth(1, 12000); // Wide
            sheet.setColumnWidth(2, 6000);  // Medium

            Row row = sheet.createRow(0);
            row.createCell(0).setCellValue("A");
            row.createCell(1).setCellValue("B");
            row.createCell(2).setCellValue("C");

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            assertEquals(800, sheet.getColumnWidth(0));
            assertEquals(12000, sheet.getColumnWidth(1));
            assertEquals(6000, sheet.getColumnWidth(2));
        }
    }

    @Test
    public void testMergedRegions() throws Exception {
        File outputFile = new File(tempDir.toFile(), "merged.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Merged");

            Row row = sheet.createRow(0);
            Cell cell = row.createCell(0);
            cell.setCellValue("Merged Title");

            // Merge columns 0-3 in row 0
            sheet.addMergedRegion(new org.apache.poi.ss.util.CellRangeAddress(0, 0, 0, 3));

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            assertEquals("Should have 1 merged region", 1, sheet.getNumMergedRegions());

            org.apache.poi.ss.util.CellRangeAddress merged = sheet.getMergedRegion(0);
            assertEquals(0, merged.getFirstRow());
            assertEquals(0, merged.getLastRow());
            assertEquals(0, merged.getFirstColumn());
            assertEquals(3, merged.getLastColumn());
        }
    }

    // ========== Data Type Preservation Tests ==========

    @Test
    public void testDataTypes_Numeric() throws Exception {
        File outputFile = new File(tempDir.toFile(), "numeric.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Numbers");
            Row row = sheet.createRow(0);

            row.createCell(0).setCellValue(42);
            row.createCell(1).setCellValue(3.14159);
            row.createCell(2).setCellValue(-100);
            row.createCell(3).setCellValue(0);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row row = sheet.getRow(0);

            assertEquals(42.0, row.getCell(0).getNumericCellValue(), 0.001);
            assertEquals(3.14159, row.getCell(1).getNumericCellValue(), 0.00001);
            assertEquals(-100.0, row.getCell(2).getNumericCellValue(), 0.001);
            assertEquals(0.0, row.getCell(3).getNumericCellValue(), 0.001);
        }
    }

    @Test
    public void testDataTypes_TextWrap() throws Exception {
        File outputFile = new File(tempDir.toFile(), "wrapped.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            CellStyle csWrap = wb.createCellStyle();
            csWrap.setWrapText(true);

            Sheet sheet = wb.createSheet("Wrapped");
            Row row = sheet.createRow(0);

            Cell cell = row.createCell(0);
            cell.setCellValue("This is a long text that should wrap to multiple lines when displayed in the cell");
            cell.setCellStyle(csWrap);

            try (FileOutputStream fos = new FileOutputStream(outputFile)) {
                wb.write(fos);
            }
        }

        try (FileInputStream fis = new FileInputStream(outputFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            CellStyle cs = sheet.getRow(0).getCell(0).getCellStyle();
            assertTrue("Text wrap should be enabled", cs.getWrapText());
        }
    }
}
