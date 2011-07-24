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


public class RunThread extends Thread {
	
	ChipApp app;
	int      t;

	String getJobName(Job job){
		int id = job.getId();
		String cnt = ""+id;
        for(;id<100000;id*=10)cnt = "0"+cnt;
        return "Job_"+cnt;
	}
	
	public RunThread(int t, ChipApp app){
		System.out.print("T "+t+" ");
		if(t%32==0)System.out.println();
		this.t   = t;
		this.app = app;
	}
	
	public void run () {
		Job job = app.next();
		while(job != null){
			String err=null;
		    err = runfile(job);
			if(err != null)System.err.println(err);
			job = app.next();
			try {
				sleep(0);
			} catch (InterruptedException e) {
				
			}
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
    public String runfile(Job job) {
   
    	if(app.dtrules){
    		return new EvaluateJobDTRules(t, app).evaluate(job);
    	}else{
    		return new EvaluateJobJava(t, app).evaluate(job);
    	}
       
    }
    
}
