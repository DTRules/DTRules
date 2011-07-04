package com.dtrules.SampleProject2.application;

import java.io.PrintStream;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.mapping.Mapping;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.XMLPrinter;

public class Application {
	
	public static String decisionTable 	= "Compute_Eligibility";
	public static String path    		= System.getProperty("user.dir")+"/";
	/**
	 * Runs the SampleProject2 rule set as a stand alone application.	  
	 * @param args
	 */
    public static void main(String[] args) {
    	try {
    		// First we need to get a Rules Directory.  The Rules Directory knows
    		// about all the Rule Sets we have defined within is configuration 
    		// file.  
    		//
    		// We supply the path to the configuration file, and the name of the
    		// configuration file.
    		//
    		// Normally, the configuration file is named DTRules.xml.  But 
    		// this is just a convention, and its name is supplied when creating
    		// the Rules Directory.
    		
    		RulesDirectory rd      = new RulesDirectory(
    				path+"repository/",			 
    				"DTRules.xml");
    		
    		// The RuleSet is built by loading the XML for the project.  This is
    		// done only once, and the results cached by the RulesDirectory.
    		RuleSet        rs      = rd.getRuleSet("SampleProject2");
    		
    		// A Session creates an instance of a Rules Set.  The Rules Engine is
    		// factored so that all the Rules Engine State is built off of the 
    		// DTState object in the Session.  The Rules Engine is Thread safe,
    		// so multiple threads can have sessions that run against the same
    		// Rule Set, and the only objects unique to a session is the DTState
    		// object and objects it holds.
    		IRSession      session = rs.newSession();
    		
    		// We are going to map the data into the session with the default mapping
    		// defined by the RuleSet.  Generally a Rule Set will use only one 
    		// mapping file.  However, you can build other mapping files.
    		Mapping        mapping = session.getMapping();
    		
    		// We are going to get our Data from an XML source.  The Mapping file
    		// and the XML Data source is all we need to populate the Rules 
    		// Session.
    		mapping.loadData(session, path+"testfiles/"+"TestCase_001.xml");
    		
    		// We will begin Execution at the main Decision Table for our Rule Set.
    		// Furthermore, we are going to only execute the Decision Tables once.
    		// You may, however, interact with the state of the Rules Engine, load
    		// more data, and execute any of the Decision Tables as needed.  This
    		// would allow one or more Rule Sets to manage the behavior of an 
    		// application in an on going mannor.
    		session.execute(decisionTable);
    
    		// Once the Decision Tables have executed, we need to extract the data
    		// from the Rules engine.  We will use a modified version of the 
    		// printReport() method from the SamplesProject2, the Rules Development
    		// Project for our application.
    		printReport(session, System.out);
    		
    	}catch(RulesException e){
    		// Should any error occur, print out the message.
    		System.out.println(e.toString());
    	}
	}

    /**
     * printReport produces an XML stream to the given PrintStream using the XML formatter
     * provided by DTRules.
     * @param session			The Rules Engine session
     * @param _out				The PrintStream we will write the XML stream to.
     * @throws RulesException
     */
    public static void printReport(IRSession session, PrintStream _out) throws RulesException {
        XMLPrinter xout = new XMLPrinter(_out);  	// Get an instance of the XML Formatter.
        
        DTState state   = session.getState();		// Get all the state information from our 
        										    //   Session.
        
        IREntity job     = state.findEntity("job"); // Get the Job Entity from the Entity Stack
        
        RArray   results = job.get("job.results").rArrayValue(); // Get the Rules Engine list
        											//   of result Entities
        											
        for(IRObject r :results){					// For each result entity 
            IREntity result = r.rEntityValue();		//   (which we will alias to an IREntity
            										//    pointer to cut down on type casting).

            xout.opentag("Client","id",result.get("client_id").stringValue()); // Output the
            										// client tag and the client ID
	            prt(xout,result,"totalGroupIncome");// Print the totalGroupIncome 
	            prt(xout,result,"client_fpl");		// Print the fpl percentage
	            
	            if(result.get("eligible").booleanValue()){	// Test their eligiblity result
	                xout.printdata("eligibility", "Approved");	// If approved, print the details of their			
	                prt(xout,result,"program");		//   approval.  
	                prt(xout,result,"programLevel");
	               
	            }else{
	            	xout.printdata("eligibility", "Not Approved"); // If not approved, print the 
	                prt(xout,result,"program");		//   details of their rejection.        
	            }
	            RArray notes = result.get("notes").rArrayValue();
	            xout.opentag("Notes");				// Print out any notes attached to the result.
	                for(IRObject n : notes){
	                   xout.printdata("note",n.stringValue());
	                }
	            xout.closetag();					// Close tags
            xout.closetag();
        }
        xout.close();								// Closes any remaining open tags.
    }
    
    /**
     * Helper function that pulls the given attribute from the given entity, and uses
     * the attribute name as the tag to write the data out to the XML stream.
     * @param xout
     * @param entity
     * @param attrib
     */
    private static void prt(XMLPrinter xout, IREntity entity, String attrib){
        IRObject value = entity.get(attrib);
        xout.printdata(attrib,value);
    }
    
}
