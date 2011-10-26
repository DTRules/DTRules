package com.dtrules.samples.chipeligibility.app;

import java.io.FileOutputStream;
import java.io.OutputStream;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.mapping.DataMap;
import com.dtrules.mapping.Mapping;
import com.dtrules.samples.chipeligibility.app.dataobjects.Case;
import com.dtrules.samples.chipeligibility.app.dataobjects.Client;
import com.dtrules.samples.chipeligibility.app.dataobjects.Income;
import com.dtrules.samples.chipeligibility.app.dataobjects.Job;
import com.dtrules.samples.chipeligibility.app.dataobjects.Relationship;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;


public class EvaluateJobNone implements EvaluateJob {
	
	@Override
	public String getName() {
		return "none";
	}	

	/**
	 * Do nothing; Ignore the job.  Deny all applicants.
	 * 
	 */
	public String evaluate(int threadnum, ChipApp app, Job job) {
        Evaluate_Results(threadnum, app, job); 
		return null;
     }
    
	public void Evaluate_Results(int threadnum, ChipApp app,  Job job){
		
		int denied=0;
		
		for (Client client : job.getCase().getClients()) if(client.getApplying()){
			denied++;		
		}
		
		app.update(threadnum, job.getCase().getClients().size(), 0, denied);
	}

}
