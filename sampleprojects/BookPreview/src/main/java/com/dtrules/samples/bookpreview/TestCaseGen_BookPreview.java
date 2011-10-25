package com.dtrules.samples.bookpreview;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.PrintStream;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.Random;

import com.dtrules.xmlparser.XMLPrinter;

public class TestCaseGen_BookPreview {
	public static PrintStream ostream = System.out;
	public static PrintStream estream = System.err;
	
						      		// This is the default number of how many test cases to generate.
	static int cnt = 100;	    	// You can pass a different number on the commandline.
	
	Random 		       rand 		 = new Random(1013);
	XMLPrinter 	       xout 		 = null;
	String 	 		   path 		 = System.getProperty("user.dir")+"/testfiles/";
	
	SimpleDateFormat   sdf           = new SimpleDateFormat("MM/dd/yyyy");
	Date        	   currentdate   = new Date();
	int                id            = 1;
	
	/**
	 * Return a date that is some number of days before the given 
	 * date
	 */
	String getDate(long days){
	    Date d = new Date(currentdate.getTime()-(days*24l*60l*60l*1000l));
	    return sdf.format(d); 
	}
	
    int getId() { return id++; }
	
	int randint(int limit){
	    if(limit < 1) limit = 1;
		return Math.abs(rand.nextInt()%limit);
	}
	/**
	 * Flips a coin, returns true half the time, false half the time.
	 * @return
	 */
	boolean flip(){
		return (rand.nextInt()&1)>0;
	}
	/**
	 * Returns true for a given percent of the time.
	 * @param percent
	 * @return
	 */
	boolean chance(int percent){
		return randint(100)<=percent;
	}
	
	int genCustomer(int open_books){
		int customerId = id++;
		return customerId;
	}
		
	int numChapters;
	
	int genBook(int numPages ){
	    
	    int bookId = getId();
	    xout.opentag("book","id",bookId ); {
            
	        numChapters = 0;
            
            xout.opentag("chapters"); {
                for(int begin=0; begin < numPages; numChapters++){
                    xout.opentag("chapter"); {
                        int end = begin+randint(30)+20;
                        if(end+30 > numPages){
                            end = numPages;
                        }
                        xout.printdata("begin_page",begin);
                        xout.printdata("end_page"  ,end);
                        begin = end+1;
                    } xout.closetag();
                }
            }xout.closetag();
            
            xout.opentag("publisher");{
                xout.printdata("chapter_limit",randint(numChapters/5)+1);
                xout.printdata("page_limit", randint(numPages/50)+1);
            }xout.closetag();
                        
        } xout.closetag();
        
        return bookId;
	}
	
	void generate(){
		
        int openbooks = randint(20)+1;
        ArrayList<Integer> bookId   = new ArrayList<Integer>();
        ArrayList<Integer> pages    = new ArrayList<Integer>();
        ArrayList<Integer> chapters = new ArrayList<Integer>();

        xout.opentag("books");{
            for(int i=0; i<openbooks; i++){
                xout.opentag("book");{
                    int numPages    = randint(200)+100;  
                    bookId.add(genBook(numPages));
                    chapters.add(numChapters);
                    pages.add(numPages);
                }xout.closetag();
            }
        } xout.closetag();
        
		xout.opentag("request","id",getId()); {
    		xout.printdata("current_date",sdf.format(currentdate));
            xout.printdata("access","turn_page");

            int thebook     = randint(bookId.size());
            int numPages    = pages.get(thebook);
            int numChapters = chapters.get(thebook);
            
            xout.printdata("page_number",randint(numPages/10)+1);
            xout.printdata("book","id",bookId.get(thebook),null);
            
            xout.opentag("customer"); {
                for(int i=0; i<openbooks; i++){
                    xout.opentag("open_book"); {
                        xout.printdata("book","id",bookId.get(i),null);
                        xout.printdata("begin_date", getDate(randint(180)));
                        xout.printdata("pages_viewed", randint(numPages/2));
                        xout.printdata("chapters_viewed",randint(numChapters)/2);
                    }xout.closetag();
                }
            }xout.closetag();
            
		} xout.closetag();
		
		
	}
	
	String filename(String name, int max, int num){
		int    len = (max+"").length();
		String cnt = num+"";
		while(cnt.length()<len){ cnt = "0"+cnt; }
		return path+name+"_"+cnt+".xml";
	}
	
	void generate(String name, int numCases){
		try{
			ostream.println("Clearing away old tests");
            // Delete old output files
            File dir         = new File(path);
            if(!dir.exists()){
            	dir.mkdirs();
            }
            File oldOutput[] = dir.listFiles();
            for(File file : oldOutput){
               file.delete(); 
            }
        }catch(Exception e){
            throw new RuntimeException(e);
        }
        
		try {
			ostream.println("Generating "+numCases+" Tests");
			int inc = 100;
			if(inc < 100) inc = 1;
			if(inc < 1000) inc = 10;
			int lines = inc*10;
			
			for(int i=1;i<=numCases; i++){
				if(i>0 && i%inc   ==0 )ostream.print(i+" ");
				if(i>0 && i%lines ==0 )ostream.print("\n");
				xout = new XMLPrinter("book_preview",new FileOutputStream(filename(name,numCases,i)));
				generate();
				xout.close();
			}
		} catch (FileNotFoundException e) {
			ostream.println(e.toString());
			e.printStackTrace();
		}
	}
	
	public static void main(String args[]){
		TestCaseGen_BookPreview tcg = new TestCaseGen_BookPreview();
		tcg.generate("test",cnt);
	}
}
