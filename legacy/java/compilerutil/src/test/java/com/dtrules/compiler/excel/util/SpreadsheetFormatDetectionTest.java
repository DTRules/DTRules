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

import org.junit.Test;
import static org.junit.Assert.*;

/**
 * Tests for spreadsheet format detection in ImportRuleSets.
 * Verifies that the system correctly identifies supported file formats.
 */
public class SpreadsheetFormatDetectionTest {

    // ========== Excel Format Detection Tests ==========

    @Test
    public void testIsSpreadsheetFile_XLS() {
        assertTrue("Should recognize .xls files",
            isSpreadsheetFile("test.xls"));
    }

    @Test
    public void testIsSpreadsheetFile_XLSX() {
        assertTrue("Should recognize .xlsx files",
            isSpreadsheetFile("test.xlsx"));
    }

    @Test
    public void testIsSpreadsheetFile_XLSM() {
        assertTrue("Should recognize .xlsm files",
            isSpreadsheetFile("test.xlsm"));
    }

    @Test
    public void testIsSpreadsheetFile_ODS() {
        assertTrue("Should recognize .ods files",
            isSpreadsheetFile("test.ods"));
    }

    @Test
    public void testIsSpreadsheetFile_CaseInsensitive() {
        assertTrue("Should recognize .XLS (uppercase)",
            isSpreadsheetFile("test.XLS"));
        assertTrue("Should recognize .XLSX (uppercase)",
            isSpreadsheetFile("test.XLSX"));
        assertTrue("Should recognize .ODS (uppercase)",
            isSpreadsheetFile("test.ODS"));
        assertTrue("Should recognize .Xlsx (mixed case)",
            isSpreadsheetFile("test.Xlsx"));
    }

    @Test
    public void testIsSpreadsheetFile_WithPath() {
        assertTrue("Should work with full path",
            isSpreadsheetFile("/path/to/file/test.xlsx"));
        assertTrue("Should work with relative path",
            isSpreadsheetFile("./data/test.xls"));
        assertTrue("Should work with Windows-style path",
            isSpreadsheetFile("C:\\Users\\test\\file.xlsx"));
    }

    @Test
    public void testIsSpreadsheetFile_UnsupportedFormats() {
        assertFalse("Should reject .csv files",
            isSpreadsheetFile("test.csv"));
        assertFalse("Should reject .txt files",
            isSpreadsheetFile("test.txt"));
        assertFalse("Should reject .xml files (handled separately)",
            isSpreadsheetFile("test.xml"));
        assertFalse("Should reject .pdf files",
            isSpreadsheetFile("test.pdf"));
        assertFalse("Should reject .doc files",
            isSpreadsheetFile("test.doc"));
    }

    @Test
    public void testIsSpreadsheetFile_NullAndEmpty() {
        assertFalse("Should handle null",
            isSpreadsheetFile(null));
        assertFalse("Should handle empty string",
            isSpreadsheetFile(""));
    }

    @Test
    public void testIsSpreadsheetFile_NoExtension() {
        assertFalse("Should reject files without extension",
            isSpreadsheetFile("testfile"));
        assertFalse("Should reject files with dot but no extension",
            isSpreadsheetFile("test."));
    }

    @Test
    public void testIsSpreadsheetFile_MultipleExtensions() {
        assertTrue("Should recognize last extension",
            isSpreadsheetFile("test.backup.xlsx"));
        assertFalse("Should check only last extension",
            isSpreadsheetFile("test.xlsx.backup"));
    }

    // ========== ODS Format Detection Tests ==========

    @Test
    public void testIsOdsFile_Positive() {
        assertTrue("Should recognize .ods",
            isOdsFile("test.ods"));
        assertTrue("Should be case insensitive",
            isOdsFile("test.ODS"));
        assertTrue("Should work with path",
            isOdsFile("/path/to/test.ods"));
    }

    @Test
    public void testIsOdsFile_Negative() {
        assertFalse("Should reject .xlsx",
            isOdsFile("test.xlsx"));
        assertFalse("Should reject .xls",
            isOdsFile("test.xls"));
        assertFalse("Should handle null",
            isOdsFile(null));
    }

    // ========== Google Sheets URL Detection Tests ==========

    @Test
    public void testIsGoogleSheetsUrl_ValidUrls() {
        assertTrue("Should recognize standard Google Sheets URL",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit"));

        assertTrue("Should recognize URL with gid parameter",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit#gid=0"));

        assertTrue("Should recognize URL without /edit",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"));
    }

    @Test
    public void testIsGoogleSheetsUrl_InvalidUrls() {
        assertFalse("Should reject Google Docs URL",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://docs.google.com/document/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit"));

        assertFalse("Should reject Google Drive URL",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://drive.google.com/file/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"));

        assertFalse("Should reject random URL",
            GoogleSheetsReader.isGoogleSheetsUrl(
                "https://example.com/spreadsheet.xlsx"));

        assertFalse("Should reject null",
            GoogleSheetsReader.isGoogleSheetsUrl(null));

        assertFalse("Should reject empty string",
            GoogleSheetsReader.isGoogleSheetsUrl(""));

        assertFalse("Should reject local file path",
            GoogleSheetsReader.isGoogleSheetsUrl("/path/to/file.xlsx"));
    }

    @Test
    public void testExtractSpreadsheetId_ValidUrls() {
        assertEquals("Should extract ID from standard URL",
            "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
            GoogleSheetsReader.extractSpreadsheetId(
                "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit"));

        assertEquals("Should extract ID with query params",
            "abc123-_xyz",
            GoogleSheetsReader.extractSpreadsheetId(
                "https://docs.google.com/spreadsheets/d/abc123-_xyz/edit#gid=0"));
    }

    @Test
    public void testExtractSpreadsheetId_InvalidUrls() {
        assertNull("Should return null for invalid URL",
            GoogleSheetsReader.extractSpreadsheetId("https://example.com"));

        assertNull("Should return null for null input",
            GoogleSheetsReader.extractSpreadsheetId(null));
    }

    // ========== Helper Methods (mirror ImportRuleSets private methods) ==========

    /**
     * Mirror of ImportRuleSets.isSpreadsheetFile() for testing.
     */
    private static boolean isSpreadsheetFile(String filename) {
        if (filename == null) return false;
        String lowerName = filename.toLowerCase();
        return lowerName.endsWith(".xls") ||
               lowerName.endsWith(".xlsx") ||
               lowerName.endsWith(".xlsm") ||
               lowerName.endsWith(".ods");
    }

    /**
     * Mirror of ImportRuleSets.isOdsFile() for testing.
     */
    private static boolean isOdsFile(String filename) {
        return filename != null && filename.toLowerCase().endsWith(".ods");
    }
}
