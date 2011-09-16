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

import java.io.BufferedWriter;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.FileWriter;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.Writer;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.Iterator;

import org.apache.poi.hssf.usermodel.HSSFCell;
import org.apache.poi.hssf.usermodel.HSSFRichTextString;
import org.apache.poi.hssf.usermodel.HSSFRow;
import org.apache.poi.hssf.usermodel.HSSFSheet;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.apache.poi.poifs.filesystem.POIFSFileSystem;
import org.apache.poi.ss.usermodel.Cell;

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.RuleSet;
import com.dtrules.xmlparser.XMLPrinter;

public class ImportRuleSets {
	
    boolean CountsAreDirty;
    
    ArrayList<String> errors   = new ArrayList<String>();
    ArrayList<String> warnings = new ArrayList<String>();
    
    String tmpEDD = "tmpEDD.xml";
    
    String defaultColumns[]={"number","comments","dsl","table"};
    
    // These are the default columns for the decision tables.
    String columns[] = defaultColumns;

    /**
     * Return the column number for the given column name.
     * @param s
     * @return
     */
    int getColumn(String s){
       for(int i=0; i< columns.length && columns[i]!= null; i++){
           if(columns[i].equalsIgnoreCase(s)) return i;
       }
       return -1;
    }
  
    private void indent(StringBuffer buff, int depth){
    	for(int j=0;j<depth;j++)buff.append("  "); 
    }
    /** 
     * Convert all the Excel files in the given directory, and all sub
     * directories.  We return a string buffer of data on the decision tables
     * converted, if we find any.  Otherwise we return a null.
     * @param directory
     * @param sb
     * @return true if some file to convert was found.
     * @throws Exception
     */    
    private StringBuffer convertFiles(File directory,XMLPrinter out, int depth) throws Exception{
        boolean xlsFound = false;
        StringBuffer data = new StringBuffer();
        File[] files = directory.listFiles();
        for(int i=0; i < files.length; i++){
            if(files[i].isDirectory()){
            	indent(data, depth);
            	data.append(files[i].getName());
            	StringBuffer d2 = convertFiles(files[i],out,depth+1);
            	if(d2!=null){
            	   data.append(d2);
                   xlsFound = true;
                }   
            }else{
                if(files[i].getName().endsWith(".xls")){ 
                	indent(data, depth);
                	data.append(files[i].getName());
                	data.append("\r\n");
                	convertDecisionTable(data, files[i], out, depth+1);
                	xlsFound = true;
                }  
            }    
        }

        {
            int i = 1;
            for(String warning : warnings){
                System.err.println("WARNING["+i++ +"] "+warning);
            }
        }
        
        if(errors.size()>0){
            int i = 1;
            for(String error : errors){
                System.err.println("ERROR["+i++ +"] "+error);
            }
            System.err.println("Number of errors found: "+errors.size());
            throw new RuntimeException();
        }
        
        if(xlsFound)return data;
        return null;
    }
        
    /**
     * Convert all the excel files in the given directory, and all sub directories,
     * returning a String Buffer of all the XML produced.
     * @param directory
     * @param destinationFile
     * @return
     * @throws Exception
     */    
	public void convertDecisionTables(RuleSet ruleset ,String destinationFile) throws Exception{
	      OutputStream os = new FileOutputStream(destinationFile);
		  XMLPrinter out = new XMLPrinter("decision_tables", os);
		  String directory = ruleset.getExcel_dtfolder();
		  Iterator<String> includes = ruleset.getIncludedRuleSets().iterator();
		  while(directory != null){
		      directory = ruleset.getSystemPath()+"/"+directory;
		      StringBuffer conversion = convertFiles(new File(directory),out,0);
		      if(conversion != null){
		          System.out.print(conversion);
		      }else{
		          System.out.println("No Decision Tables Found");
		      }
		      if(includes.hasNext()){
		          String nextDirectory = includes.next();
		          directory = ruleset.getRulesDirectory().getRuleSet(nextDirectory).getExcel_dtfolder();
		      }else{
		          directory = null;
		      }
		  }
		  out.close();
		  os.close();
    }
    
	private String getCellValue(HSSFSheet sheet, int row, int column){
            if(row > sheet.getLastRowNum()) return ""; 
            HSSFRow  theRow = sheet.getRow(row);
            if(theRow==null)return "";
            HSSFCell cell = theRow.getCell(column);
            if(cell==null)return "";
            switch(cell.getCellType()){
                case HSSFCell.CELL_TYPE_BLANK :     return "";
                case HSSFCell.CELL_TYPE_BOOLEAN :   return cell.getBooleanCellValue()? "true": "false"; 
                case HSSFCell.CELL_TYPE_NUMERIC :{   
                    Double v = cell.getNumericCellValue();
                    if(v.doubleValue() == (v.longValue())){
                        return Long.toString(v.longValue());
                    }
                    return Double.toString(v);
                }
                case HSSFCell.CELL_TYPE_STRING :    
                    String v = cell.getRichStringCellValue().getString().trim();
                    return v;
                
                default :                           
                    return "";
            }        
    }
    /**
     * Looks for the value in some column, and returns that index.  This way we can be a bit more 
     * flexible in our format of the EDD.
     * @param value
     * @param sheet
     * @param row
     * @return  the Index of the value, or -1 if not found.
     */
    private int findvalue(String value, HSSFSheet sheet, int row){
        HSSFRow theRow = sheet.getRow(row);
        if(theRow==null)return -1;
        for(int i=0;i<theRow.getLastCellNum();i++){
            String v = getCellValue(sheet,row,i).trim();
            v=v.replaceAll(" ", "");
            if(v.equalsIgnoreCase(value))return i;
        }
        return -1;
    }
    /**
     * Converts a single file or an folder of files into a single EDD xml file.
     * @param excelName
     * @param outputXMLName
     * @throws Exception
     */
    public void convertEDDs(RuleSet rs, String excelName, String outputXMLName) throws Exception {
        
        EntityFactory    ef       = new EntityFactory(rs);
        Iterator<String> includes = rs.getIncludedRuleSets().iterator();
        
        while(excelName != null ){
            File excel = new File(rs.getSystemPath()+"/"+excelName);
            if(excel.isDirectory()){
                File files[] = excel.listFiles();
                for(File file : files){
                    String filename = file.getName().toLowerCase();
                    if( !file.isDirectory() && (filename.endsWith(".xls")||filename.endsWith(".xml"))){
                        convertEDD(ef, rs,file.getAbsolutePath());
                    }
                }
            }else{
                convertEDD(ef,rs,excelName);
            }
            if(includes.hasNext()){
                String includedSet = includes.next();
                excelName = rs.getRulesDirectory().getRuleSet(includedSet).getExcel_edd(); 
            }else{
                excelName = null;
            }
        }
        
        XMLPrinter   xptr  = new XMLPrinter(new FileOutputStream(outputXMLName));
        xptr.opentag("entity_data_dictionary","version","2","xmlns:xs","http://www.w3.org/2001/XMLSchema");
        ef.writeAttributes(xptr);
        xptr.close();
    }
    
    public void convertEDD(EntityFactory ef, RuleSet rs, String excelFileName ) throws Exception {
        InputStream  input = new FileInputStream(new File(excelFileName));
       
        // If the EDD is an XML file, We assume no conversion is necessary.
        if(excelFileName.endsWith(".xml")){
            ef.loadedd(rs.newSession(), excelFileName, input);
        	// Transfer bytes from in to out
            return;
            
        }else if(! (excelFileName.endsWith(".xls"))) throw new Exception("EDD Excel File name is invalid"); 
        // If the EDD is an Excel file, we need to convert the thing.        
       
        HSSFWorkbook wb = new HSSFWorkbook(input);
        HSSFSheet sheet = wb.getSheetAt(0);

        // Open the EDD.xml output file
        String     tmpEDDfilename = rs.getWorkingdirectory()+tmpEDD;
        XMLPrinter xout = new XMLPrinter(new FileOutputStream(tmpEDDfilename));
        
        // Write out a header in the EDD xml file.
        xout.opentag("edd_header");
           xout.printdata("edd_create_stamp",
                new SimpleDateFormat("EEE, d MMM yyyy HH:mm:ss Z").format(new Date())
           );
           xout.printdata("Excel_File_Name",excelFileName);
        xout.closetag();
        xout.opentag("edd");

        
        // Get the indexes of the columns we need to write out the XML for this EDD.
        int rows = sheet.getLastRowNum();
        int entityIndex    = findvalue("entity",sheet,0);
        int attributeIndex = findvalue("attribute",sheet,0);
        int typeIndex      = findvalue("type",sheet,0);
        int subtypeIndex   = findvalue("subtype",sheet,0);
        int defaultIndex   = findvalue("defaultvalue",sheet,0);
        int inputIndex     = findvalue("input",sheet,0);
        int accessIndex    = findvalue("access",sheet,0);
        int commentIndex   = findvalue("comment",sheet,0);      // optional
        int sourceIndex    = findvalue("source",sheet,0);       // optional
        
        // Some columns we just have to have.  Make sure we have them here.
        if(entityIndex <0 || attributeIndex < 0 || typeIndex < 0 || defaultIndex < 0 || accessIndex < 0 || inputIndex <0 ){
            String err = " Couldn't find the following column header(s): "+ 
              (entityIndex<0?" entity":"")+
              (attributeIndex<0?" attribute":"")+
              (typeIndex<0?" type":"")+ 
              (defaultIndex<0?" default value":"")+ 
              (accessIndex<0?" access":"")+
              (inputIndex<0?" input":"");
            throw new Exception("This EDD may not be valid, as we didn't find the proper column headers\n"+err);
        }
        
        // Go through each row, writing out each entry to the XML.
        for(int row = 1; row <=rows; row++){
            String entityname = getCellValue(sheet,row,entityIndex);    // Skip all the rows that have no Entity
            if(entityname.length()>0){
                
                String src     = sourceIndex>=0 ? getCellValue(sheet,row,sourceIndex):"";
                String comment = commentIndex>=0 ? getCellValue(sheet,row,commentIndex):"";
                xout.opentag("entry");
                xout.opentag("entity",
                        "entityname"        , entityname,
                        "attribute"         , getCellValue(sheet,row,attributeIndex),
                        "type"              , getCellValue(sheet,row,typeIndex),
                        "subtype"           , getCellValue(sheet,row,subtypeIndex),
                        "default"           , getCellValue(sheet,row,defaultIndex),
                        "access"            , getCellValue(sheet,row,accessIndex),
                        "input"             , getCellValue(sheet,row,inputIndex),
                        "comment"           , getCellValue(sheet,row,commentIndex) 
                );
                xout.closetag();
                if(comment.length()>0)xout.printdata("comment",getCellValue(sheet,row,commentIndex));
                if(src    .length()>0 )xout.printdata("source", getCellValue(sheet,row,sourceIndex));
                xout.closetag();                
            }
        }
        xout.closetag();
        xout.close();
        convertEDD(ef,rs, tmpEDDfilename);
    }
    /**
     * Pulls the ATTRIBUTE name out of the next cell.  The assumption is that
     * all ATTRIBUTES (Including the main sections of the decision table) are
     * all in the first column of a row, followed by a colon.
     * 
     * We ignore numeric Attribute names.
     * @param sheet
     * @param row
     * @return
     */
    private String getNextAttrib(HSSFSheet sheet, int row){
        String value      = getCellValue(sheet, row, 0).trim();
        int    colonIndex = value.indexOf(":");
        if(colonIndex>1){
           String attrib  = value.substring(0,colonIndex);
           attrib  = attrib.replaceAll(" ", "_");
           return attrib;
        }
        return "";
    }
   /**
    * Pulls the ATTRIBUTE Value out of the next cell.  If no attribute value
    * is found, then the value of column 0 is returned, whitespace trimmed off.
    * 
	* Ah, but we add a wrinkle.  If the value
    * we find in column 0 is a number, then we return the value from column
    * column 3.  
    *  
    * @param sheet
    * @param row
    * @return
    */
    private String getNextAttribValue(HSSFSheet sheet, int row){
        String value      = getCellValue(sheet, row, 0).trim();
        int    colonIndex = value.indexOf(":");
        if(colonIndex>1){
           value = value.substring(colonIndex+1).trim();
        }else {
            try{
            	Integer.parseInt(value);
            	value = getCellValue(sheet,row,2).trim();
            }catch(NumberFormatException e){};
        }
        return value;
    }
    /**
     * Returns the value of the number column.  You don't have to have one 
     * of these. 
     * 
     * @param sheet
     * @param row
     * @return
     */
    private String getNumber(HSSFSheet sheet, int row){        
        int field = getColumn("number");
        if(field==-1)return ""; 
        String value = getCellValue(sheet,row, field);
        return value.trim();
    }
    /**
     * Any Section can get the DSL specified in the section from this call. 
     * 
     * @param sheet
     * @param row
     * @return
     */
    private String getDSL(HSSFSheet sheet, int row){        
        int field = getColumn("dsl");
        if(field==-1)throw new RuntimeException("No DSL Column"); 
        String value = getCellValue(sheet,row, field).trim();
        return value;
    }
    /**
     * Get Policy Statement 
     * 
     * @param sheet
     * @param row
     * @return
     */
    private String getPolicyStatement(HSSFSheet sheet, int row){        
        int field = getColumn("comments");
        if(field==-1)throw new RuntimeException("No Comment or Policy Column"); 
        String value = getCellValue(sheet,row, field);
        return value.trim();
    }
    /**
     * Returns the contents of the Comment column... You don't have to have one
     * of these.
     *  
     * @param sheet
     * @param row
     * @return
     */
    private String getComments(HSSFSheet sheet, int row){
        int field = getColumn("comments");
        if(field==-1)return "";  
        String value = getCellValue(sheet,row, field);
        return value.trim();
    }
    
    /**
     * Clear a number field that shouldn't have any numbers.
     * @param sheet
     * @param row
     */
    void clearNumber(HSSFSheet sheet, int row){
        String numberFound = getNumber(sheet,row);
        int field = getColumn("number");
        if(field!= -1){
            if(sheet != null && sheet.getRow(row)!=null){
              sheet.getRow(row).createCell(field).setCellValue("");
            }
        }
    }
    
    
    /**
     * Prints the Context/InitialAction/Condition/Action number, does checks, prints
     * errors.  We MIGHT make it fix the errors...
     * @param out
     * @param sheet
     * @param row
     * @param label
     * @param count
     * @return Return an error message, or null.
     */
    private String printNumber(XMLPrinter out, HSSFSheet sheet, int row, String label, int count){
        String numberFound = getNumber(sheet, row);
        int v;
        String result = null;
        try {
            v = Integer.parseInt(numberFound);
        } catch (NumberFormatException e) {
            result = " Invalid number "+label+" on the "+count;
            v = count;
            int field = getColumn("number");
            if(field!=-1){
               sheet.getRow(row).createCell(field, HSSFCell.CELL_TYPE_NUMERIC);
               sheet.getRow(row).getCell(field).setCellValue((double)count);
            }
            CountsAreDirty = true;
        }
        if(v != count){
            result = " Incorrect Count "+label+" on the "+count+".  Found "+numberFound;
            int field =  getColumn("number");
            if(field!=-1){
               sheet.getRow(row).getCell(field).setCellValue((double)count);
            }
            CountsAreDirty = true;
        }
        out.printdata(label, numberFound);
        return result;
    }
    
    /**
     * Returns the Table value
     * of these.
     *  
     * @param sheet
     * @param row
     * @return
     */
    private String getTableValue(HSSFSheet sheet, int row, int tableIndex){
        int field = getColumn("table");
        if(field==-1)return "";  
        String value = getCellValue(sheet,row, field+tableIndex);
        return value.trim();
    }
    
    /**
     * Returns the value of the Requirement Reference column... You don't have to have one
     * of these.
     *  
     * @param sheet
     * @param row
     * @return
     */
    private String getRequirement(HSSFSheet sheet, int row){
        int field = getColumn("requirement");
        if(field==-1)return "";  
        String value = getCellValue(sheet,row, field);
        return value.trim();
    }
    
    
    /**
     * Returns the index of the heading of the next block.
     * @param sheet
     * @param row
     * @return
     */
    int nextBlock(HSSFSheet sheet, int row){
        String attrib = getNextAttrib(sheet, row);
        if(sheet.getRow(row)==null){
            return row;
        }
        Cell   c      = sheet.getRow(row).getCell(0);
        while(attrib.equals("") && c.getCellType()!= HSSFCell.CELL_TYPE_FORMULA){
            row++;
            attrib = getNextAttrib(sheet, row);
            if(row > sheet.getLastRowNum()) return row-1;
            c      = sheet.getRow(row).getCell(0);
        }
        return row;
    }
    
    /**
     * Reads the decision table out of an Excel spreadsheet and generates the
     * approriate XML. 
     * @param file
     * @param sb
     * @return true if at least one decision table was found in this file
     * @throws Exception
     */
	public boolean convertDecisionTable(StringBuffer data, File file,XMLPrinter out, int depth) throws Exception{
		if(! (file.getName().endsWith(".xls"))) return false; 
		
		InputStream input = new FileInputStream(file.getAbsolutePath());
        POIFSFileSystem fs = new POIFSFileSystem( input );
        HSSFWorkbook wb = new HSSFWorkbook(fs);
        boolean tablefound = false;
        CountsAreDirty = false;
        for(int i=0; i< wb.getNumberOfSheets(); i++){
            tablefound |= convertOneSheet(data, file.getName(),wb.getSheetAt(i),out,depth);
        }
        if(CountsAreDirty == true){
            System.out.println("Line Numbers on Contexts, Initial Actions, Conditions, and/or Actions are incorrect.\r\n" +
            		"A Corrected version has been written to the decision table directory");
            OutputStream output = new FileOutputStream(file.getAbsolutePath()+".fixedCounts");
            wb.write(output);
        }else{
            (new File(file.getAbsolutePath()+".fixedCounts")).delete();
        }
        return tablefound;
        
	}   
	
	String currentDT = "";
	
    /**
     * Returns true if the given sheet describes a valid DecisionTable.
     *
     * @param filename
     * @param sheet
     * @param sb
     * @return
     */
    private boolean convertOneSheet(
            StringBuffer    data, 
            String          filename, 
            HSSFSheet       sheet,
            XMLPrinter      out, 
            int             depth    ) throws Exception{    
        columns = defaultColumns;
        
        // The first row of a decision table has to provide the decision table name.  This is required!
        // Must be the first row, must have a NAME: tag, followed by a decision table name!
        String attrib = getNextAttrib(sheet, 0);
        String value  = getNextAttribValue(sheet, 0);
        if(!attrib.equalsIgnoreCase("name") || value.length()==0){
        	return false;
        }
        out.opentag("decision_table");
        
        String dtName = value.replaceAll("[\\s]+", "_");
        currentDT = dtName;
        
        indent(data,depth);
        data.append(dtName);
        data.append("\r\n");
        
        ArrayList<String> attributes = new ArrayList<String>();
        
        int rowIndex = 1;
        // Go through the attribute rows, identified by some tag followed by a colon.  When we reach a heading 
        // tag of CONDITIONS: then we stop!  These headers we preserve in the resulting XML.
        for(rowIndex=1;true;rowIndex++){
            
          attrib = getNextAttrib(sheet, rowIndex);
          value  = getNextAttribValue(sheet, rowIndex);  
          
          // Once we transitioned to the table upon seeing the CONDITIONS: tag.  We are
          // grandfathering that into our parsing.  Any other segment (Context or Initial Actions)
          // requires a blank line to step the processing on.  First optional segment is Contexts:,
          // followed by Initial_Actions:, followed by the Condition table.
          if( attrib.equalsIgnoreCase("conditions") || (attrib.length()==0 && value.length()==0)){  
              break;
          }
          String v = value.trim();
          attributes.add(attrib); 
          attributes.add(v);
          
          // If we have a columns attribute, then the column numbers and order are specfied there.
          // COLUMNS:  number, comment, requirements, dsl, table                                                                        
          if(attrib.equalsIgnoreCase("columns")){
              v = v.substring(2).trim();
              columns = v.split("[,\\s]+");
          }
          
        }  
        out.printdata("table_name",dtName);
        out.printdata("xls_file",filename);
        out.opentag("attribute_fields");
          for(int i=0; i< attributes.size(); i+=2){
              out.printdata(attributes.get(i),attributes.get(i+1));
          }
        out.closetag();
  
        rowIndex = nextBlock(sheet, rowIndex);
        attrib = getNextAttrib(sheet, rowIndex);
      
        if(attrib.equalsIgnoreCase("contexts")){
            rowIndex++;
            out.opentag("contexts");
            int contextCount = 1;
            while(true){
                attrib           = getNextAttrib(sheet, rowIndex);
                if(attrib.length()>0)break;   
                String context   = getDSL(sheet, rowIndex);
                if(!context.equals("") ) 
                {
                    out.opentag("context_details");
                    String err = printNumber(out,sheet,rowIndex,"context_number",contextCount++);
                    if(err!=null){
                        warnings.add(dtName+" : "+err);
                    }
                    String contextComment = getComments(sheet, rowIndex);
                    out.printdata("context_comment",contextComment);
                    out.printdata("context_description",context);
                    out.closetag();
                }else{  
                    clearNumber(sheet,rowIndex);
                }
                rowIndex++;
            }
            out.closetag();
        }
     
        rowIndex = nextBlock(sheet, rowIndex);
        attrib = getNextAttrib(sheet, rowIndex);
        
        if(attrib.equalsIgnoreCase("initial_actions")){
            rowIndex++;
            out.opentag("initial_actions");
            int iactionCount = 1;
            while(isAction(sheet,rowIndex)){  
                String initialActionDescription = getDSL(sheet, rowIndex); 
    
                if(!initialActionDescription.equals("")) 
                {
                    out.opentag("initial_action_details");
                    String err = printNumber(out,sheet,rowIndex,"intial_action_number",iactionCount++);
                    if(err!=null){
                        warnings.add(dtName+" : "+err);

                    }

                    String initialActionComment = getComments(sheet, rowIndex);
                    out.printdata("initial_action_comment",initialActionComment);
    
                    String requirements         = getRequirement(sheet, rowIndex);
                    out.printdata("initial_action_requirement", requirements);
                    
                    out.printdata("initial_action_description",initialActionDescription);
                    out.closetag();
                }else{  
                    clearNumber(sheet,rowIndex);
                }
                rowIndex++;
            }
            out.closetag();
        }
        
        rowIndex = nextBlock(sheet, rowIndex);
        attrib = getNextAttrib(sheet, rowIndex);
        rowIndex++;
        out.opentag("conditions");
        int conditionCount = 1;
        while(isCondition(sheet, rowIndex)){
            
        	String conditionDescription = getDSL(sheet, rowIndex); 

        	if(!conditionDescription.equals("")) {
        		out.opentag("condition_details");
                String err = printNumber(out,sheet,rowIndex,"condition_number",conditionCount++);
                if(err!=null){
                    warnings.add(dtName+" : "+err);

                }
	
	        	String conditionComment = getComments(sheet, rowIndex);
                out.printdata("condition_comment",conditionComment);
                
                String requirements         = getRequirement(sheet, rowIndex);
                out.printdata("condition_requirement", requirements);
                
                out.printdata("condition_description",conditionDescription);
	        	
	        	for(int j=0; j<16;j++){
	        		String columnValue =getTableValue(sheet, rowIndex, j).trim();  
	        		if(columnValue.equals("") || columnValue.equals(RDecisionTable.DASH)){
	        		    columnValue = RDecisionTable.DASH;
	        		}
	        		if ((columnValue.equalsIgnoreCase("*"))     ||
	        		    (columnValue.equalsIgnoreCase("y"))     || 
	        		    (columnValue.equalsIgnoreCase("n"))) {	               
	        		    out.printdata("condition_column","column_number",""+(j+1),"column_value",columnValue,null);
	        		}else if (columnValue.equalsIgnoreCase(RDecisionTable.DASH)){
	        		}else{
	        			if(!columnValue.equals("")){
		                  errors.add(dtName+": Undesired value in the condition matrix: '"+
                                  columnValue+"' Row: "+rowIndex);
	        			}
	        		}
	        	}
                out.closetag();
        	}else{ 
                clearNumber(sheet,rowIndex);
            }
        	rowIndex++;
        }
        out.closetag();
      
        rowIndex = nextBlock(sheet, rowIndex);
        attrib = getNextAttrib(sheet, rowIndex);
        rowIndex++;
                
        out.opentag("actions");
        int actionCount = 1;
        	while(isAction(sheet,rowIndex)){
        	    String actionDescription = getDSL(sheet, rowIndex);  
                if(actionDescription.length()>0){
                    out.opentag("action_details");
                    String err = printNumber(out,sheet,rowIndex,"action_number",actionCount++);
                    if(err!=null){
                        warnings.add(dtName+" : "+err);
                    }

                	String actionComment = getComments(sheet, rowIndex); 
                    out.printdata("action_comment",actionComment);
                    
                    String requirements         = getRequirement(sheet, rowIndex);
                    out.printdata("action_requirement", requirements);
                    
                    out.printdata("action_description",actionDescription);
                	
                	for(int j=0; j<16;j++){
                		String columnValue = getTableValue(sheet, rowIndex, j);  
                		if (columnValue.equalsIgnoreCase("x") ||
                			columnValue.equalsIgnoreCase("s")    ) {
                            out.printdata("action_column","column_number",""+(j+1),"column_value",columnValue,null);
                		}else{
                			if(columnValue.length() != 0){
        	                   errors.add(dtName+": Undesired value '"+columnValue+"' in the action matrix ("+j+","+rowIndex+")");
                			}  
                		}
                	}
                    out.closetag();
                }else{	
                    clearNumber(sheet,rowIndex);
                }    
            	rowIndex++;  
            	
        	}
            out.closetag();
            
            rowIndex = nextBlock(sheet, rowIndex);
            attrib = getNextAttrib(sheet, rowIndex);
            rowIndex++;
            
            out.opentag("policy_statements");
            int pscnt = 0;
                while(isPolicy(sheet,rowIndex)){
                    String policyStatement = getPolicyStatement(sheet, rowIndex);  
                    if(policyStatement.length()>0){
                        String ps_num = getNumber(sheet, rowIndex);
                        out.opentag("policy_statement","column",ps_num);
                        out.printdata("policy_description",policyStatement);
                        out.closetag();
                    }    
                    rowIndex++;  
                }
            out.closetag();
                        
            
        out.closetag();
        return true;
	}
	
	/**
	 * We used to do something really smart.  Now we just return false at the end of the spread
	 * sheet or if we encounter another block.
	 * @param sheet
	 * @param rowIndex
	 * @return
	 */
	private boolean isAction(HSSFSheet sheet, int rowIndex){
	     String attrib = getNextAttrib(sheet, rowIndex);
	     if (attrib.length()>0) return false;
		 if(rowIndex > sheet.getLastRowNum()) return false;
	     return true;
	}
	
	/**
     * We used to do something really smart.  Now we just return false at the end of the spread
     * sheet or if we encounter another block.
     * @param sheet
     * @param rowIndex
     * @return
     */
    private boolean isPolicy(HSSFSheet sheet, int rowIndex){
         String attrib = getNextAttrib(sheet, rowIndex);
         if (attrib.length()>0) return false;
         if(rowIndex > sheet.getLastRowNum()) return false;
         return true;
    }
	
    /**
     * We used to do something really smart.  Now we just return false at the end of the spread
     * sheet or if we encounter another block.
     * @param sheet
     * @param rowIndex
     * @return
     */
    private boolean isCondition(HSSFSheet sheet, int rowIndex){
         String attrib = getNextAttrib(sheet, rowIndex);
         if (attrib.length()>0) return false;
         if(rowIndex > sheet.getLastRowNum()) return false;
         return true;
    }
	
	public void setContents(File aFile, String aContents)
    throws FileNotFoundException, IOException {
		if (aFile == null) {
		throw new IllegalArgumentException("File should not be null.");
		}
		
		//declared here only to make visible to finally clause; generic reference
		Writer output = null;
		try {
		//use buffering
		//FileWriter always assumes default encoding is OK!
		output = new BufferedWriter( new FileWriter(aFile) );
		output.write( aContents );
		}
		finally {
		//flush and close both "output" and its underlying FileWriter
		if (output != null) output.close();
		}
	}
	

}
