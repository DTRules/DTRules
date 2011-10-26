package com.dtrules.samples.bookpreview.app;

import java.io.FileOutputStream;
import java.io.OutputStream;
import java.util.List;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.mapping.DataMap;
import com.dtrules.mapping.Mapping;
import com.dtrules.samples.bookpreview.datamodel.Book;
import com.dtrules.samples.bookpreview.datamodel.Chapter;
import com.dtrules.samples.bookpreview.datamodel.Customer;
import com.dtrules.samples.bookpreview.datamodel.Open_Book;
import com.dtrules.samples.bookpreview.datamodel.Publisher;
import com.dtrules.samples.bookpreview.datamodel.Request;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;


public class EvaluateJobDTRules implements EvaluateJob {
	
	@Override
	public String getName() {
		return "dtrules";
	}
	
	String getJobName(Request request){
		int id = request.getId();
		String cnt = ""+id;
        for(;id<100000;id*=10)cnt = "0"+cnt;
        return "Request_"+cnt;
	}
	
	public String evaluate(int threadnum, BookPreviewApp app, Request request) {
            
        try {
             IRSession      session    = app.rs.newSession();
             
             if(app.trace && (request.getId()%app.save == 0) ){
            	 OutputStream out = new FileOutputStream(
            			 app.getOutputDirectory()+getJobName(request)+"_trace.xml");
            	 session.getState().setOutput(out, System.out);
            	 session.getState().setState(DTState.TRACE | DTState.DEBUG);
            	 session.getState().traceStart();
             }
             
             // Map the data from the job into the Rules Engine
              
             // First create a data map.
             Mapping   	mapping  = session.getMapping();
	         DataMap 	datamap  = session.getDataMap(mapping,null);
	         
	         datamap.opentag(request,"request");{
	             datamap.readDO(request, "request");
	             
	             Book b = request.getBook();
	             datamap.opentag(b,"book"); {
	             datamap.readDO(b,"book");
     
                 Customer c = request.getCustomer();
	             datamap.opentag(c,"customer");
		         datamap.readDO(c,"customer");
		            datamap.opentag("open_books"); for(Open_Book ob : c.getOpen_books()){
		         	    List<Chapter> chapters = ob.getChapters_viewed();
		         	    datamap.opentag(client,"client");
		         		datamap.readDO(client,"client");
		         			for(Income income : client.getIncomes()){
		         				
		    	         		datamap.readDO(income,"income");
		    	         		
		         			}
		         		datamap.closetag();
		         	}datamap.closetag();
		         	for(Relationship r : c.getRelationships()){
		         		datamap.opentag("relationship");
			         		datamap.printdata("type", r.getType());
			         		datamap.readDO(r.getSource(), "source");
			         		datamap.readDO(r.getTarget(), "target");
			         	datamap.closetag();
		         	}
		         datamap.closetag();
		         
	         datamap.closetag();
	         
             int    id  = request.getId();
             
             if(app.save >0 && id % app.save == 0){
            	 String cnt = ""+id;
                 for(;id<100000;id*=10)cnt = "0"+cnt;
                 try {
                	 datamap.print(new FileOutputStream(app.getOutputDirectory()+"job_"+cnt+".xml"));
                 } catch (Exception e) {
                	 System.out.println("Couldn't write to '"+app.getOutputDirectory()+"job_"+cnt+".xml'");
                 }
             }
             
	         mapping.loadData(session, datamap);
	         
	         // Once the data is loaded, execute the rules.
             session.execute(app.getDecisionTableName());
		     
             if(app.trace && (request.getId()%app.save == 0)){
                 session.getState().traceEnd();
             }
             
             
             printReport(threadnum, app, session);
           
            
         } catch ( Exception ex ) {
             System.out.print("<-ERR  ");
             ex.printStackTrace();
             return "\nAn Error occurred while running the example:\n"+ex+"\n";
         }
         return null;
     }
    
    public void printReport(int threadnum, BookPreviewApp app, IRSession session) throws RulesException {
        IREntity 	job     = session.getState().find("job.job").rEntityValue();
        RArray 		results = job.get("job.results").rArrayValue();
        String      jobId   = job.get("id").stringValue();
        RArray      clients = session.getState().find("case.clients").rArrayValue();
      
        if(results.size()==0 && app.console){
        	System.out.println("No results for job "+jobId);
        }
        
        
        int approved 	= 0;
        int denied		= 0;
        for(IRObject r :results){
            IREntity result = r.rEntityValue();
            
            IREntity 	client 		= result.get("client").rEntityValue();
            String 		clinetId 	= client.get("id").stringValue();
            String 		answer      = "";
            if(result.get("eligible").booleanValue()){
                answer = "Approved";
                approved++;
                app.getApprovedClients().add(Integer.parseInt(clinetId));
            }else{
            	answer = "Denied";
            	denied++;
                app.getDeniedClients().add(Integer.parseInt(clinetId));
            }
            
            if(app.console){
	            String		notes		= "";
	            RArray		ns			= result.get("notes").rArrayValue();
	            for(IRObject n : ns){
	            	notes += "   "+n.stringValue() +"\n";
	            }
	            System.out.println();
	            System.out.println("Job "+jobId+" Client "+clinetId+" was "+answer+"\n"+notes);
            }
            
            RArray notes = result.get("notes").rArrayValue();
            synchronized (app) {
                for(IRObject n : notes){
                    String note = n.stringValue();
                    Integer v = app.results.get(note);
                    if(v == null) v = 0;
                    app.results.put(note,v+1);
                }
            }
        }
        
        app.update(threadnum, clients.size(), approved, denied);
        
    }
}
