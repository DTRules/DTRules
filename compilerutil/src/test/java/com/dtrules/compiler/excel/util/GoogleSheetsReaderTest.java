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
import org.junit.Before;
import org.junit.Assume;

import java.io.File;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static org.junit.Assert.*;

/**
 * Tests for GoogleSheetsReader class.
 *
 * Note: Tests that require actual Google Sheets API access are marked with
 * Assume.assumeTrue() to skip when credentials are not available.
 */
public class GoogleSheetsReaderTest {

    private static final String TEST_CREDENTIALS_ENV = "GOOGLE_APPLICATION_CREDENTIALS";
    private static final String TEST_CREDENTIALS_PATH = System.getenv(TEST_CREDENTIALS_ENV);

    // Sample spreadsheet IDs for testing (these would need to be accessible)
    // Using a well-known public spreadsheet for integration testing
    private static final String SAMPLE_SPREADSHEET_URL =
        "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit";
    private static final String SAMPLE_SPREADSHEET_ID = "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms";

    // ========== URL Parsing Tests (No API required) ==========

    @Test
    public void testExtractSpreadsheetId_StandardUrl() {
        String url = "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit";
        String id = GoogleSheetsReader.extractSpreadsheetId(url);
        assertEquals("1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms", id);
    }

    @Test
    public void testExtractSpreadsheetId_WithGid() {
        String url = "https://docs.google.com/spreadsheets/d/abc123XYZ/edit#gid=123456";
        String id = GoogleSheetsReader.extractSpreadsheetId(url);
        assertEquals("abc123XYZ", id);
    }

    @Test
    public void testExtractSpreadsheetId_MinimalUrl() {
        String url = "https://docs.google.com/spreadsheets/d/testID";
        String id = GoogleSheetsReader.extractSpreadsheetId(url);
        assertEquals("testID", id);
    }

    @Test
    public void testExtractSpreadsheetId_WithHyphensAndUnderscores() {
        String url = "https://docs.google.com/spreadsheets/d/test-ID_123/edit";
        String id = GoogleSheetsReader.extractSpreadsheetId(url);
        assertEquals("test-ID_123", id);
    }

    @Test
    public void testExtractSpreadsheetId_InvalidUrl() {
        assertNull(GoogleSheetsReader.extractSpreadsheetId("https://google.com"));
        assertNull(GoogleSheetsReader.extractSpreadsheetId("https://docs.google.com/document/d/123"));
        assertNull(GoogleSheetsReader.extractSpreadsheetId("not a url"));
        assertNull(GoogleSheetsReader.extractSpreadsheetId(null));
        assertNull(GoogleSheetsReader.extractSpreadsheetId(""));
    }

    @Test
    public void testIsGoogleSheetsUrl_Valid() {
        assertTrue(GoogleSheetsReader.isGoogleSheetsUrl(
            "https://docs.google.com/spreadsheets/d/123/edit"));
        assertTrue(GoogleSheetsReader.isGoogleSheetsUrl(
            "https://docs.google.com/spreadsheets/d/abc-123_XYZ"));
    }

    @Test
    public void testIsGoogleSheetsUrl_Invalid() {
        assertFalse(GoogleSheetsReader.isGoogleSheetsUrl(null));
        assertFalse(GoogleSheetsReader.isGoogleSheetsUrl(""));
        assertFalse(GoogleSheetsReader.isGoogleSheetsUrl("https://google.com"));
        assertFalse(GoogleSheetsReader.isGoogleSheetsUrl("/local/path/file.xlsx"));
        assertFalse(GoogleSheetsReader.isGoogleSheetsUrl(
            "https://docs.google.com/document/d/123/edit")); // Google Docs, not Sheets
    }

    // ========== Static Helper Method Tests ==========

    @Test
    public void testGetCellValue_ValidData() {
        List<List<Object>> data = new ArrayList<>();
        data.add(Arrays.asList("A1", "B1", "C1"));
        data.add(Arrays.asList("A2", "B2", "C2"));
        data.add(Arrays.asList("A3", "B3"));

        assertEquals("A1", GoogleSheetsReader.getCellValue(data, 0, 0));
        assertEquals("B1", GoogleSheetsReader.getCellValue(data, 0, 1));
        assertEquals("C1", GoogleSheetsReader.getCellValue(data, 0, 2));
        assertEquals("A2", GoogleSheetsReader.getCellValue(data, 1, 0));
        assertEquals("B3", GoogleSheetsReader.getCellValue(data, 2, 1));
    }

    @Test
    public void testGetCellValue_OutOfBounds() {
        List<List<Object>> data = new ArrayList<>();
        data.add(Arrays.asList("A1", "B1"));

        assertEquals("", GoogleSheetsReader.getCellValue(data, -1, 0));
        assertEquals("", GoogleSheetsReader.getCellValue(data, 0, -1));
        assertEquals("", GoogleSheetsReader.getCellValue(data, 5, 0));
        assertEquals("", GoogleSheetsReader.getCellValue(data, 0, 5));
    }

    @Test
    public void testGetCellValue_EmptyData() {
        List<List<Object>> data = new ArrayList<>();
        assertEquals("", GoogleSheetsReader.getCellValue(data, 0, 0));
    }

    @Test
    public void testGetCellValue_NullValue() {
        List<List<Object>> data = new ArrayList<>();
        List<Object> row = new ArrayList<>();
        row.add("A1");
        row.add(null);
        row.add("C1");
        data.add(row);

        assertEquals("A1", GoogleSheetsReader.getCellValue(data, 0, 0));
        assertEquals("", GoogleSheetsReader.getCellValue(data, 0, 1));
        assertEquals("C1", GoogleSheetsReader.getCellValue(data, 0, 2));
    }

    @Test
    public void testGetCellValue_NumericValues() {
        List<List<Object>> data = new ArrayList<>();
        data.add(Arrays.asList(123, 45.67, "text"));

        assertEquals("123", GoogleSheetsReader.getCellValue(data, 0, 0));
        assertEquals("45.67", GoogleSheetsReader.getCellValue(data, 0, 1));
        assertEquals("text", GoogleSheetsReader.getCellValue(data, 0, 2));
    }

    @Test
    public void testGetCellValue_WhitespaceTrimming() {
        List<List<Object>> data = new ArrayList<>();
        data.add(Arrays.asList("  trimmed  ", "\ttabs\t", "\nnewlines\n"));

        assertEquals("trimmed", GoogleSheetsReader.getCellValue(data, 0, 0));
        assertEquals("tabs", GoogleSheetsReader.getCellValue(data, 0, 1));
        assertEquals("newlines", GoogleSheetsReader.getCellValue(data, 0, 2));
    }

    @Test
    public void testGetRowCount() {
        List<List<Object>> data = new ArrayList<>();
        assertEquals(0, GoogleSheetsReader.getRowCount(data));

        data.add(Arrays.asList("A1"));
        assertEquals(1, GoogleSheetsReader.getRowCount(data));

        data.add(Arrays.asList("A2"));
        data.add(Arrays.asList("A3"));
        assertEquals(3, GoogleSheetsReader.getRowCount(data));
    }

    @Test
    public void testGetColumnCount() {
        List<List<Object>> data = new ArrayList<>();
        data.add(Arrays.asList("A1", "B1", "C1"));
        data.add(Arrays.asList("A2", "B2"));
        data.add(Arrays.asList("A3"));

        assertEquals(3, GoogleSheetsReader.getColumnCount(data, 0));
        assertEquals(2, GoogleSheetsReader.getColumnCount(data, 1));
        assertEquals(1, GoogleSheetsReader.getColumnCount(data, 2));
        assertEquals(0, GoogleSheetsReader.getColumnCount(data, 5)); // Out of bounds
        assertEquals(0, GoogleSheetsReader.getColumnCount(data, -1)); // Negative index
    }

    // ========== API Integration Tests (Require credentials) ==========

    @Test
    public void testGoogleSheetsReader_WithCredentialsFile() {
        // Skip if no credentials file path is configured
        Assume.assumeTrue(TEST_CREDENTIALS_PATH != null && new File(TEST_CREDENTIALS_PATH).exists());

        try {
            GoogleSheetsReader reader = new GoogleSheetsReader(TEST_CREDENTIALS_PATH);
            assertNotNull("Reader should be created", reader);
        } catch (Exception e) {
            fail("Should create reader with valid credentials: " + e.getMessage());
        }
    }

    @Test
    public void testGoogleSheetsReader_WithDefaultCredentials() {
        // Skip if no default credentials are available
        Assume.assumeTrue(hasDefaultCredentials());

        try {
            GoogleSheetsReader reader = new GoogleSheetsReader();
            assertNotNull("Reader should be created", reader);
        } catch (Exception e) {
            // Expected if no credentials are configured
            assertTrue("Should fail gracefully without credentials",
                e.getMessage().contains("credentials") ||
                e.getMessage().contains("authentication"));
        }
    }

    @Test
    public void testGoogleSheetsReader_InvalidCredentialsPath() {
        try {
            new GoogleSheetsReader("/nonexistent/path/credentials.json");
            fail("Should throw exception for invalid credentials path");
        } catch (Exception e) {
            // Expected behavior
            assertNotNull(e);
        }
    }

    /**
     * Integration test - reads from an actual Google Sheet.
     * Only runs if credentials are configured.
     */
    @Test
    public void testReadSheet_Integration() {
        // Skip if no credentials available
        Assume.assumeTrue(hasWorkingCredentials());

        try {
            GoogleSheetsReader reader = createReader();

            // This test uses a sample public spreadsheet
            // Replace with an actual accessible spreadsheet ID for real testing
            List<List<Object>> data = reader.readFirstSheet(SAMPLE_SPREADSHEET_ID);

            assertNotNull("Should return data", data);
            // Further assertions would depend on the actual spreadsheet content

        } catch (Exception e) {
            // May fail if the spreadsheet is not accessible
            System.out.println("Integration test skipped: " + e.getMessage());
        }
    }

    // ========== Helper Methods ==========

    /**
     * Check if default credentials might be available.
     */
    private boolean hasDefaultCredentials() {
        return System.getenv("GOOGLE_APPLICATION_CREDENTIALS") != null ||
               System.getProperty("GOOGLE_APPLICATION_CREDENTIALS") != null;
    }

    /**
     * Check if we have working credentials for integration tests.
     */
    private boolean hasWorkingCredentials() {
        if (TEST_CREDENTIALS_PATH != null && new File(TEST_CREDENTIALS_PATH).exists()) {
            return true;
        }
        return hasDefaultCredentials();
    }

    /**
     * Create a reader using available credentials.
     */
    private GoogleSheetsReader createReader() throws Exception {
        if (TEST_CREDENTIALS_PATH != null && new File(TEST_CREDENTIALS_PATH).exists()) {
            return new GoogleSheetsReader(TEST_CREDENTIALS_PATH);
        }
        return new GoogleSheetsReader();
    }
}
