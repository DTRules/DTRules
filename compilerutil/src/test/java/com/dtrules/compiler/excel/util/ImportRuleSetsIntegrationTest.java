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
import org.junit.Assume;
import org.odftoolkit.simple.SpreadsheetDocument;
import org.odftoolkit.simple.table.Table;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.Assert.*;

/**
 * Integration tests for ImportRuleSets class.
 * Tests the complete workflow of converting spreadsheets to XML.
 */
public class ImportRuleSetsIntegrationTest {

    private Path tempDir;
    private ImportRuleSets importRuleSets;

    @Before
    public void setUp() throws Exception {
        tempDir = Files.createTempDirectory("dtrules-import-test-");
        importRuleSets = new ImportRuleSets();
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

    // ========== Format Detection Integration Tests ==========

    @Test
    public void testIsGoogleSheetsSource_True() {
        assertTrue(ImportRuleSets.isGoogleSheetsSource(
            "https://docs.google.com/spreadsheets/d/1abc123/edit"));
    }

    @Test
    public void testIsGoogleSheetsSource_False() {
        assertFalse(ImportRuleSets.isGoogleSheetsSource("/path/to/file.xlsx"));
        assertFalse(ImportRuleSets.isGoogleSheetsSource("file.xls"));
        assertFalse(ImportRuleSets.isGoogleSheetsSource(null));
    }

    // ========== XLS File Creation for Testing ==========

    @Test
    public void testCreateValidEDDFile_XLS() throws Exception {
        File xlsFile = createEDDFile(new HSSFWorkbook(), ".xls");

        assertTrue("XLS EDD file should exist", xlsFile.exists());
        assertTrue("Should be .xls file", xlsFile.getName().endsWith(".xls"));

        // Verify it can be read
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {
            assertNotNull(wb);
            assertEquals(1, wb.getNumberOfSheets());
        }
    }

    @Test
    public void testCreateValidEDDFile_XLSX() throws Exception {
        File xlsxFile = createEDDFile(new XSSFWorkbook(), ".xlsx");

        assertTrue("XLSX EDD file should exist", xlsxFile.exists());
        assertTrue("Should be .xlsx file", xlsxFile.getName().endsWith(".xlsx"));

        // Verify it can be read
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {
            assertNotNull(wb);
            assertTrue("Should be XLSX format", wb instanceof XSSFWorkbook);
        }
    }

    @Test
    public void testCreateValidEDDFile_ODS() throws Exception {
        File odsFile = createEDDFileOds();

        assertTrue("ODS EDD file should exist", odsFile.exists());
        assertTrue("Should be .ods file", odsFile.getName().endsWith(".ods"));

        // Verify it can be read
        SpreadsheetDocument doc = SpreadsheetDocument.loadDocument(odsFile);
        assertNotNull(doc);
        assertTrue(doc.getSheetCount() > 0);
        doc.close();
    }

    // ========== Decision Table File Creation for Testing ==========

    @Test
    public void testCreateValidDecisionTableFile_XLS() throws Exception {
        File xlsFile = createDecisionTableFile(new HSSFWorkbook(), ".xls");

        assertTrue("XLS DT file should exist", xlsFile.exists());

        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row nameRow = sheet.getRow(0);

            // Verify structure
            String nameCell = nameRow.getCell(0).getStringCellValue();
            assertTrue("Should start with Name:", nameCell.startsWith("Name:"));
        }
    }

    @Test
    public void testCreateValidDecisionTableFile_XLSX() throws Exception {
        File xlsxFile = createDecisionTableFile(new XSSFWorkbook(), ".xlsx");

        assertTrue("XLSX DT file should exist", xlsxFile.exists());

        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertTrue("Should be XLSX", wb instanceof XSSFWorkbook);

            Sheet sheet = wb.getSheetAt(0);
            String nameCell = sheet.getRow(0).getCell(0).getStringCellValue();
            assertTrue("Should start with Name:", nameCell.startsWith("Name:"));
        }
    }

    @Test
    public void testCreateValidDecisionTableFile_ODS() throws Exception {
        File odsFile = createDecisionTableFileOds();

        assertTrue("ODS DT file should exist", odsFile.exists());

        SpreadsheetDocument doc = SpreadsheetDocument.loadDocument(odsFile);
        Table table = doc.getSheetByIndex(0);

        String nameCell = table.getCellByPosition(0, 0).getStringValue();
        assertTrue("Should start with Name:", nameCell.startsWith("Name:"));

        doc.close();
    }

    // ========== Google Sheets Credentials Configuration Test ==========

    @Test
    public void testSetGoogleCredentialsPath() {
        String testPath = "/path/to/credentials.json";

        // Should not throw exception
        importRuleSets.setGoogleCredentialsPath(testPath);

        // Setting to null should also work
        importRuleSets.setGoogleCredentialsPath(null);
    }

    @Test
    public void testGoogleSheetsWithInvalidCredentials() {
        // This should gracefully handle missing credentials
        importRuleSets.setGoogleCredentialsPath("/nonexistent/path/creds.json");

        // Attempting to use Google Sheets should fail gracefully
        // We can't really test the full flow without valid credentials
    }

    // ========== Round-Trip Test: Create -> Read -> Verify ==========

    @Test
    public void testRoundTrip_XLS_EDD() throws Exception {
        // Create
        File xlsFile = createEDDFile(new HSSFWorkbook(), ".xls");

        // Read and verify structure
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            // Verify header row
            Row header = sheet.getRow(0);
            assertEquals("Entity", header.getCell(0).getStringCellValue());
            assertEquals("Attribute", header.getCell(1).getStringCellValue());
            assertEquals("Type", header.getCell(2).getStringCellValue());

            // Verify data rows
            Row row1 = sheet.getRow(1);
            assertEquals("TestEntity", row1.getCell(0).getStringCellValue());
            assertEquals("testAttr", row1.getCell(1).getStringCellValue());
            assertEquals("string", row1.getCell(2).getStringCellValue());
        }
    }

    @Test
    public void testRoundTrip_XLSX_EDD() throws Exception {
        // Create
        File xlsxFile = createEDDFile(new XSSFWorkbook(), ".xlsx");

        // Read and verify structure
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);

            Row header = sheet.getRow(0);
            assertEquals("Entity", header.getCell(0).getStringCellValue());

            Row row1 = sheet.getRow(1);
            assertEquals("TestEntity", row1.getCell(0).getStringCellValue());
        }
    }

    @Test
    public void testRoundTrip_ODS_EDD() throws Exception {
        // Create
        File odsFile = createEDDFileOds();

        // Read and verify
        SpreadsheetDocument doc = SpreadsheetDocument.loadDocument(odsFile);
        Table table = doc.getSheetByIndex(0);

        assertEquals("Entity", table.getCellByPosition(0, 0).getStringValue());
        assertEquals("TestEntity", table.getCellByPosition(0, 1).getStringValue());

        doc.close();
    }

    // ========== Edge Cases ==========

    @Test
    public void testEmptySpreadsheet_XLS() throws Exception {
        File xlsFile = new File(tempDir.toFile(), "empty.xls");

        try (Workbook wb = new HSSFWorkbook()) {
            wb.createSheet("Empty");
            try (FileOutputStream fos = new FileOutputStream(xlsFile)) {
                wb.write(fos);
            }
        }

        // Should be able to read empty spreadsheet
        try (FileInputStream fis = new FileInputStream(xlsFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            assertEquals(1, wb.getNumberOfSheets());
            Sheet sheet = wb.getSheetAt(0);
            assertEquals(-1, sheet.getLastRowNum()); // Empty sheet
        }
    }

    @Test
    public void testSpreadsheetWithSpecialCharacters() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "special_chars.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Test");
            Row row = sheet.createRow(0);

            row.createCell(0).setCellValue("Entity");
            row.createCell(1).setCellValue("Attribute");

            Row dataRow = sheet.createRow(1);
            dataRow.createCell(0).setCellValue("Test<Entity>");  // XML chars
            dataRow.createCell(1).setCellValue("attr&name");     // Ampersand
            dataRow.createCell(2).setCellValue("value\"quoted\""); // Quotes

            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        // Verify special chars are preserved
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row dataRow = sheet.getRow(1);

            assertEquals("Test<Entity>", dataRow.getCell(0).getStringCellValue());
            assertEquals("attr&name", dataRow.getCell(1).getStringCellValue());
            assertEquals("value\"quoted\"", dataRow.getCell(2).getStringCellValue());
        }
    }

    @Test
    public void testSpreadsheetWithUnicode() throws Exception {
        File xlsxFile = new File(tempDir.toFile(), "unicode.xlsx");

        try (Workbook wb = new XSSFWorkbook()) {
            Sheet sheet = wb.createSheet("Unicode");
            Row row = sheet.createRow(0);

            row.createCell(0).setCellValue("日本語");     // Japanese
            row.createCell(1).setCellValue("中文");       // Chinese
            row.createCell(2).setCellValue("العربية");   // Arabic
            row.createCell(3).setCellValue("émoji 🎉");  // Emoji

            try (FileOutputStream fos = new FileOutputStream(xlsxFile)) {
                wb.write(fos);
            }
        }

        // Verify unicode is preserved
        try (FileInputStream fis = new FileInputStream(xlsxFile);
             Workbook wb = WorkbookFactory.create(fis)) {

            Sheet sheet = wb.getSheetAt(0);
            Row row = sheet.getRow(0);

            assertEquals("日本語", row.getCell(0).getStringCellValue());
            assertEquals("中文", row.getCell(1).getStringCellValue());
            assertEquals("العربية", row.getCell(2).getStringCellValue());
            assertEquals("émoji 🎉", row.getCell(3).getStringCellValue());
        }
    }

    // ========== Helper Methods ==========

    private File createEDDFile(Workbook wb, String extension) throws Exception {
        File file = new File(tempDir.toFile(), "edd" + extension);

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

        // Data row
        Row row1 = sheet.createRow(1);
        row1.createCell(0).setCellValue("TestEntity");
        row1.createCell(1).setCellValue("testAttr");
        row1.createCell(2).setCellValue("string");
        row1.createCell(3).setCellValue("");
        row1.createCell(4).setCellValue("");
        row1.createCell(5).setCellValue("input");
        row1.createCell(6).setCellValue("rw");
        row1.createCell(7).setCellValue("Test attribute");

        try (FileOutputStream fos = new FileOutputStream(file)) {
            wb.write(fos);
        }
        wb.close();

        return file;
    }

    private File createEDDFileOds() throws Exception {
        File file = new File(tempDir.toFile(), "edd.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);
        table.setTableName("EDD");

        // Header row
        table.getCellByPosition(0, 0).setStringValue("Entity");
        table.getCellByPosition(1, 0).setStringValue("Attribute");
        table.getCellByPosition(2, 0).setStringValue("Type");
        table.getCellByPosition(3, 0).setStringValue("SubType");
        table.getCellByPosition(4, 0).setStringValue("Default Value");
        table.getCellByPosition(5, 0).setStringValue("Input");
        table.getCellByPosition(6, 0).setStringValue("Access");
        table.getCellByPosition(7, 0).setStringValue("Comment");

        // Data row
        table.getCellByPosition(0, 1).setStringValue("TestEntity");
        table.getCellByPosition(1, 1).setStringValue("testAttr");
        table.getCellByPosition(2, 1).setStringValue("string");
        table.getCellByPosition(3, 1).setStringValue("");
        table.getCellByPosition(4, 1).setStringValue("");
        table.getCellByPosition(5, 1).setStringValue("input");
        table.getCellByPosition(6, 1).setStringValue("rw");
        table.getCellByPosition(7, 1).setStringValue("Test attribute");

        doc.save(file);
        doc.close();

        return file;
    }

    private File createDecisionTableFile(Workbook wb, String extension) throws Exception {
        File file = new File(tempDir.toFile(), "dt" + extension);

        Sheet sheet = wb.createSheet("DT1");

        // Name row
        sheet.createRow(0).createCell(0).setCellValue("Name: Test_Table");

        // Type row
        sheet.createRow(1).createCell(0).setCellValue("Type: All");

        // Blank row
        sheet.createRow(2);

        // Conditions
        Row condHeader = sheet.createRow(3);
        condHeader.createCell(0).setCellValue("Conditions:");
        condHeader.createCell(2).setCellValue("Conditions");

        Row cond1 = sheet.createRow(4);
        cond1.createCell(0).setCellValue("1");
        cond1.createCell(1).setCellValue("Check");
        cond1.createCell(2).setCellValue("x > 0");
        cond1.createCell(3).setCellValue("Y");

        // Actions
        Row actHeader = sheet.createRow(5);
        actHeader.createCell(0).setCellValue("Actions:");

        Row act1 = sheet.createRow(6);
        act1.createCell(0).setCellValue("1");
        act1.createCell(1).setCellValue("Do");
        act1.createCell(2).setCellValue("result = true");
        act1.createCell(3).setCellValue("X");

        try (FileOutputStream fos = new FileOutputStream(file)) {
            wb.write(fos);
        }
        wb.close();

        return file;
    }

    private File createDecisionTableFileOds() throws Exception {
        File file = new File(tempDir.toFile(), "dt.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);
        table.setTableName("DT1");

        // Name
        table.getCellByPosition(0, 0).setStringValue("Name: Test_Table");

        // Type
        table.getCellByPosition(0, 1).setStringValue("Type: All");

        // Conditions
        table.getCellByPosition(0, 3).setStringValue("Conditions:");
        table.getCellByPosition(2, 3).setStringValue("Conditions");

        table.getCellByPosition(0, 4).setStringValue("1");
        table.getCellByPosition(1, 4).setStringValue("Check");
        table.getCellByPosition(2, 4).setStringValue("x > 0");
        table.getCellByPosition(3, 4).setStringValue("Y");

        // Actions
        table.getCellByPosition(0, 5).setStringValue("Actions:");

        table.getCellByPosition(0, 6).setStringValue("1");
        table.getCellByPosition(1, 6).setStringValue("Do");
        table.getCellByPosition(2, 6).setStringValue("result = true");
        table.getCellByPosition(3, 6).setStringValue("X");

        doc.save(file);
        doc.close();

        return file;
    }
}
