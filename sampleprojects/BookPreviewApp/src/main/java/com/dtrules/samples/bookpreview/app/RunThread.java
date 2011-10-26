package com.dtrules.samples.bookpreview.app;

import com.dtrules.samples.bookpreview.datamodel.DataObj;


public class RunThread extends Thread {
	
	BookPreviewApp app;
	int      t;

	String getJobName(DataObj job){
		int id = job.getId();
		String cnt = ""+id;
        for(;id<100000;id*=10)cnt = "0"+cnt;
        return "Job_"+cnt;
	}
	
	public RunThread(int t, BookPreviewApp app){
		System.out.print("T "+t+" ");
		if(t%32==0)System.out.println();
		this.t   = t;
		this.app = app;
	}
	
	public void run () {
		DataObj job = app.next();
		while(job != null){
			String err=null;
	        if(app.db_delay!=0){
	            try {
	                Thread.sleep(app.db_delay);
	            } catch (InterruptedException e) { }
	        }
		    err = runfile(job);
			if(err != null)System.err.println(err);
			job = app.next();
		}
		synchronized (this) {
			System.out.print(t+"F ");
			app.threads--;
		}
	}
	
	/**
     * Returns the error if an error is thrown.  Otherwise, a null.
     * @param rd
     * @param rs
     * @param dfcnt
     * @param path
     * @param dataset
     * @return
     */
    public String runfile(DataObj job) {
   
    	return app.ejob.evaluate(t, app, job);
       
    }
    
}
