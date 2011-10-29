package com.dtrules.samples.bookpreview.app;

import com.dtrules.samples.bookpreview.datamodel.DataObj;


public class EvaluateJobNone implements EvaluateJob {
	
	@Override
	public String getName() {
		return "none";
	}	

	/**
	 * Do nothing; Ignore the job.  Deny all applicants.
	 * 
	 */
	public String evaluate(int threadnum, BookPreviewApp app, DataObj request) {
        Evaluate_Results(threadnum, app, request); 
		return null;
     }
    
	public void Evaluate_Results(int threadnum, BookPreviewApp app,  DataObj request){
				
	}

}
