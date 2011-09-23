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

import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.nio.channels.FileChannel;
import java.util.Date;
import java.util.Iterator;
import java.util.List;

import com.dtrules.admin.RulesAdminService;
import com.dtrules.compiler.decisiontables.DTCompiler;
import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.decisiontables.DTNode.Coordinate;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.mapping.IMapGenerator;
import com.dtrules.mapping.MapGenerator;
import com.dtrules.mapping.MapGenerator2;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IDecisionTableError;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.testsupport.ChangeReport;

public class Excel2XML {
	public static PrintStream ostream = System.out;
	
    private RuleSet       ruleSet;
    private final String  UDTFilename = "Uncompiled_DecisionTables.xml";    
    String                path;
    String                rulesDirectoryXML;
    String                ruleset;
    DTCompiler            dtcompiler = null;
    boolean               verbose = true;
    /**
     * Returns the ruleSet being used by this compiler
     * @return
     */
    public RuleSet getRuleSet() {
        return ruleSet;
    }

    /**
     * We assume that the Rule Set provided gives us all the parameters needed
     * to open a set of Excel Files and convert them into XML files. 
     * @param propertyfile
     */
    public Excel2XML(RuleSet ruleSet) {
    	this.ruleSet = ruleSet;
    }

    /**
     * Opens the RulesDirectory for the caller.
     * @param rulesDirectoryXML
     * @param ruleset
     */
    public Excel2XML(String path, String rulesDirectoryXML, String ruleset){
        this.path               = path;
        this.rulesDirectoryXML  = rulesDirectoryXML;
        this.ruleset            = ruleset;
        reset();
    }

    /**
     * We reset the Rules Directory after we have modified the XML used to
     * define the Rules Directory.
     */
    public void reset (){
        RulesDirectory rd = new RulesDirectory(path,rulesDirectoryXML);
        this.ruleSet = rd.getRuleSet(ruleset);
    }
        
    /**
     * Return the Last DecisionTable Compiler used.  Null if none exists.
     *
     * @return Return the Last DecisionTable Compiler used.
     */
    public DTCompiler getDTCompiler(){
        return dtcompiler;
    }
    /**
     * Converts the RuleSet
     * @throws Exception
     */
    public void convertRuleset() throws Exception {    
        if(ruleSet == null){
            throw new Exception("The rule set '"+ruleset+"' could not be found");
        }
        
        if(new File(ruleSet.getWorkingdirectory()).mkdirs()){
            if(verbose) ostream.println("Created Directories "+ruleSet.getWorkingdirectory());
        }
        
        ImportRuleSets dt = new ImportRuleSets();
        dt.convertEDDs          (ruleSet, ruleSet.getExcel_edd(), ruleSet.getFilepath()+ruleSet.getEDD_XMLName());
        dt.convertDecisionTables(ruleSet, ruleSet.getFilepath()+UDTFilename);

        copyFile(ruleSet.getFilepath()+UDTFilename,ruleSet.getFilepath()+ruleSet.getDT_XMLName());
        reset();
    }   
    
    /**
     * Copy a file
     * Throws a runtime exception if anything goes wrong. 
     */
    public static void copyFile(String file1, String file2) {
        FileChannel inChannel  = null; 
        FileChannel outChannel = null;
        try{
            inChannel   = new FileInputStream(file1).getChannel();
            outChannel  = new FileOutputStream(file2).getChannel();
            try {
                // Stupid fix for Windows --  64Mb - 1024 bytes to avoid a bug. 
                int blksize = (64 * 1024 * 1024)-1024;
                long size = inChannel.size();
                long position = 0;
                while (position < size) {
                   position += inChannel.transferTo(position, blksize, outChannel);
                }
                inChannel.close();
                outChannel.close();
            } catch (IOException e) {
                throw new RuntimeException(e.getMessage());
            }
        }catch(FileNotFoundException e) {
            throw new RuntimeException(e);
        }   
    }

    /**
     * Compiles the RuleSet
     * @param NumErrorsToReport How many errors to print
     * @param err The output stream which to print the errors
     */
    @SuppressWarnings({ "deprecation" })
    public void compile(int NumErrorsToReport, PrintStream err) {
        
        try {
            IRSession         session       = new RSession(ruleSet); 
            Class<ICompiler>  compilerClass = ruleSet.getDefaultCompiler();
            if(compilerClass == null){
            	throw new RulesException("undefined", "Excel2XML", "No default compiler has been found." +
            			"  We cannot convert and compile the XML without one");
            }
            ICompiler defaultCompiler = (ICompiler) compilerClass.newInstance(); 
            defaultCompiler.setSession(session);

            ICompiler      compiler     = defaultCompiler;
                           dtcompiler   = new DTCompiler(compiler);
            
            InputStream    inDTStream   = new FileInputStream(ruleSet.getFilepath()+"/"+UDTFilename);
            OutputStream   outDTStream  = new FileOutputStream(ruleSet.getFilepath()+"/"+ruleSet.getDT_XMLName());
            
            dtcompiler.compile(inDTStream, outDTStream);
                        
            RulesDirectory  rd  = new RulesDirectory(path, rulesDirectoryXML);
            RuleSet         rs  = rd.getRuleSet(RName.getRName(ruleset));
            EntityFactory   ef  = rs.newSession().getEntityFactory();
            IREntity        dt  = ef.getDecisiontables();
            Iterator<RName> idt = ef.getDecisionTableRNameIterator();
            
            while(idt.hasNext()){
                RDecisionTable t = (RDecisionTable) dt.get(idt.next());
                t.build(session.getState());
                List<IDecisionTableError> errs = t.compile();
                for (IDecisionTableError error : errs){
                    dtcompiler.logError(
                            t.getName().stringValue(),
                            t.getFilename(),
                            "validity check", 
                            error.getMessage(), 
                            0, 
                            "In the "+error.getErrorType().name()+" row "+error.getIndex()+"\r\n"+error.getSource());
                }
                Coordinate err_RowCol = t.validate();
                if(!t.isCompiled()  || err_RowCol!=null){
                    int column = 0;
                    int row    = 0;
                    if(err_RowCol!=null){
                        row    = err_RowCol.getRow();
                        column = err_RowCol.getCol();
                    }
                    dtcompiler.logError(
                            t.getName().stringValue(),
                            t.getFilename(),
                            "validity check", 
                            "Decision Table did not compile", 
                            0, 
                            "A problem may have been found on row "+row+" and column "+column);
                }
            }
            
            if(verbose) {
                dtcompiler.printErrors(err, NumErrorsToReport);
                err.println("Total Errors Found: "+dtcompiler.getErrors().size());
            }
            if(dtcompiler.getErrors().size() == 0){
                rd  = new RulesDirectory(path, rulesDirectoryXML);
                rs  = rd.getRuleSet(RName.getRName(ruleset));
                PrintStream btables = new PrintStream(rs.getWorkingdirectory()+"balanced.txt");
                rs.newSession().printBalancedTables(btables);

                if(verbose) {
                    RulesAdminService admin = new RulesAdminService(rs.newSession(),rd);
                    List<?> tables = admin.getDecisionTables(rs.getName());
                    for(Object table : tables){
                       RDecisionTable dtable = admin.getDecisionTable(rs.getName(),(String)table);
                       dtable.check(ostream);
                    }
                }
                
            }
        } catch (Exception e) {
            err.print(e);
        }
        
    }
    /**
     * Generates a default Mapping File.  This file is not complete, but 
     * requires adjustments to match XML tags if they differ from the 
     * Attribute names used in the EDD.  Also, the initial Entity Stack and
     * frequency of Entity types must be examined and updated.
     * 
     * @param mapping	Tag for the mapping file.  Attributes with this tag in the
     * 					EDD will be mapped in the generated mapping
     * @param filename	File name for the mapping file generated
     * @throws Exception
     */
    @Deprecated
    public void generateMap(String mapping, String filename) throws Exception {
        RuleSet        rs   = getRuleSet();
        IMapGenerator   mgen = new MapGenerator();           
        mgen.generateMapping(
                mapping,
                rs.getFilepath()+rs.getEDD_XMLName(), 
                rs.getWorkingdirectory()+"map.xml");
    }
    
    /**
     * Generates a default Mapping File.  This file is not complete, but 
     * requires adjustments to match XML tags if they differ from the 
     * Attribute names used in the EDD.  Also, the initial Entity Stack and
     * frequency of Entity types must be examined and updated.
     * 
     * @param mapping   Version of EDD to map.
     * @param mapping   Tag for the mapping file.  Attributes with this tag in the
     *                  EDD will be mapped in the generated mapping
     * @param filename  File name for the mapping file generated
     * @throws Exception
     */
    public void generateMap(int version, String mapping, String filename) throws Exception {
        filename = filename.trim();
        RuleSet        rs   = getRuleSet();
        IMapGenerator   mgen;
        if(version == 1){
            mgen = new MapGenerator();
        }else{
            mgen = new MapGenerator2();
        }
        if(!filename.toLowerCase().endsWith(".xml")){
            filename += ".xml";
        }
        mgen.generateMapping(
                mapping,
                rs.getFilepath()+rs.getEDD_XMLName(),
                rs.getWorkingdirectory()+filename);
    }
    
    /**
     * Helper function that assumes no mappings are needed, and no changes to
     * the number of errors to report.
     * 
     * @param path
     * @param rulesConfig
     * @param ruleset
     * @param applicationRepositoryPath
     * @throws Exception
     */
    public static void compile (
            String path, 
            String rulesConfig, 
            String ruleset,
            String applicationRepositoryPath) throws Exception {
        compile(path,rulesConfig,ruleset,applicationRepositoryPath,null);
    }
    
    /**
     * Helper function that assumes you need mappings, but do not want to change
     * the default for the number of errors reported.
     * @param path
     * @param rulesConfig
     * @param ruleset
     * @param applicationRepositoryPath
     * @param mappings
     * @throws Exception
     */
    public static void compile (
            String path, 
            String rulesConfig, 
            String ruleset,
            String applicationRepositoryPath,
            String [] mappings) throws Exception {
    	compile(path, rulesConfig, ruleset,applicationRepositoryPath,mappings,80);
    }
    
    /**
     * Helper function for compiling Rule Sets.
     * @param path                  The Base Path in the file system.  All other files 
     *                              are defined as relative points away from this Base Path
     * @param rulesConfig           The name of the Rule Set configuration file
     * @param ruleset               The name of the Rule Set to compile. (Most of the parameters
     *                              to the compiler are pulled from the configuration file 
     * @param applicationRepositoryPath     This is a relative path to a possibly different
     *                              version of this Rule Set.  Generally, this is the version 
     *                              that is currently deployed.  The compile will produce 
     *                              a compare of changes between the Rule Set under development,
     *                              and this Rule Set.  If null, this comparison will be skipped.
     * @param errorcnt              Number of errors to be printed.
     * @throws Exception
     */
    public static void compile (
            String path, 
            String rulesConfig, 
            String ruleset,
            String applicationRepositoryPath,
            String [] mappings,
            int    errorcnt) {
        System.out.println("Starting: "+ new Date());
        Excel2XML excel2XML = new Excel2XML(path,rulesConfig,ruleset);
        //excel2XML.verbose = true;
        excel2XML.compileRuleSet(path, rulesConfig, ruleset, applicationRepositoryPath, mappings, errorcnt);
    }
        /**
         * Helper function for compiling Rule Sets.
         * @param path                  The Base Path in the file system.  All other files 
         *                              are defined as relative points away from this Base Path
         * @param rulesConfig           The name of the Rule Set configuration file
         * @param ruleset               The name of the Rule Set to compile. (Most of the parameters
         *                              to the compiler are pulled from the configuration file 
         * @param applicationRepositoryPath     This is a relative path to a possibly different
         *                              version of this Rule Set.  Generally, this is the version 
         *                              that is currently deployed.  The compile will produce 
         *                              a compare of changes between the Rule Set under development,
         *                              and this Rule Set.  If null, this comparison will be skipped.
         * @param errorcnt              Number of errors to be printed.
         * @throws Exception
         */
        public void compileRuleSet (
                String path, 
                String rulesConfig, 
                String ruleset,
                String applicationRepositoryPath,
                String [] mappings,
                int    errorcnt) {
        
        try{
            if(verbose) ostream.println("Starting: "+ new Date());
            if(verbose) ostream.println("Converting: "+ new Date());
            convertRuleset();
            if(verbose) ostream.println("Compiling: "+ new Date());
            compile(errorcnt,ostream);
            if(verbose) ostream.println("Done: "+ new Date());
            
            if(mappings != null) for(String map : mappings){
                generateMap(0, map, "mapping_"+map);
            }
            
            if(getDTCompiler().getErrors().size()==0 && applicationRepositoryPath != null){
                ChangeReport cr = new ChangeReport(
                        ruleset,
                        path,
                        rulesConfig,
                        "development",
                        applicationRepositoryPath,
                        rulesConfig,
                        "deployed");
                cr.compare(ostream);
                cr.compare(new FileOutputStream(getRuleSet().getWorkingdirectory()+"changes.xml"));   
            }
    
        } catch ( Exception ex ) {
            ostream.println("Failed to convert the Excel files");
            ostream.println(ex.toString());
            ex.printStackTrace();
        }
    }
}
