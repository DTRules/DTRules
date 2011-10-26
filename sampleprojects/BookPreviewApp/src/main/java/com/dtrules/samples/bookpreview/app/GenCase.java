package com.dtrules.samples.bookpreview.app;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.Random;

import com.dtrules.samples.bookpreview.TestCaseGen_BookPreview;
import com.dtrules.samples.bookpreview.datamodel.Request;

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
	
	Request generate(){
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
			Request request = generate();
			app.requests.add(request);
		}
	}
	
}
