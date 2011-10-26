package com.dtrules.samples.bookpreview;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.Random;

import com.dtrules.interpreter.RName;
import com.dtrules.mapping.DataMap;
import com.dtrules.samples.bookpreview.datamodel.Book;
import com.dtrules.samples.bookpreview.datamodel.Chapter;
import com.dtrules.samples.bookpreview.datamodel.Customer;
import com.dtrules.samples.bookpreview.datamodel.Open_Book;
import com.dtrules.samples.bookpreview.datamodel.Page;
import com.dtrules.samples.bookpreview.datamodel.Publisher;
import com.dtrules.samples.bookpreview.datamodel.Request;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.XMLPrinter;

public class TestCaseGen_BookPreview {
	public static PrintStream ostream = System.out;
	public static PrintStream estream = System.err;
	
						      		// This is the default number of how many test cases to generate.
	static int cnt = 100;	    	// You can pass a different number on the commandline.
	
	static Random      rand 		 = new Random(1013);
	XMLPrinter 	       xout 		 = null;
	String 	 		   path 		 = System.getProperty("user.dir")+"/testfiles/";
	
	SimpleDateFormat   sdf           = new SimpleDateFormat("MM/dd/yyyy");
	Date        	   currentdate   = new Date();
	int                id            = 1;
	
	Request            request       = null;
	static Publisher   publishers[]  = new Publisher[8];       // We select from one of four publishers
	
    static {
	    for(int i=0;i<publishers.length;i++){
	        publishers[i]= new Publisher();
	        publishers[i].setChapter_limit(randint(3)+1);
	        publishers[i].setPage_limit(randint(100)+1); 
	    }
	}
	
	/**
	 * Return a date that is some number of days before the given 
	 * date
	 */
	Date getDate(long days){
	    Date d = new Date(currentdate.getTime()-(days*24l*60l*60l*1000l));
	    return d; 
	}
	
    int getId() { return id++; }
	
	static int randint(int limit){
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
	
	private Book newBook(){
	    Book book = new Book();
	    book.setPublisher(publishers[randint(publishers.length)]);
	    int pages = randint(200)+200;
	    book.setPages(pages);
	    int start = 1;
	    boolean excluded = false;
	    while(start < pages){
	        int end = randint(20)+20+start;            // Chapters are 20 to 40 pages
	        if(pages-end < 10 || end >= pages) end = pages; // Okay, we fit in the last chapter
	        Chapter c = new Chapter();                 // Set up the chapter
	        c.setBegin_page(start);
	        c.setEnd_page(end);
	        
	        book.getChapters().add(c);                 // Add the chapter, and 
	        start = end+1;                             //  set the start of the next chapter
	  
	        if(excluded){                              // We exclude the ending chapters randomly.
	            book.getExcluded_chapters().add(c);
	        }else{
	           excluded = randint(100)>70;             // Once we start excluding, we exclude the rest.
	        }
	    }
	    
	    if(flip()||flip()){                            // Half the books get a day limit.
	        book.setDay_limit((randint(5)+1)*30);      // Limit to 30, 60, 90, 120, or 180 days.
	    }else{
	        book.setDay_limit(0);
	    }
	    return book;
	}
	
	public Request generate() throws Exception {
	    request = new Request();
	    request.setCustomer(new Customer());
	    request.setBook(newBook());
	    request.setAccess("turn_page");
	    request.setCurrent_date(sdf.parse("01/02/2011"));
	    request.setPage_number(randint(request.getBook().getPages())+1);
	    
	    Customer customer = request.getCustomer();
	    int num_openbooks = randint(5)+1;
	    for(int i=0; i<num_openbooks; i++){
	        Open_Book ob = new Open_Book();
	        ob.setBook(newBook());
	        customer.getOpen_books().add(ob);
	        ob.setBegin_date(getDate(randint(400)));
	    }
	    int index = randint(customer.getOpen_books().size());
	    customer.getOpen_books().get(index).setBook(request.getBook()); // Make one of the open books match 
	                                                                    //   the request.
	    
	    for(int i=0; i<num_openbooks; i++){
	        Open_Book  ob          = customer.getOpen_books().get(i);
	        Book       b           = ob.getBook();
	        int        pages_read  = randint(60);
	        int        this_page   = randint(5)+1;
	        Chapter    c           = b.getChapters().get(0);
	        
	        while(this_page < pages_read){
	            if(b.getExcluded_chapters().contains(c)) break;
	            if(this_page > pages_read)break;
	            if(this_page <= c.getEnd_page()){
	                if(!ob.getChapters_viewed().contains(c)){
	                    ob.getChapters_viewed().add(c);
	                }
	                Page page = new Page();
	                page.setNumber(this_page);
	                this_page += 1;
	                ob.getPages().add(page);
	            }
	            if(this_page > c.getEnd_page()){
	                for(Chapter n : b.getChapters()){
	                    if(n.getBegin_page()<this_page && n.getEnd_page()>= this_page){
	                        c = n;
	                        break;
	                    }
	                }
	            }
	            this_page += (randint(20)/7);
	        }
	    }
	
	    return request;
	}  
	
	
	String filename(String name, int max, int num){
		int    len = (max+"").length();
		String cnt = num+"";
		while(cnt.length()<len){ cnt = "0"+cnt; }
		return path+name+"_"+cnt+".xml";
	}
	
	void generate(String name, int numCases) throws Exception {
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

		String          path    = System.getProperty("user.dir")+"/";
		String          config  = "DTRules_BookPreview.xml";
        RulesDirectory  rd      = new RulesDirectory(path, config);
        RuleSet         rs      = rd.getRuleSet(RName.getRName("BookPreview"));
        IRSession       session = rs.newSession();

		try {
			ostream.println("Generating "+numCases+" Tests");
			int inc = 100;
			if(inc < 100) inc = 1;
			if(inc < 1000) inc = 10;
			int lines = inc*10;
			
			for(int i=1;i<=numCases; i++){
				if(i>0 && i%inc   ==0 )ostream.print(i+" ");
				if(i>0 && i%lines ==0 )ostream.print("\n");
				OutputStream out = new FileOutputStream(filename(name,numCases,i));
                DataMap datamap = session.getDataMap(session.getMapping(),"BookPreview");
				
                Request request = generate();
				request.print(datamap);
                datamap.print(out);
			}
		} catch (FileNotFoundException e) {
			ostream.println(e.toString());
			e.printStackTrace();
		}
	}
	
	public static void main(String args[]) throws Exception {
		TestCaseGen_BookPreview tcg = new TestCaseGen_BookPreview();
		tcg.generate("test",cnt);
	}
}
