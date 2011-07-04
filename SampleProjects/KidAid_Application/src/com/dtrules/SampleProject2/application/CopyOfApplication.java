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

public class CopyOfApplication {
	
	public static String decisionTable 	= "Compute_Eligibility";
	public static String path    		= System.getProperty("user.dir")+"/";
	/**
	 * Runs the SampleProject2 rule set as a stand alone application.	  
	 * @param args
	 */
    public static void main(String[] args) {
    	try {
    		
    		RulesDirectory rd      = new RulesDirectory(
    				path+"repository/",			 
    				"DTRules.xml");
    		
    		RuleSet        rs      = rd.getRuleSet("SampleProject2");
    		
    		IRSession      session = rs.newSession();
    		
    		Mapping        mapping = session.getMapping();
    		
    		mapping.loadData(session, path+"testfiles/"+"TestCase_001.xml");
    		
    		session.execute(decisionTable);
    
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
