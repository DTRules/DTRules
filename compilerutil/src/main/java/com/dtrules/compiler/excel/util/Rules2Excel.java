package com.dtrules.compiler.excel.util;

import java.awt.font.FontRenderContext;
import java.awt.font.LineBreakMeasurer;
import java.awt.font.TextAttribute;
import java.io.FileOutputStream;
import java.text.AttributedString;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

import org.apache.poi.hssf.usermodel.HSSFFont;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.apache.poi.hssf.util.HSSFColor;
import org.apache.poi.ss.usermodel.Cell;
import org.apache.poi.ss.usermodel.CellStyle;
import org.apache.poi.ss.usermodel.Font;
import org.apache.poi.ss.usermodel.IndexedColors;
import org.apache.poi.ss.usermodel.Row;
import org.apache.poi.ss.usermodel.Sheet;
import org.apache.poi.ss.usermodel.Workbook;
import org.apache.poi.ss.util.CellRangeAddress;

import com.dtrules.admin.IRulesAdminService;
import com.dtrules.admin.RulesAdminService;
import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

public class Rules2Excel {

    boolean             balanced = false;
    int                 maxCol   = 16;
    
    IRulesAdminService  admin;
    RulesDirectory      rd;           
    RuleSet             rs;
    IRSession           session;
    
    String         tablefilename;
    
    Workbook       wb;
    int            sheetCnt    = 0;                     // The index of the "about to be created" sheet.
    
    CellStyle      cs_default;
    CellStyle      cs_title;
    CellStyle      cs_header;
    CellStyle      cs_num_header;
    CellStyle      cs_field;
    CellStyle      cs_comment;
    CellStyle      cs_formal;
    CellStyle      cs_table;
    CellStyle      cs_type;
    CellStyle      cs_number;
    
    
    public Rules2Excel(boolean balanced){
        this.balanced = balanced;
    }
    
    void newWb() {
        wb = new HSSFWorkbook();
        
        sheetCnt = 0;
        
        cs_default      = wb.createCellStyle();
        cs_title        = wb.createCellStyle();
        cs_header       = wb.createCellStyle();
        cs_num_header   = wb.createCellStyle();
        cs_field        = wb.createCellStyle();
        cs_comment      = wb.createCellStyle();
        cs_formal       = wb.createCellStyle();
        cs_table        = wb.createCellStyle();
        cs_type         = wb.createCellStyle();
        cs_number       = wb.createCellStyle();
        
        Font title_font = wb.createFont();
        title_font.setFontHeightInPoints((short)12);
        title_font.setColor(HSSFColor.DARK_BLUE.index);
        
        Font header_font = wb.createFont();
        header_font.setBoldweight(HSSFFont.BOLDWEIGHT_BOLD);
        header_font.setBoldweight(HSSFFont.BOLDWEIGHT_BOLD);
        
        Font table_font = wb.createFont();
        table_font.setBoldweight(HSSFFont.BOLDWEIGHT_BOLD);
        
        defaultcellstyle(cs_default);
        defaultcellstyle(cs_title);
        defaultcellstyle(cs_header);
        defaultcellstyle(cs_num_header);
        defaultcellstyle(cs_field);
        defaultcellstyle(cs_comment);
        defaultcellstyle(cs_formal);
        defaultcellstyle(cs_table);
        defaultcellstyle(cs_type);
        defaultcellstyle(cs_number);

        // Fonts
        cs_title.setFont(title_font);
        cs_header.setFont(header_font);
        cs_table.setFont(table_font);

        // Background
        cs_title.setVerticalAlignment(CellStyle.VERTICAL_CENTER);

        cs_header.setFillForegroundColor(HSSFColor.GREY_25_PERCENT.index);
        cs_header.setFillPattern(CellStyle.SOLID_FOREGROUND);

        cs_num_header.setFillForegroundColor(HSSFColor.GREY_25_PERCENT.index);
        cs_num_header.setFillPattern(CellStyle.SOLID_FOREGROUND);
        cs_num_header.setAlignment(CellStyle.ALIGN_CENTER);
        cs_num_header.setVerticalAlignment(CellStyle.VERTICAL_CENTER);

        cs_formal.setVerticalAlignment(CellStyle.VERTICAL_CENTER);
        cs_comment.setVerticalAlignment(CellStyle.VERTICAL_CENTER);

        cs_type.setFillForegroundColor(HSSFColor.LIGHT_YELLOW.index);
        cs_type.setFillPattern(CellStyle.SOLID_FOREGROUND);
        
        cs_number.setAlignment(CellStyle.ALIGN_CENTER);
        cs_number.setVerticalAlignment(CellStyle.VERTICAL_CENTER);
        
        cs_table.setAlignment(CellStyle.ALIGN_CENTER);
        cs_table.setVerticalAlignment(CellStyle.VERTICAL_CENTER);
        
    }

    private void defaultcellstyle(CellStyle cs){
        
        cs.setBorderBottom(CellStyle.BORDER_THIN);
        cs.setBottomBorderColor(IndexedColors.BLACK.getIndex());
        cs.setBorderTop(CellStyle.BORDER_THIN);
        cs.setTopBorderColor(IndexedColors.BLACK.getIndex());
        cs.setBorderLeft(CellStyle.BORDER_THIN);
        cs.setLeftBorderColor(IndexedColors.BLACK.getIndex());
        cs.setBorderRight(CellStyle.BORDER_THIN);
        cs.setRightBorderColor(IndexedColors.BLACK.getIndex());
        cs.setWrapText(true);
        cs.setVerticalAlignment(CellStyle.VERTICAL_CENTER);
        cs.setAlignment(CellStyle.ALIGN_LEFT);



    }
        
    private Sheet newDecisionTableSheet(){
        Sheet s         = wb.createSheet();             // Create the sheet for this decision table
        
        s.setColumnWidth(0,800);                        // Numbers for Contexts, Actions, Conditions, etc.
        s.setColumnWidth(1,12000);                      // Numbers for Contexts, Actions, Conditions, etc.
        s.setColumnWidth(2,12000);                      // Numbers for Contexts, Actions, Conditions, etc.
        
        for(int i = 0; i < maxCol; i++){
            s.setColumnWidth(i+3, 700);
        }
        
        Cell c;
        Row  r = s.createRow(0);
        r.setHeightInPoints(20);
        for(int i =  0; i <= maxCol+2; i++){
            c = r.createCell(i);
            c.setCellValue("");
            c.setCellStyle(cs_default);
        }
        
        return s;
    }
    
    
    Row nextRow(Sheet s, int cRow, int columnCnt){
        Row   r = s.createRow(cRow);
       
        r.setHeight((short)-1);
        
        for(int i =  0; i <= columnCnt; i++){
            Cell c = r.createCell(i);
            if(i==0){
                c.setCellStyle(cs_number);
            }else if(i>=3){
                c.setCellStyle(cs_table);
            }else{
                c.setCellStyle(cs_default);
            }
        }
        return r;
    }
    
    void setCell (Row r, Cell c, String text, int width){
        if(text == null) text = "";
        c.setCellValue(text);
        // Create Font object with Font attribute (e.g. Font family, Font size, etc)
        // for calculation
        if(text.length()>0){
            java.awt.Font currFont = new java.awt.Font("Arial", java.awt.Font.PLAIN, 10);
            AttributedString attrStr = new AttributedString(text);
            attrStr.addAttribute(TextAttribute.FONT, currFont);
    
            // Use LineBreakMeasurer to count number of lines needed for the text
            FontRenderContext frc = new FontRenderContext(null, true, true);
            LineBreakMeasurer measurer = new LineBreakMeasurer(attrStr.getIterator(),
            frc);
            int nextPos = 0;
            int lineCnt = 0;
            while (measurer.getPosition() < text.length())  {
                nextPos = measurer.nextOffset(width/43);            // mergedCellWidth is the max width of each line
                lineCnt++;
                measurer.setPosition(nextPos);
            }
            if(255*lineCnt > r.getHeight()){
                r.setHeight((short)(255 * lineCnt));
            }
        }
    }
    
    
    /**
     * Write out one Decision Table into one Sheet.
     * @param dt
     */
    private void writeDT(RDecisionTable dt){
        
        
        if(balanced){
            maxCol = dt.getActionTableBalanced(session).length>0 ? dt.getActionTableBalanced(session)[0].length : 0;
            if(maxCol <16) maxCol = 16;
        }else{
            maxCol = 16;
        }

        Sheet s         = newDecisionTableSheet();                   // Create the sheet for this decision table
        
        int   cRow      = 0;                                         // Our current row.
        
        cRow = writeName(dt, s, cRow);
        cRow = writeType(dt, s, cRow);
        cRow = writeFields(dt, s, cRow);
        cRow = writeContexts(dt, s, cRow);
        cRow = writeInitialActions(dt, s, cRow);
        cRow = writeConditions(dt, s, cRow);
        cRow = writeActions(dt, s, cRow);
        cRow = writePolicyStatements(dt, s, cRow);
    }
   
    int writeName(RDecisionTable dt, Sheet s, int cRow){    
        // Name
        String name     = dt.getName().stringValue();   // Get the name of the Decision Table
        
        name = name.replaceAll("_", " ");
          
        String sname    = name;
        
        if(sname.length()>30){
            sname = sname.substring(0,30)+sheetCnt;
        }
        
        boolean tryAgain    = true;
        int     cnt         = 0;
        String  n           = sname;
        while(tryAgain){
            try{
                wb.setSheetName(sheetCnt, sname);           // Set the sheet name, and increment the count.
                tryAgain = false;
            }catch(IllegalArgumentException e){
                sname = ++cnt + n; 
            }
        }
        
        sheetCnt++;
        Row  r = s.getRow(cRow);                        // Get the first row.
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
        setCell(r, c, "Name: "+name, 
                s.getColumnWidth(1)+
                s.getColumnWidth(2)+
                s.getColumnWidth(3)*maxCol);
        c.setCellStyle(cs_title);
        
        return cRow+1; 
    }        
        
    int writeType(RDecisionTable dt, Sheet s, int cRow){
        // Type
        Row  r = nextRow(s, cRow, maxCol+2);                         // Create a new row
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
        setCell(r, c, "Type: "+dt.getType(), 
                s.getColumnWidth(1)+
                s.getColumnWidth(2)+
                s.getColumnWidth(3)*maxCol);
        c.setCellStyle(cs_type);
        
        return cRow+1;
    }

    int writeFields(RDecisionTable dt, Sheet s, int cRow){

        //  Get the fields, sort them so they always come out in the same order
        Map<RName,String> fields        = dt.fields;
        RName []          fieldnames    = new RName[0];
        fieldnames = fields.keySet().toArray(fieldnames);
        for(int i=0; i < fieldnames.length-1; i++){
            for(int j=0; j < fieldnames.length-i-1;j++){
                try {
                    if(fieldnames[j].compare(fieldnames[j+1])>0){
                        RName hld       = fieldnames[j];
                        fieldnames[j]   = fieldnames[j+1];
                        fieldnames[j+1] = hld;
                    }
                } catch (RulesException e) { }      // Nothing to do on an error to compare...
            }
        }
        
        for(RName field : fieldnames){
            if(field.stringValue().equalsIgnoreCase("type")) continue;
            Row  r = nextRow(s, cRow, maxCol+2);                          // Create a new row
            Cell c = r.createCell(0);
            s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
            setCell(r, c, field.stringValue()+": "+fields.get(field), 
                    s.getColumnWidth(1)+
                    s.getColumnWidth(2)+
                    s.getColumnWidth(3)*maxCol);
            c.setCellStyle(cs_field);
            cRow++;
        }

        // Add a blank row
        Row r = nextRow(s, cRow, maxCol+2);                               // Create a new row
        Cell c = r.createCell(0);
        c.setCellStyle(cs_field);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
        return cRow+1;
    }
    
    int writeContexts(RDecisionTable dt, Sheet s, int cRow){

        //CONTEXTS:
        {
            Row  r = nextRow(s, cRow, maxCol+2);                          // Create a new row
            Cell c = r.createCell(0);
            s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
            c.setCellValue("Contexts:");
            c.setCellStyle(cs_header);
            cRow++;
    
            String contexts  [] = dt.getContexts();
            String ccontexts [] = dt.getContextsComment();
            for(int cnt = 1; cnt <= contexts.length; cnt++){
                r = nextRow(s, cRow, maxCol+2);                            // Create a new row
                c = r.createCell(0);
                c.setCellValue(cnt);
                c.setCellStyle(cs_number);
                
                c = r.createCell(1);
                c.setCellStyle(cs_comment);
                setCell(r,c, ccontexts[cnt-1],s.getColumnWidth(1));
                
                c = r.createCell(2);
                s.addMergedRegion(new CellRangeAddress(cRow,cRow,2,maxCol+2));
                c.setCellStyle(cs_formal);
                setCell(r,c, contexts[cnt-1], s.getColumnWidth(2)+s.getColumnWidth(3)*maxCol);
                
                cRow++;
            }
            r = nextRow(s, cRow, maxCol+2);                              // Create a new row
            s.addMergedRegion(new CellRangeAddress(cRow,cRow,2,maxCol+2));
        }
        
        return cRow+1;
    }
        
     
    int writeInitialActions(RDecisionTable dt, Sheet s, int cRow){

        Row  r = nextRow(s, cRow, maxCol+2);                             // Create a new row
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,1));
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,2,maxCol+2));
        c.setCellValue("Initial Actions:");
        c.setCellStyle(cs_header);

        c = r.createCell(2);
        c.setCellValue("Initial Actions");
        c.setCellStyle(cs_header);
        
        cRow++;

        String iactions  [] = dt.getInitialActions();
        String ciactions [] = dt.getInitialActionsComment();
        for(int cnt=1; cnt<=iactions.length; cnt++){
            r = nextRow(s, cRow, maxCol+2);                             // Create a new row
            c = r.createCell(0);
            c.setCellValue(cnt);
            c.setCellStyle(cs_number);
            
            c = r.createCell(1);
            c.setCellStyle(cs_comment);
            setCell(r,c,ciactions[cnt-1],s.getColumnWidth(1));
            
            c = r.createCell(2);
            s.addMergedRegion(new CellRangeAddress(cRow,cRow,2,maxCol+2));
            c.setCellStyle(cs_formal);
            setCell(r,c,iactions[cnt-1],s.getColumnWidth(2)+s.getColumnWidth(3)*maxCol);
            c.setCellValue(iactions[cnt-1]);
            
            cRow++;
        }
        r = nextRow(s, cRow, maxCol+2);                               // Create a new row
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,2,maxCol+2));

        return cRow+1;
    }
    
    int writeConditions(RDecisionTable dt, Sheet s, int cRow){

        Row  r = nextRow(s, cRow, maxCol+2);                             // Create a new row
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,1));
        c.setCellValue("Conditions:");
        c.setCellStyle(cs_header);

        for(int i = 3; i <= maxCol+2; i++){
            c = r.createCell(i);
            c.setCellValue(i-2);
            c.setCellStyle(cs_num_header);
        }
        
        c = r.createCell(2);
        c.setCellValue("Conditions");
        c.setCellStyle(cs_header);
        
        cRow++;

        int startRow = cRow;
        
        String conditions  [] = dt.getConditions();
        String cConditions [] = dt.getConditionsComment();
        for(int cnt=1; cnt <= conditions.length; cnt++){
            r = nextRow(s, cRow, maxCol+2);                             // Create a new row
            c = r.createCell(0);
            c.setCellValue(cnt);
            c.setCellStyle(cs_number);
            
            c = r.createCell(1);
            c.setCellStyle(cs_comment);
            setCell(r, c, cConditions[cnt-1], s.getColumnWidth(1));
            
            c = r.createCell(2);
            c.setCellStyle(cs_formal);
            setCell(r, c, conditions[cnt-1], s.getColumnWidth(2));
            
            cRow++;
        }
        r = nextRow(s, cRow, maxCol+2);                               // Create a new row
       
        String conditionTable[][] = balanced ? dt.getConditionTableBalanced(session) : dt.getConditiontable();
        
        for(int i = 0; i < conditionTable.length; i++){
            Row cr = s.getRow(startRow+i);
            for(int j = 0; j < conditionTable[i].length; j++){
                String v = "";
                if(conditionTable[i][j] != null && !conditionTable[i][j].equals("-")){
                    v = conditionTable[i][j];
                }
                Cell cell = cr.createCell(3+j);
                cell.setCellValue(v);
                cell.setCellStyle(cs_table);
            }
        }
        
        
        
        
        return cRow+1;
    }
    
    int writeActions(RDecisionTable dt, Sheet s, int cRow){

        Row  r = nextRow(s, cRow, maxCol+2);                             // Create a new row
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,1));
        c.setCellValue("Actions:");
        c.setCellStyle(cs_header);

        for(int i = 3; i <= maxCol+2; i++){
            c = r.createCell(i);
            c.setCellValue(i-2);
            c.setCellStyle(cs_num_header);
        }
        
        c = r.createCell(2);
        c.setCellValue("Actions");
        c.setCellStyle(cs_header);
        
        cRow++;

        int startRow = cRow;
        
        String actions  [] = dt.getActions();
        String cActions [] = dt.getActionsComment();
        for(int cnt=1; cnt <= actions.length; cnt++){
            r = nextRow(s, cRow, maxCol+2);                             // Create a new row
            c = r.createCell(0);
            c.setCellValue(cnt);
            c.setCellStyle(cs_number);
            
            c = r.createCell(1);
            c.setCellStyle(cs_comment);
            setCell(r, c, cActions[cnt-1], s.getColumnWidth(1));
            
            c = r.createCell(2);
            c.setCellStyle(cs_formal);
            setCell(r, c, actions[cnt-1], s.getColumnWidth(2));
            
            cRow++;
        }
        r = nextRow(s, cRow, maxCol+2);                               // Create a new row
       
        String actionTable[][] = balanced ? dt.getActionTableBalanced(session) : dt.getActiontable();
        for(int i = 0; i < actionTable.length; i++){
            Row cr = s.getRow(startRow+i);
            for(int j = 0; j < actionTable[i].length; j++){
                String v = "";
                if(actionTable[i][j] != null && !actionTable[i][j].equals("-")){
                    v = actionTable[i][j];
                }
                Cell cell = cr.createCell(3+j);
                cell.setCellValue(v);
                cell.setCellStyle(cs_table);
            }
        }
        
        return cRow+1;
    }
    
    int writePolicyStatements(RDecisionTable dt, Sheet s, int cRow){

        Row  r = nextRow(s, cRow, maxCol+2);                             // Create a new row
        Cell c = r.createCell(0);
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,0,maxCol+2));
        c.setCellValue("Policy Statements:");
        c.setCellStyle(cs_header);

        for(int i = 3; i <= maxCol+2; i++){
            c = r.createCell(i);
            c.setCellValue(i-2);
            c.setCellStyle(cs_num_header);
        }
                
        cRow++;
 
        String policystatements  [] = 
                balanced? dt.getPolicyStatementsBalanced(session) : dt.getPolicystatements();
        
        for(int cnt=1; cnt < policystatements.length; cnt++){
            r = nextRow(s, cRow, maxCol+2);                             // Create a new row
            s.addMergedRegion(new CellRangeAddress(cRow,cRow,1,maxCol+2));
            c = r.createCell(0);
            c.setCellValue(cnt);
            c.setCellStyle(cs_number);
            
            c = r.createCell(1);
            c.setCellStyle(cs_formal);
            setCell(r, c, policystatements[cnt], 
                    s.getColumnWidth(1)+
                    s.getColumnWidth(2)+
                    s.getColumnWidth(3)*maxCol);
                        
            cRow++;
        }
        r = nextRow(s, cRow, maxCol+2);                               // Create a new row
        s.addMergedRegion(new CellRangeAddress(cRow,cRow,1,maxCol+2));
        c = r.createCell(0);
        c.setCellStyle(cs_number);
        c = r.createCell(1);
        c.setCellStyle(cs_formal);
        
        return cRow+1;
    }
    
    
    /**
     * We by fields specified.  The first field is the primary key, followed by the secondary, etc.
     * @param dts
     * @param field
     * @param ascending
     */
    private void sort(ArrayList<RDecisionTable> dts, String fields[], boolean ascending){
        if(fields == null) return;
        for(int i = fields.length-1; i >=0 ; i--){
            sort(dts,fields[i],ascending);
        }
    }
    
    private void sort(ArrayList<RDecisionTable> dts, String field, boolean ascending){
        for(int i=0; i < dts.size()-1; i++){
            for(int j=0; j < dts.size()-1; j++){
                RDecisionTable a = dts.get(j);
                RDecisionTable b = dts.get(j+1);
                String af = a.getField(field);
                String bf = b.getField(field);
                if(af != null && bf != null && af.compareTo(bf) < 0 ^ ascending){
                    dts.set(j+1,a);
                    dts.set(j, b);
                }
            }
        }
    }

    
    @SuppressWarnings("unchecked")
    public void writeDecisionTables(String excelName, String fields[], boolean ascending, int limit){
        try{            
            List<?>                     decisiontables  = admin.getDecisionTables(rs.getName());
            ArrayList<RDecisionTable>   dts             = new ArrayList<RDecisionTable>();
            
            for(String dt : (List<String>)decisiontables){
                RDecisionTable rdt = session.getEntityFactory().findTable(dt);
                dts.add(rdt);
            }
            
            sort(dts,fields,ascending);
            
            int index   = 0;
            int filecnt = 1;
            
            while(index < dts.size()){
                newWb();
           
                for(int i = 0; i < limit && index < dts.size();i++,index++){
                    writeDT(dts.get(index));
                }
                String filename = excelName+"_"+filecnt++ +"_.xls";
                FileOutputStream excelfile = new FileOutputStream(filename);
                wb.write(excelfile);
                excelfile.close();
            }
            
            for(String dt : (List<String>)decisiontables){
                RDecisionTable rdt = session.getEntityFactory().findTable(dt);
                writeDT(rdt);
            }
                       
        }catch(Exception e){
            System.err.println("\n"+e.toString());
        }
    }    
    
    private Sheet newEDDSheet(){
        Sheet s         = wb.createSheet();             // Create the sheet for this decision table
        
        s.setColumnWidth(0,4000);                        // Entity
        s.setColumnWidth(1,4000);                        // Attribute
        s.setColumnWidth(2,2000);                        // Type
        s.setColumnWidth(3,3000);                        // SubType
        s.setColumnWidth(4,6000);                        // Default Value
        s.setColumnWidth(5,3000);                        // Input
        s.setColumnWidth(6,3000);                        // Access
        s.setColumnWidth(7,16000);                       // Comment

        Cell c;
        Row  r = s.createRow(0);
        r.setHeightInPoints(20);
        
        r.createCell(0).setCellValue("Entity");
        r.createCell(1).setCellValue("Attribute");
        r.createCell(2).setCellValue("Type");
        r.createCell(3).setCellValue("SubType");
        r.createCell(4).setCellValue("Default Value");
        r.createCell(5).setCellValue("Input");
        r.createCell(6).setCellValue("Access");
        r.createCell(7).setCellValue("Comment");

        cs_title.setAlignment(CellStyle.ALIGN_CENTER);
        cs_title.setVerticalAlignment(CellStyle.VERTICAL_CENTER);
        
        for(int i=0; i<8; i++){
            r.getCell(i).setCellStyle(cs_title);
        }
        
        return s;
    }
    
    int writeEntity(Sheet s, int rowC, RName entity ){
        IREntity e = session.getEntityFactory().findRefEntity(entity);
        
        ArrayList<RName> attributes = EntityFactory.sorted(e.getAttributeIterator(),true);
        for(RName a : attributes){
            Row          row   = nextRow(s, rowC, 8);
            REntityEntry entry = e.getEntry(a);
            
            row.getCell(0).setCellValue(e.getName().stringValue());      // Name
            row.getCell(1).setCellValue(a.stringValue());                // Attribute
            row.getCell(2).setCellValue(entry.type.toString());          // Type
            row.getCell(3).setCellValue(entry.subtype);                  // SubType
            row.getCell(4).setCellValue(entry.defaulttxt);               // Default
            row.getCell(5).setCellValue(entry.input);                    // Input
            row.getCell(6).setCellValue( (entry.readable? "r":"")  +     // Access
                                         (entry.writable ? "w":"") ); 
            
            setCell(row, row.getCell(7), entry.comment, 16000);      // Add the comment.  Wrap it.
            rowC++;    
        }
        
        
        return rowC;
    }
        
    void writeEDD(String excelName){
        try{            
            Sheet s = newEDDSheet();
            EntityFactory ef = session.getEntityFactory();
            
            ArrayList<RName> list    = EntityFactory.sorted(ef.getEntityRNameIterator(),true);
            int rowC = 1;
            
            for(RName entity : list){
                rowC = writeEntity(s, rowC, entity );
                nextRow(s, rowC, 8);
                rowC++;
            }
            
            FileOutputStream excelfile = new FileOutputStream(excelName);
            
            wb.write(excelfile);
            excelfile.close();
            
        }catch(Exception e){
            System.err.println("\n"+e.toString());
        }
    }

    
    public void writeExcel(
            IRulesAdminService  admin, 
            RuleSet             ruleset, 
            String              excelName, 
            String              fields[], 
            boolean             ascending,
            int                 limit ){
        try{
            this.admin    = admin;
            rd            = admin.getRulesDirectory();
            rs            = ruleset;
            session       = rs.newSession();
            
            writeDecisionTables(ruleset.getWorkingdirectory()+excelName,fields,ascending, limit);
            writeEDD(ruleset.getWorkingdirectory()+excelName+"_edd.xls");
        
        }catch(Exception e){
            System.err.println("\n"+e.toString());
        }
    }    

    
}
