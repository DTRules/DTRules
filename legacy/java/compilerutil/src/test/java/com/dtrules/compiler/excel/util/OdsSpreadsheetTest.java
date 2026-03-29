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

import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.odftoolkit.simple.SpreadsheetDocument;
import org.odftoolkit.simple.table.Cell;
import org.odftoolkit.simple.table.Row;
import org.odftoolkit.simple.table.Table;

import java.io.File;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.Assert.*;

/**
 * Tests for ODS (OpenDocument Spreadsheet) format support.
 * Uses the simple-odf library to create and read ODS files.
 */
public class OdsSpreadsheetTest {

    private Path tempDir;

    @Before
    public void setUp() throws Exception {
        tempDir = Files.createTempDirectory("dtrules-ods-test-");
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

    // ========== ODS Creation and Reading Tests ==========

    @Test
    public void testCreateAndReadODS() throws Exception {
        File odsFile = new File(tempDir.toFile(), "test.ods");

        // Create ODS file
        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);
        table.setTableName("TestSheet");

        // Add header row
        Row headerRow = table.getRowByIndex(0);
        headerRow.getCellByIndex(0).setStringValue("Column1");
        headerRow.getCellByIndex(1).setStringValue("Column2");
        headerRow.getCellByIndex(2).setStringValue("Column3");

        // Add data row
        Row dataRow = table.getRowByIndex(1);
        dataRow.getCellByIndex(0).setStringValue("Value1");
        dataRow.getCellByIndex(1).setDoubleValue(100.0);
        dataRow.getCellByIndex(2).setStringValue("Value3");

        doc.save(odsFile);
        doc.close();

        assertTrue("ODS file should exist", odsFile.exists());

        // Read ODS file
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        assertEquals("TestSheet", readTable.getTableName());

        // Verify header
        assertEquals("Column1", readTable.getCellByPosition(0, 0).getStringValue());
        assertEquals("Column2", readTable.getCellByPosition(1, 0).getStringValue());
        assertEquals("Column3", readTable.getCellByPosition(2, 0).getStringValue());

        // Verify data
        assertEquals("Value1", readTable.getCellByPosition(0, 1).getStringValue());
        assertEquals(100.0, readTable.getCellByPosition(1, 1).getDoubleValue(), 0.001);
        assertEquals("Value3", readTable.getCellByPosition(2, 1).getStringValue());

        readDoc.close();
    }

    @Test
    public void testODS_MultipleSheets() throws Exception {
        File odsFile = new File(tempDir.toFile(), "multisheets.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();

        // First sheet is created by default
        Table sheet1 = doc.getSheetByIndex(0);
        sheet1.setTableName("Sheet1");
        sheet1.getCellByPosition(0, 0).setStringValue("Sheet 1 Data");

        // Add second sheet
        Table sheet2 = doc.appendSheet("Sheet2");
        sheet2.getCellByPosition(0, 0).setStringValue("Sheet 2 Data");

        // Add third sheet
        Table sheet3 = doc.appendSheet("Sheet3");
        sheet3.getCellByPosition(0, 0).setStringValue("Sheet 3 Data");

        doc.save(odsFile);
        doc.close();

        // Verify
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        assertEquals(3, readDoc.getSheetCount());

        assertEquals("Sheet1", readDoc.getSheetByIndex(0).getTableName());
        assertEquals("Sheet2", readDoc.getSheetByIndex(1).getTableName());
        assertEquals("Sheet3", readDoc.getSheetByIndex(2).getTableName());

        assertEquals("Sheet 1 Data", readDoc.getSheetByIndex(0).getCellByPosition(0, 0).getStringValue());
        assertEquals("Sheet 2 Data", readDoc.getSheetByIndex(1).getCellByPosition(0, 0).getStringValue());
        assertEquals("Sheet 3 Data", readDoc.getSheetByIndex(2).getCellByPosition(0, 0).getStringValue());

        readDoc.close();
    }

    @Test
    public void testODS_CellTypes() throws Exception {
        File odsFile = new File(tempDir.toFile(), "celltypes.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);

        Row row = table.getRowByIndex(0);
        row.getCellByIndex(0).setStringValue("String");
        row.getCellByIndex(1).setDoubleValue(123.45);
        row.getCellByIndex(2).setBooleanValue(true);
        row.getCellByIndex(3).setStringValue(""); // Empty

        doc.save(odsFile);
        doc.close();

        // Verify
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        assertEquals("String", readTable.getCellByPosition(0, 0).getStringValue());
        assertEquals(123.45, readTable.getCellByPosition(1, 0).getDoubleValue(), 0.001);
        assertTrue(readTable.getCellByPosition(2, 0).getBooleanValue());

        readDoc.close();
    }

    @Test
    public void testODS_LargeData() throws Exception {
        File odsFile = new File(tempDir.toFile(), "large.ods");
        int numRows = 100;
        int numCols = 10;

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);

        for (int r = 0; r < numRows; r++) {
            Row row = table.getRowByIndex(r);
            for (int c = 0; c < numCols; c++) {
                row.getCellByIndex(c).setStringValue("R" + r + "C" + c);
            }
        }

        doc.save(odsFile);
        doc.close();

        // Verify
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        assertEquals("R0C0", readTable.getCellByPosition(0, 0).getStringValue());
        assertEquals("R50C5", readTable.getCellByPosition(5, 50).getStringValue());
        assertEquals("R99C9", readTable.getCellByPosition(9, 99).getStringValue());

        readDoc.close();
    }

    // ========== EDD Structure Tests (ODS) ==========

    @Test
    public void testODS_EDDStructure() throws Exception {
        File odsFile = new File(tempDir.toFile(), "edd.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);
        table.setTableName("EDD");

        // Header row
        Row header = table.getRowByIndex(0);
        header.getCellByIndex(0).setStringValue("Entity");
        header.getCellByIndex(1).setStringValue("Attribute");
        header.getCellByIndex(2).setStringValue("Type");
        header.getCellByIndex(3).setStringValue("SubType");
        header.getCellByIndex(4).setStringValue("Default Value");
        header.getCellByIndex(5).setStringValue("Input");
        header.getCellByIndex(6).setStringValue("Access");
        header.getCellByIndex(7).setStringValue("Comment");

        // Data rows
        Row row1 = table.getRowByIndex(1);
        row1.getCellByIndex(0).setStringValue("Person");
        row1.getCellByIndex(1).setStringValue("name");
        row1.getCellByIndex(2).setStringValue("string");
        row1.getCellByIndex(3).setStringValue("");
        row1.getCellByIndex(4).setStringValue("");
        row1.getCellByIndex(5).setStringValue("input");
        row1.getCellByIndex(6).setStringValue("rw");
        row1.getCellByIndex(7).setStringValue("Person's name");

        Row row2 = table.getRowByIndex(2);
        row2.getCellByIndex(0).setStringValue("Person");
        row2.getCellByIndex(1).setStringValue("age");
        row2.getCellByIndex(2).setStringValue("integer");
        row2.getCellByIndex(3).setStringValue("");
        row2.getCellByIndex(4).setStringValue("0");
        row2.getCellByIndex(5).setStringValue("input");
        row2.getCellByIndex(6).setStringValue("rw");
        row2.getCellByIndex(7).setStringValue("Person's age");

        doc.save(odsFile);
        doc.close();

        // Verify structure
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        // Check headers
        assertEquals("Entity", readTable.getCellByPosition(0, 0).getStringValue());
        assertEquals("Attribute", readTable.getCellByPosition(1, 0).getStringValue());
        assertEquals("Type", readTable.getCellByPosition(2, 0).getStringValue());

        // Check data
        assertEquals("Person", readTable.getCellByPosition(0, 1).getStringValue());
        assertEquals("name", readTable.getCellByPosition(1, 1).getStringValue());
        assertEquals("string", readTable.getCellByPosition(2, 1).getStringValue());

        assertEquals("Person", readTable.getCellByPosition(0, 2).getStringValue());
        assertEquals("age", readTable.getCellByPosition(1, 2).getStringValue());
        assertEquals("integer", readTable.getCellByPosition(2, 2).getStringValue());

        readDoc.close();
    }

    // ========== Decision Table Structure Tests (ODS) ==========

    @Test
    public void testODS_DecisionTableStructure() throws Exception {
        File odsFile = new File(tempDir.toFile(), "dt.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);
        table.setTableName("DecisionTable");

        // Name row
        table.getCellByPosition(0, 0).setStringValue("Name: Test_Decision_Table");

        // Type row
        table.getCellByPosition(0, 1).setStringValue("Type: All");

        // Blank row (row 2)

        // Conditions header
        table.getCellByPosition(0, 3).setStringValue("Conditions:");
        table.getCellByPosition(2, 3).setStringValue("Conditions");
        table.getCellByPosition(3, 3).setStringValue("1");
        table.getCellByPosition(4, 3).setStringValue("2");

        // Condition row
        table.getCellByPosition(0, 4).setStringValue("1");
        table.getCellByPosition(1, 4).setStringValue("Age check");
        table.getCellByPosition(2, 4).setStringValue("person.age > 18");
        table.getCellByPosition(3, 4).setStringValue("Y");
        table.getCellByPosition(4, 4).setStringValue("N");

        // Actions header
        table.getCellByPosition(0, 5).setStringValue("Actions:");

        // Action row
        table.getCellByPosition(0, 6).setStringValue("1");
        table.getCellByPosition(1, 6).setStringValue("Set adult");
        table.getCellByPosition(2, 6).setStringValue("person.isAdult = true");
        table.getCellByPosition(3, 6).setStringValue("X");

        doc.save(odsFile);
        doc.close();

        // Verify structure
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        // Check name
        String nameCell = readTable.getCellByPosition(0, 0).getStringValue();
        assertTrue("Should have Name:", nameCell.startsWith("Name:"));
        assertTrue("Should contain table name", nameCell.contains("Test_Decision_Table"));

        // Check conditions header
        assertEquals("Conditions:", readTable.getCellByPosition(0, 3).getStringValue());

        // Check condition content
        assertEquals("person.age > 18", readTable.getCellByPosition(2, 4).getStringValue());
        assertEquals("Y", readTable.getCellByPosition(3, 4).getStringValue());
        assertEquals("N", readTable.getCellByPosition(4, 4).getStringValue());

        // Check actions header
        assertEquals("Actions:", readTable.getCellByPosition(0, 5).getStringValue());

        // Check action content
        assertEquals("X", readTable.getCellByPosition(3, 6).getStringValue());

        readDoc.close();
    }

    // ========== Helper Method Tests ==========

    @Test
    public void testGetCellValue_Empty() throws Exception {
        File odsFile = new File(tempDir.toFile(), "empty.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);

        // Set only one cell
        table.getCellByPosition(0, 0).setStringValue("Only Cell");

        doc.save(odsFile);
        doc.close();

        // Test reading empty cells
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        assertEquals("Only Cell", readTable.getCellByPosition(0, 0).getStringValue());

        // These should return empty string or null
        Cell emptyCell = readTable.getCellByPosition(5, 5);
        String emptyValue = emptyCell.getStringValue();
        assertTrue("Empty cell should be null or empty",
            emptyValue == null || emptyValue.isEmpty());

        readDoc.close();
    }

    @Test
    public void testODS_SpecialCharacters() throws Exception {
        File odsFile = new File(tempDir.toFile(), "special.ods");

        SpreadsheetDocument doc = SpreadsheetDocument.newSpreadsheetDocument();
        Table table = doc.getSheetByIndex(0);

        Row row = table.getRowByIndex(0);
        row.getCellByIndex(0).setStringValue("Line1\nLine2");  // Newline
        row.getCellByIndex(1).setStringValue("Tab\there");     // Tab
        row.getCellByIndex(2).setStringValue("Quote\"test");   // Quote
        row.getCellByIndex(3).setStringValue("<xml>&amp;");    // XML chars

        doc.save(odsFile);
        doc.close();

        // Verify
        SpreadsheetDocument readDoc = SpreadsheetDocument.loadDocument(odsFile);
        Table readTable = readDoc.getSheetByIndex(0);

        assertEquals("Line1\nLine2", readTable.getCellByPosition(0, 0).getStringValue());
        assertEquals("Tab\there", readTable.getCellByPosition(1, 0).getStringValue());
        assertEquals("Quote\"test", readTable.getCellByPosition(2, 0).getStringValue());
        assertEquals("<xml>&amp;", readTable.getCellByPosition(3, 0).getStringValue());

        readDoc.close();
    }
}
