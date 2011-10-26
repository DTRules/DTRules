package com.dtrules.samples.bookpreview.app;

import com.dtrules.samples.bookpreview.TestCaseGen_BookPreview;
import com.dtrules.samples.bookpreview.datamodel.DataObj;

public class GenCase  {

    int level;
    
    public void setLevel(int level){
        if (level < 100)level = 100;
        this.level = level;
    }
    
    BookPreviewApp app = null;
    
	GenCase(BookPreviewApp app){
		this.app = app;
	}
	
	DataObj generate(){
	    TestCaseGen_BookPreview gen = new TestCaseGen_BookPreview();
	    try{
	        return gen.generate();
	    }catch(Exception e){
	        return generate();
	    }
	}
	
	/**
	 * This method is going to watch the queue in the BookPreviewApp, and fill
	 * it with test cases until until full (i.e. has level many jobs in it).
	 */
	public void fill() {
		while(app.jobsWaiting()<level){
		    DataObj request = generate();
			app.jobs.add(request);
		}
	}
	
}
