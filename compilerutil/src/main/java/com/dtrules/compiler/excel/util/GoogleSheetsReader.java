/**
 * Copyright 2004-2011 DTRules.com, Inc.
 *
 * See http://DTRules.com for updates and documentation for the DTRules Rules Engine
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

import com.google.api.client.googleapis.javanet.GoogleNetHttpTransport;
import com.google.api.client.http.HttpRequestInitializer;
import com.google.api.client.http.javanet.NetHttpTransport;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.gson.GsonFactory;
import com.google.api.services.sheets.v4.Sheets;
import com.google.api.services.sheets.v4.SheetsScopes;
import com.google.api.services.sheets.v4.model.Sheet;
import com.google.api.services.sheets.v4.model.Spreadsheet;
import com.google.api.services.sheets.v4.model.ValueRange;
import com.google.auth.http.HttpCredentialsAdapter;
import com.google.auth.oauth2.GoogleCredentials;
import com.google.auth.oauth2.ServiceAccountCredentials;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.util.Collections;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Utility class for reading Google Sheets spreadsheets.
 * Supports authentication via:
 * - Service account credentials (JSON key file)
 * - Application Default Credentials (ADC)
 *
 * Usage:
 * <pre>
 * // Using service account
 * GoogleSheetsReader reader = new GoogleSheetsReader("/path/to/credentials.json");
 *
 * // Using Application Default Credentials
 * GoogleSheetsReader reader = new GoogleSheetsReader();
 *
 * // Read data from a sheet
 * List<List<Object>> data = reader.readSheet("spreadsheetId", "Sheet1");
 * </pre>
 */
public class GoogleSheetsReader {

    private static final String APPLICATION_NAME = "DTRules Spreadsheet Converter";
    private static final JsonFactory JSON_FACTORY = GsonFactory.getDefaultInstance();

    // Pattern to extract spreadsheet ID from Google Sheets URL
    private static final Pattern SHEETS_URL_PATTERN = Pattern.compile(
        "https://docs\\.google\\.com/spreadsheets/d/([a-zA-Z0-9-_]+)"
    );

    private final Sheets sheetsService;

    /**
     * Create a GoogleSheetsReader using Application Default Credentials.
     * This works with:
     * - GOOGLE_APPLICATION_CREDENTIALS environment variable
     * - gcloud auth application-default login
     * - Google Cloud compute environment
     *
     * @throws IOException if credentials cannot be loaded
     * @throws GeneralSecurityException if HTTP transport cannot be created
     */
    public GoogleSheetsReader() throws IOException, GeneralSecurityException {
        GoogleCredentials credentials = GoogleCredentials.getApplicationDefault()
            .createScoped(Collections.singletonList(SheetsScopes.SPREADSHEETS_READONLY));
        this.sheetsService = createSheetsService(credentials);
    }

    /**
     * Create a GoogleSheetsReader using a service account credentials file.
     *
     * @param credentialsPath Path to the service account JSON key file
     * @throws IOException if credentials file cannot be read
     * @throws GeneralSecurityException if HTTP transport cannot be created
     */
    public GoogleSheetsReader(String credentialsPath) throws IOException, GeneralSecurityException {
        GoogleCredentials credentials = ServiceAccountCredentials
            .fromStream(new FileInputStream(credentialsPath))
            .createScoped(Collections.singletonList(SheetsScopes.SPREADSHEETS_READONLY));
        this.sheetsService = createSheetsService(credentials);
    }

    /**
     * Create a GoogleSheetsReader using pre-configured credentials.
     *
     * @param credentials Google credentials to use
     * @throws IOException if service cannot be created
     * @throws GeneralSecurityException if HTTP transport cannot be created
     */
    public GoogleSheetsReader(GoogleCredentials credentials) throws IOException, GeneralSecurityException {
        this.sheetsService = createSheetsService(credentials);
    }

    private Sheets createSheetsService(GoogleCredentials credentials)
            throws IOException, GeneralSecurityException {
        final NetHttpTransport httpTransport = GoogleNetHttpTransport.newTrustedTransport();
        HttpRequestInitializer requestInitializer = new HttpCredentialsAdapter(credentials);

        return new Sheets.Builder(httpTransport, JSON_FACTORY, requestInitializer)
            .setApplicationName(APPLICATION_NAME)
            .build();
    }

    /**
     * Extract the spreadsheet ID from a Google Sheets URL.
     *
     * @param url The Google Sheets URL (e.g., https://docs.google.com/spreadsheets/d/ABC123/edit)
     * @return The spreadsheet ID, or null if the URL is not a valid Google Sheets URL
     */
    public static String extractSpreadsheetId(String url) {
        if (url == null) return null;
        Matcher matcher = SHEETS_URL_PATTERN.matcher(url);
        if (matcher.find()) {
            return matcher.group(1);
        }
        return null;
    }

    /**
     * Check if a string is a Google Sheets URL.
     *
     * @param url The string to check
     * @return true if it's a Google Sheets URL
     */
    public static boolean isGoogleSheetsUrl(String url) {
        return url != null && SHEETS_URL_PATTERN.matcher(url).find();
    }

    /**
     * Get spreadsheet metadata including sheet names.
     *
     * @param spreadsheetId The spreadsheet ID
     * @return The Spreadsheet object containing metadata
     * @throws IOException if the API call fails
     */
    public Spreadsheet getSpreadsheet(String spreadsheetId) throws IOException {
        return sheetsService.spreadsheets()
            .get(spreadsheetId)
            .execute();
    }

    /**
     * Get the list of sheet names in a spreadsheet.
     *
     * @param spreadsheetId The spreadsheet ID
     * @return List of sheet names
     * @throws IOException if the API call fails
     */
    public List<Sheet> getSheets(String spreadsheetId) throws IOException {
        Spreadsheet spreadsheet = getSpreadsheet(spreadsheetId);
        return spreadsheet.getSheets();
    }

    /**
     * Read all data from a specific sheet.
     *
     * @param spreadsheetId The spreadsheet ID
     * @param sheetName The name of the sheet to read
     * @return List of rows, where each row is a list of cell values
     * @throws IOException if the API call fails
     */
    public List<List<Object>> readSheet(String spreadsheetId, String sheetName) throws IOException {
        ValueRange response = sheetsService.spreadsheets().values()
            .get(spreadsheetId, sheetName)
            .execute();
        List<List<Object>> values = response.getValues();
        return values != null ? values : Collections.emptyList();
    }

    /**
     * Read a specific range from a sheet.
     *
     * @param spreadsheetId The spreadsheet ID
     * @param range The A1 notation range (e.g., "Sheet1!A1:D10")
     * @return List of rows, where each row is a list of cell values
     * @throws IOException if the API call fails
     */
    public List<List<Object>> readRange(String spreadsheetId, String range) throws IOException {
        ValueRange response = sheetsService.spreadsheets().values()
            .get(spreadsheetId, range)
            .execute();
        List<List<Object>> values = response.getValues();
        return values != null ? values : Collections.emptyList();
    }

    /**
     * Read the first sheet of a spreadsheet.
     *
     * @param spreadsheetId The spreadsheet ID
     * @return List of rows, where each row is a list of cell values
     * @throws IOException if the API call fails
     */
    public List<List<Object>> readFirstSheet(String spreadsheetId) throws IOException {
        List<Sheet> sheets = getSheets(spreadsheetId);
        if (sheets.isEmpty()) {
            return Collections.emptyList();
        }
        String sheetName = sheets.get(0).getProperties().getTitle();
        return readSheet(spreadsheetId, sheetName);
    }

    /**
     * Get the number of sheets in a spreadsheet.
     *
     * @param spreadsheetId The spreadsheet ID
     * @return The number of sheets
     * @throws IOException if the API call fails
     */
    public int getSheetCount(String spreadsheetId) throws IOException {
        return getSheets(spreadsheetId).size();
    }

    /**
     * Get the name of a sheet by index.
     *
     * @param spreadsheetId The spreadsheet ID
     * @param index The sheet index (0-based)
     * @return The sheet name
     * @throws IOException if the API call fails
     * @throws IndexOutOfBoundsException if the index is out of range
     */
    public String getSheetName(String spreadsheetId, int index) throws IOException {
        List<Sheet> sheets = getSheets(spreadsheetId);
        return sheets.get(index).getProperties().getTitle();
    }

    /**
     * Get the cell value as a string.
     *
     * @param data The sheet data (list of rows)
     * @param row The row index (0-based)
     * @param column The column index (0-based)
     * @return The cell value as a string, or empty string if the cell is empty
     */
    public static String getCellValue(List<List<Object>> data, int row, int column) {
        if (row < 0 || row >= data.size()) {
            return "";
        }
        List<Object> rowData = data.get(row);
        if (column < 0 || column >= rowData.size()) {
            return "";
        }
        Object value = rowData.get(column);
        return value != null ? value.toString().trim() : "";
    }

    /**
     * Get the number of rows in the data.
     *
     * @param data The sheet data
     * @return The number of rows
     */
    public static int getRowCount(List<List<Object>> data) {
        return data.size();
    }

    /**
     * Get the number of columns in a specific row.
     *
     * @param data The sheet data
     * @param row The row index
     * @return The number of columns in that row
     */
    public static int getColumnCount(List<List<Object>> data, int row) {
        if (row < 0 || row >= data.size()) {
            return 0;
        }
        return data.get(row).size();
    }
}
