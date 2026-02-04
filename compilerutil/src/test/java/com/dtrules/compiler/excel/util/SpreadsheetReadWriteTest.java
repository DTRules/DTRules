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

import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.Assert.*;

/**
 * Tests for reading and writing various spreadsheet formats.
 * Creates temporary test files and verifies round-trip conversion.
 */
public class SpreadsheetReadWriteTest {

    private Path tempDir;

    @Before
    public void setUp() throws Exception {
        tempDir = Files.createTempDirectory("dtrules-test-");
    }

    @After
    public void tearDown() throws Exception {
        // Clean up temp files
        if (tempDir != null) {
            Files.walk(tempDir)
                .sorted((a, b) -> b.compareTo(a)) // Reverse order for deletion
                .forEach(path -> {
                    try {
                        Files.deleteIfExists(path);
                    } catch (IOException e) {
                        // Ignore cleanup errors
                    }
                });
        }
    }

    // ========== XLS (Excel 97-2003) Tests ==========

    @Test
    public void testCreateAndReadXLS() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "test.xls");

        // Create XLS file
        try (Workbook wb = new HSSFWorkbook()) {
            createTestSheet(wb, "TestSheet");
            try (FileOutputStream fos = new FileOutputStream(xlsFile)) {
                wb.write(fos);
            }
        }

        assertTrue("XLS file should exist", xlsFile.exists());

        // Read XLS file using WorkbookFactory (auto-detection)
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertEquals("Should have 1 sheet", 1, wb.getNumberOfSheets());
            Sheet sheet = wb.getSheetAt(0);
            assertEquals("Sheet name should match", "TestSheet", sheet.getSheetName());

            verifyTestSheetContent(sheet);
        }
    }

    @Test
    public void testXLSCellTypes() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "celltypes.xls");

        // Create XLS with various cell types
        try (Workbook wb = new HSSFWorkbook()) {
            Sheet sheet = wb.createSheet("CellTypes");
            Row row = sheet.createRow(0);

            row.createCell(0).setCellValue("String");
            row.createCell(1).setCellValue(123.45);
            row.createCell(2).setCellValue(true);
            row.createCell(3).setCellValue((String) null); // Blank
            row.createCell(4); // Empty cell

            try (FileOutputStream fos = new FileOutputStream(xlsFile)) {
                wb.write(fos);
            }
        }

        // Verify reading
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row row = sheet.getRow(0);

            assertEquals(CellType.STRING, row.getCell(0).getCellType());
            assertEquals("String", row.getCell(0).getStringCellValue());

            assertEquals(CellType.NUMERIC, row.getCell(1).getCellType());
            assertEquals(123.45, row.getCell(1).getNumericCellValue(), 0.001);

            assertEquals(CellType.BOOLEAN, row.getCell(2).getCellType());
            assertTrue(row.getCell(2).getBooleanCellValue());
        }
    }

    // ========== XLSX (Excel 2007+) Tests ==========

    @Test
    public void testCreateAndReadXLSX() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "test.xlsx");

        // Create XLSX file
        try (Workbook wb = new XSSFWorkbook()) {
            createTestSheet(wb, "TestSheet");
            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        assertTrue("XLSX file should exist", xlsxFile.exists());

        // Read XLSX file
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertEquals("Should have 1 sheet", 1, wb.getNumberOfSheets());
            Sheet sheet = wb.getSheetAt(0);
            assertEquals("Sheet name should match", "TestSheet", sheet.getSheetName());

            verifyTestSheetContent(sheet);
        }
    }

    @Test
    public void testXLSXMultipleSheets() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "multisheets.xlsx");

        // Create XLSX with multiple sheets
        try (Workbook wb = new XSSFWorkbook()) {
            createTestSheet(wb, "Sheet1");
            createTestSheet(wb, "Sheet2");
            createTestSheet(wb, "Sheet3");

            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        // Verify multiple sheets
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertEquals("Should have 3 sheets", 3, wb.getNumberOfSheets());
            assertEquals("Sheet1", wb.getSheetName(0));
            assertEquals("Sheet2", wb.getSheetName(1));
            assertEquals("Sheet3", wb.getSheetName(2));
        }
    }

    @Test
    public void testXLSXLargeData() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "large.xlsx");
        int numRows = 1000;
        int numCols = 20;

        // Create XLSX with large dataset
        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("LargeData");

            for (int r = 0; r < numRows; r++) {
                Row row = sheet.createRow(r);
                for (int c = 0; c < numCols; c++) {
                    row.createCell(c).setCellValue("R" + r + "C" + c);
                }
            }

            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        // Verify
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            assertEquals(numRows - 1, sheet.getLastRowNum());

            // Spot check some values
            assertEquals("R0C0", sheet.getRow(0).getCell(0).getStringCellValue());
            assertEquals("R500C10", sheet.getRow(500).getCell(10).getStringCellValue());
            assertEquals("R999C19", sheet.getRow(999).getCell(19).getStringCellValue());
        }
    }

    // ========== Format Auto-Detection Tests ==========

    @Test
    public void testWorkbookFactoryAutoDetection_XLS() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "autodetect.xls");

        // Create XLS
        try (Workbook wb = new HSSFWorkbook()) {
            wb.createSheet("Test");
            try (FileOutputStream fos = new FileOutputStream(xlsFile)) {
                wb.write(fos);
            }
        }

        // Auto-detect and read
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {
            assertTrue("Should detect as HSSF (XLS)", wb instanceof HSSFWorkbook);
        }
    }

    @Test
    public void testWorkbookFactoryAutoDetection_XLSX() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "autodetect.xlsx");

        // Create XLSX
        try (Workbook wb = new XSSFWorkbook()) {
            wb.createSheet("Test");
            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        // Auto-detect and read
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {
            assertTrue("Should detect as XSSF (XLSX)", wb instanceof XSSFWorkbook);
        }
    }

    // ========== EDD-like Structure Tests ==========

    @Test
    public void testEDDStructure_XLS() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "edd.xls");
        createEDDTestFile(xlsFile, new HSSFWorkbook());
        verifyEDDStructure(xlsFile);
    }

    @Test
    public void testEDDStructure_XLSX() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "edd.xlsx");
        createEDDTestFile(xlsxFile, new XSSFWorkbook());
        verifyEDDStructure(xlsxFile);
    }

    // ========== Decision Table Structure Tests ==========

    @Test
    public void testDecisionTableStructure_XLS() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "dt.xls");
        createDecisionTableTestFile(xlsFile, new HSSFWorkbook());
        verifyDecisionTableStructure(xlsFile);
    }

    @Test
    public void testDecisionTableStructure_XLSX() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "dt.xlsx");
        createDecisionTableTestFile(xlsxFile, new XSSFWorkbook());
        verifyDecisionTableStructure(xlsxFile);
    }

    // ========== Helper Methods ==========

    private void createTestSheet(Workbook wb, String sheetName) {
        Sheet sheet = wb.createSheet(sheetName);
        Row headerRow = sheet.createRow(0);
        headerRow.createCell(0).setCellValue("Column1");
        headerRow.createCell(1).setCellValue("Column2");
        headerRow.createCell(2).setCellValue("Column3");

        Row dataRow = sheet.createRow(1);
        dataRow.createCell(0).setCellValue("Value1");
        dataRow.createCell(1).setCellValue(100);
        dataRow.createCell(2).setCellValue(true);
    }

    private void verifyTestSheetContent(Sheet sheet) {
        Row headerRow = sheet.getRow(0);
        assertEquals("Column1", headerRow.getCell(0).getStringCellValue());
        assertEquals("Column2", headerRow.getCell(1).getStringCellValue());
        assertEquals("Column3", headerRow.getCell(2).getStringCellValue());

        Row dataRow = sheet.getRow(1);
        assertEquals("Value1", dataRow.getCell(0).getStringCellValue());
        assertEquals(100.0, dataRow.getCell(1).getNumericCellValue(), 0.001);
        assertTrue(dataRow.getCell(2).getBooleanCellValue());
    }

    private void createEDDTestFile(File file, Workbook wb) throws Exception {
        Sheet sheet = wb.createSheet("EDD");

        // Header row
        Row header = sheet.createRow(0);
        header.createCell(0).setCellValue("Entity");
        header.createCell(1).setCellValue("Attribute");
        header.createCell(2).setCellValue("Type");
        header.createCell(3).setCellValue("SubType");
        header.createCell(4).setCellValue("Default Value");
        header.createCell(5).setCellValue("Input");
        header.createCell(6).setCellValue("Access");
        header.createCell(7).setCellValue("Comment");

        // Data rows
        Row row1 = sheet.createRow(1);
        row1.createCell(0).setCellValue("Person");
        row1.createCell(1).setCellValue("name");
        row1.createCell(2).setCellValue("string");
        row1.createCell(3).setCellValue("");
        row1.createCell(4).setCellValue("");
        row1.createCell(5).setCellValue("input");
        row1.createCell(6).setCellValue("rw");
        row1.createCell(7).setCellValue("Person's name");

        Row row2 = sheet.createRow(2);
        row2.createCell(0).setCellValue("Person");
        row2.createCell(1).setCellValue("age");
        row2.createCell(2).setCellValue("integer");
        row2.createCell(3).setCellValue("");
        row2.createCell(4).setCellValue("0");
        row2.createCell(5).setCellValue("input");
        row2.createCell(6).setCellValue("rw");
        row2.createCell(7).setCellValue("Person's age");

        try (FileOutputStream fos = new FileOutputStream(file)) {
            wb.write(fos);
        }
        wb.close();
    }

    private void verifyEDDStructure(File file) throws Exception {
        try (FileInputStream fis = new FileInputStream(file);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row header = sheet.getRow(0);

            // Verify headers
            assertEquals("Entity", header.getCell(0).getStringCellValue());
            assertEquals("Attribute", header.getCell(1).getStringCellValue());
            assertEquals("Type", header.getCell(2).getStringCellValue());

            // Verify data
            Row row1 = sheet.getRow(1);
            assertEquals("Person", row1.getCell(0).getStringCellValue());
            assertEquals("name", row1.getCell(1).getStringCellValue());
            assertEquals("string", row1.getCell(2).getStringCellValue());
        }
    }

    private void createDecisionTableTestFile(File file, Workbook wb) throws Exception {
        Sheet sheet = wb.createSheet("DecisionTable1");

        // Name row
        Row nameRow = sheet.createRow(0);
        nameRow.createCell(0).setCellValue("Name: Test_Decision_Table");

        // Type row
        Row typeRow = sheet.createRow(1);
        typeRow.createCell(0).setCellValue("Type: All");

        // Blank row
        sheet.createRow(2);

        // Conditions header
        Row condHeader = sheet.createRow(3);
        condHeader.createCell(0).setCellValue("Conditions:");
        condHeader.createCell(1).setCellValue("");
        condHeader.createCell(2).setCellValue("Conditions");
        condHeader.createCell(3).setCellValue("1");
        condHeader.createCell(4).setCellValue("2");

        // Condition rows
        Row cond1 = sheet.createRow(4);
        cond1.createCell(0).setCellValue("1");
        cond1.createCell(1).setCellValue("Age check");
        cond1.createCell(2).setCellValue("person.age > 18");
        cond1.createCell(3).setCellValue("Y");
        cond1.createCell(4).setCellValue("N");

        // Actions header
        Row actHeader = sheet.createRow(5);
        actHeader.createCell(0).setCellValue("Actions:");

        // Action rows
        Row act1 = sheet.createRow(6);
        act1.createCell(0).setCellValue("1");
        act1.createCell(1).setCellValue("Set adult flag");
        act1.createCell(2).setCellValue("person.isAdult = true");
        act1.createCell(3).setCellValue("X");
        act1.createCell(4).setCellValue("");

        try (FileOutputStream fos = new FileOutputStream(file)) {
            wb.write(fos);
        }
        wb.close();
    }

    private void verifyDecisionTableStructure(File file) throws Exception {
        try (FileInputStream fis = new FileInputStream(file);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            // Verify Name row
            Row nameRow = sheet.getRow(0);
            String nameCell = nameRow.getCell(0).getStringCellValue();
            assertTrue("Should have Name:", nameCell.startsWith("Name:"));
            assertTrue("Should have table name", nameCell.contains("Test_Decision_Table"));

            // Verify Conditions header exists
            boolean foundConditions = false;
            for (int i = 0; i <= sheet.getLastRowNum(); i++) {
                Row row = sheet.getRow(i);
                if (row != null && row.getCell(0) != null) {
                    String cell = row.getCell(0).getStringCellValue();
                    if (cell.startsWith("Conditions:")) {
                        foundConditions = true;
                        break;
                    }
                }
            }
            assertTrue("Should have Conditions section", foundConditions);
        }
    }
}
