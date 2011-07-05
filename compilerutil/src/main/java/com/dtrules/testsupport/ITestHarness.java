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
import java.io.OutputStream;
import java.io.PrintStream;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.DataMap;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.XMLTree.Node;

/**
 * @author ps24876
 *
 */
public interface ITestHarness {
	
	/**
	 * Returns the TestHarness Version.  At this point we have the following 
	 * versions:
	 * <br><br>
	 * 1 -- Use the dataMap interface
	 * 2 -- Use the autoDataMap interface
	 *    
	 * @return
	 */
	int harnessVersion();
	
	/** AutoDataMap Support
	 * 
	 * Returns the entry point into the Rule Set.
	 * @return
	 */
	String entrypoint();
	
	/** AutoDataMap Support
	 * 
	 * Returns the map file name for this test.
	 * @return
	 */
	String mapName();
	
    /**
     * Returns the path to the Project Directory under which we organize the
     * test files, defined the Rule Sets, etc. for this project.
     * @return
     */
    String getPath();
    /**
     * Get the path to the Rules Directory Control File.
     */
    String getRulesDirectoryPath();
    /**
     * Defines the Rules Directory for this RuleSet.  Generally it will
     * be named "DTRules.xml" and located in the xml directory.  But not
     * always, depending on how Rules are deployed.
     * @return
     */
    String getRulesDirectoryFile();
    
    /**
     * This is the name of the Decision Table that has to be executed to evaluate
     * a data set against this rule set.
     * @return
     */
    String getDecisionTableName();
    
    /**
     * Get a list of decision tables to execute.
     * @return
     */
    String [] getDecisionTableNames();
    
    /**
     * Defines the name of the RuleSet we are testing
     */
    String getRuleSetName();
    
    /**
     * Called to execute the Decision Tables.  Some Test Harnesses might need
     * to make multiple Decision Table Calls
     */
    public void executeDecisionTables(IRSession session)throws RulesException;

    /**
     * Specifies a directory of test files.  Directories or files that are
     * not .xml files are ignored.  Each test file is executed, and a report
     * entry is made in the report file.
     * @return
     */
    String getTestDirectory();
   
    /**
     * This is where we are going to put the trace files, report files, etc.
     * @return
     */
    String getOutputDirectory();
    
    /**
     * This is where we are going to look for past results to compare
     * our new results to.
     * @return
     */
    String getResultDirectory();
    
    /**
     * Do you want to print the report data to the Console as well as to the
     * report file?  If so, this method should return true.
     * @return
     */
    boolean Console();
    /**
     * If verbose, we are going to print the EDD before we run the rules as 
     * well as after we run the rules.
     * @return
     */
    boolean Verbose();
    
    /**
     * If true, then the files generated will have a number attached to them.
     * This is important if the test run will execute a set of files multiple times,
     * and for whatever reason, you want all the results and traces of each run.   
     */
    boolean numbered();
    
    /**
     * If you want to trace a run, this function should return true;
     * @return
     */
    boolean Trace();
    
    /**
     * Provides a way for a project to manage how data is loaded into a 
     * session.
     * @param session
     * @param path
     * @param dataset
     * @throws Exception
     */
    public void loadData(IRSession session, String path, String dataset)throws Exception ;
    
    /**
     * Writes the currently loaded decision tables out to an Excel File,
     * if possible.  If not, a RuntimeException is thrown.
     * 
     * @param tables
     */
    public void writeDecisionTables(
            String  tables, 
            String  fields[], 
            boolean ascending, 
            int     limit) throws RuntimeException;
    /**
     * The name of the report file.
     * @return
     */
    String getReportFileName();
    
    /**
     * Runs all the test files in the TestDirectory;
     */
    void runTests();
    
    /**
     * Once the two XML results are returned, this test is used to check 
     * if they are the same.  If the two nodes are equal, this method must
     * return a null.  Anything else will be interpreted as being different,
     * and the String will be used to report that difference.
     * @param thisResult
     * @param oldResult
     * @return null if the nodes are equal, or a String detailing the differences
     * if the Nodes are not the same.
     */
    String compareNodes(Node thisResult, Node oldResult);
    
    /**
     * When we compare the results of one run of a set of tests against another,
     * this is where we write the report.
     * 
     * @return
     */
    PrintStream  compareTestResultsReport () throws Exception ;
    
    /**
     * Compare our new results with a set of past result files.
     */
    public void    compareTestResults() throws Exception ;

    
    /**
     * Print reports; Since the test harness can run multiple tests, the number of 
     * the test having been run is passed down to printReport.  That way, if it needs
     * to print additional reports or results, it can tag them with the appropriate
     * run number
     * @param runNumber the number of the test being run
     * @param session the session of the Rules Engine to use as the basis of the report
     * @param the result file opened by the TestHarness... Most tests need one, so we
     * open one for you.
     */
    void printReport(int runNumber, IRSession session, PrintStream out) throws Exception;
        
    public DataMap getDataMap();
    
    /**
     * Alternate Configuration File (generally the deployed rule set) to use as a 
     * basis to produce a change report XML 
     * @return
     */
    public String  referenceRulesDirectoryFile ();
    /**
     * Path to the Alternate Configuration File (generally the deployed rule set) to 
     * use as a basis to produce a change report XML 
     * @return
     */
    public String  referencePath ();
    /**
     * Generate a change report xml to the given output stream.
     * @param report
     */
    public void    changeReportXML(OutputStream report);

    /**
     * If true, and trace files are produced, then a coverage report
     * will be generated.
     * @return
     */
    public boolean coverageReport();
    
    /**
     * Returns the filename of the test set currently under test.
     * @return
     */
    public String getCurrentFile();

    /**
     * Returns the set of files to run as part of the test, in the order
     * provided.  By overriding this method, a test routine can be 
     * written to avoid certain files, repeat the execution of files, or
     * run files in a different order.
     * @return
     */
    public File [] getFiles();
    
    

}
