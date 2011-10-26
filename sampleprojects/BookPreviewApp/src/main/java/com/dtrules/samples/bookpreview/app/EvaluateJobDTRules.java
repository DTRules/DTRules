package com.dtrules.samples.bookpreview.app;

import java.io.FileOutputStream;
import java.io.OutputStream;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.DataMap;
import com.dtrules.mapping.Mapping;
import com.dtrules.samples.bookpreview.datamodel.DataObj;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;


public class EvaluateJobDTRules implements EvaluateJob {
	
	@Override
	public String getName() {
		return "dtrules";
	}
	
	String getJobName(DataObj request){
		int id = request.getId();
		String cnt = ""+id;
        for(;id<100000;id*=10)cnt = "0"+cnt;
        return "Request_"+cnt;
	}
	
	public String evaluate(int threadnum, BookPreviewApp app, DataObj request) {
            
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
	         DataMap 	datamap  = session.getDataMap(mapping,"BookPreview");
	         
	         request.write2DataMap(datamap);
	         
             int    id  = request.getId();
             
             if(app.save >0 && id % app.save == 0){
            	 String cnt = ""+id;
                 for(;id<100000;id*=10)cnt = "0"+cnt;
                 try {
                	 datamap.print(new FileOutputStream(app.getOutputDirectory()+"BP_request_"+cnt+".xml"));
                 } catch (Exception e) {
                	 System.out.println("Couldn't write to '"+app.getOutputDirectory()+"BP_request_"+cnt+".xml'");
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
        IREntity 	request  = session.getState().find("request").rEntityValue();
        IREntity    customer = request.get("customer").rEntityValue();
        
    }
}
