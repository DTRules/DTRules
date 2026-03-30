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

package com.dtrules.testsupport;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

import com.dtrules.admin.RulesAdminService;
import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author ps24876
 *
 */
public class Coverage {
    
    String currentFile = "";
    
    public static class Stats {

        final RDecisionTable table;
             
        final int columnCount;
        final int columnHits[];
        final int conditionCount;
        final int conditions[];
        final int actionCount;
        final int actions[];
              int noColumns = 0;    // Count of how many times no columns were executed.
              int calledCount = 0;
        
        public Stats(RDecisionTable table,int columnCount,int conditionCount,int actionCount){
            this.table          = table;
            this.columnCount    = columnCount;
            columnHits          = new int[columnCount>0?columnCount:1];
            this.conditionCount = conditionCount;
            conditions          = new int[conditionCount];
            this.actionCount    = actionCount;
            actions             = new int[actionCount];
        }
    }
    
    RuleSet                 rs;
    String                  traceFiles; 
    ArrayList<String>       traceFilesProcessed = new ArrayList<String>();
    ArrayList<String>       minFilesNeeded      = new ArrayList<String>();
    
    HashMap<String,Stats>   tables = new HashMap<String,Stats>();
    
    int 					totalColumns = 0;
    
    public Coverage(RuleSet rs, String traceFiles)throws RulesException {
        this.rs         = rs;
        this.traceFiles = traceFiles;
        initTables();
    }
    
    public class TraceLoad implements IGenericXMLParser {

        ArrayList<String> dts = new ArrayList<String>();
        void   pushDT(String dt)    {dts.add(dt); }
        String popDT()              {return dts.remove(dts.size()-1);}
        String currentDT()          {return dts.get(dts.size()-1);}
        
        public void beginTag(String[] tagstk, int tagstkptr, String tag,
                HashMap<String, String> attribs) throws IOException, Exception {
            if(tag.equals("decisiontable")){
                pushDT(attribs.get("name"));
                Stats stats   = tables.get(currentDT());
                stats.calledCount++;
            }else if(tag.equals("column")){
                addColumns(attribs.get("n"));
            }
        }

        public void endTag(String[] tagstk, int tagstkptr, String tag,
                String body, HashMap<String, String> attribs) throws Exception,
                IOException {
            if(tag.equals("decisiontable")){
                popDT();
            }
        }
        
        private void addColumns(String columns){
           columns = columns.trim();
           
           Stats stats   = tables.get(currentDT());
           
           if(columns.length()==0){             // If no column is specified, no column
               if(stats.noColumns == 0){
                   if(!minFilesNeeded.contains(currentFile)){
                       minFilesNeeded.add(currentFile);
                   }
               }
               stats.noColumns++;               //   was executed.  Count that too.
               return;
           }
           
           String cols[] = columns.split(" ");
           for (String col : cols){
               int index = Integer.parseInt(col)-1;
               if(stats.columnHits[index]==0){
                   if(!minFilesNeeded.contains(currentFile)){
                       minFilesNeeded.add(currentFile);
                   }                   
               }
               stats.columnHits[index]++;
               totalColumns++;
           }
        }
        
        public boolean error(String v) throws Exception {
            return false;
        }
        
    }
    
    
    public void compute() throws Exception {
        File dir = new File(traceFiles);
        if(!dir.isDirectory()){
            coverage(dir);   // Maybe they only want coverage of a single file?
        }else{
            File files[] = dir.listFiles();
            for(File file : files){
                if(file.getName().endsWith("_trace.xml")){
                    System.out.println(file.getName());
                    traceFilesProcessed.add(file.getName());
                    coverage(file);
                }
            }
        }
    }
    
    @SuppressWarnings("unchecked")
	private void initTables() throws RulesException {
        IRSession session = rs.newSession();
        RulesAdminService admin = new RulesAdminService(session,rs.getRulesDirectory());
        List tables = admin.getDecisionTables(rs.getName());
        
        for(Object table : tables){
            RDecisionTable dt = admin.getDecisionTable(rs.getName(),(String)table);
            int columns     = dt.getConditiontable().length>0 ? dt.getConditiontable()[0].length:0;
            int conditions  = dt.getConditiontable().length;
            int actions     = dt.getActions().length;
            Stats stats = new Stats(dt,columns,conditions,actions);
            
            this.tables.put(dt.getName().stringValue(),stats);
        }
    }
    
    private void coverage(File f) throws Exception {
        currentFile = f.getName();
        GenericXMLParser.load(new FileInputStream(f),new TraceLoad());
    }
    
    public void printReport(PrintStream o){
        XMLPrinter xout = new XMLPrinter(o);
       
        xout.opentag("coverage","total_columns_executed", totalColumns);
       
        xout.opentag("minimum_files_for_coverage");
        for(String file : minFilesNeeded){
            xout.printdata("trace_file",file);
        }
        xout.closetag();
        
        xout.opentag("trace_files");
        for(String file : traceFilesProcessed){
            xout.printdata("trace_file",file);
        }
        xout.closetag();
        
        
        Object keys[] = tables.keySet().toArray();
        
        for(int i=0;i<keys.length-1;i++){
            for(int j=0;j<keys.length-1-i;j++){
                if(keys[j].toString().compareTo(keys[j+1].toString())>0){
                    Object hld = keys[j];
                    keys[j]    = keys[j+1];
                    keys[j+1]  = hld;
                }
            }
        }
        
        xout.opentag("tables");
        for(Object key : keys){
            Stats stats = tables.get(key);
            int total_columns = 0;
            int columns_covered = 0;
            if(stats.table.getHasNullColumn()){
               total_columns ++;
            }
            if(stats.noColumns>0){
                columns_covered++;
            }
            for(int i=0; i< stats.columnCount; i++){
                if(stats.table.getColumnsSpecified()[i]){
                    total_columns++;
                    if( stats.columnHits[i]>0){
                        columns_covered++;
                    }
                }
            }
         
            double fpercent = ((double) columns_covered / total_columns)*100.0;
            int    percent  = (int) fpercent;
            int    fraction = (int) ((fpercent - percent)*100);
            
            xout.opentag("table","name",key,"count_of_calls",stats.calledCount, "percent_covered",percent+"."+fraction);
            
            if(stats.table.getHasNullColumn()){
                xout.printdata("column","n","none","hits",stats.noColumns,null);
            }
            
            for(int i=0; i< stats.columnCount; i++){
                if(stats.table.getColumnsSpecified()[i]){
                    xout.printdata("column","n",i+1,"hits",stats.columnHits[i],null);
                }
            }
            xout.closetag();
        }
        xout.closetag();
        xout.closetag();
    }
    
    public static void main(String arg[]) throws Exception {
        String         path     = "C:\\maximus\\eb_dev2\\rulesDevelopment\\eb-newyork\\";
        RulesDirectory rd       = new RulesDirectory(path,"DTRules.xml");
        RName          rsName   = RName.getRName("autoassign");
        RuleSet        rs       = rd.getRuleSet(rsName);
        
        try {
            Coverage c = new Coverage(rs,path+"ny-autoassign\\testfiles\\output\\");
            c.compute();
            c.printReport(System.out);
        } catch (RulesException e) {
            System.out.println(e.toString());
        }
        
    }
    
}
