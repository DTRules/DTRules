package com.dtrules.samples.bookpreview.app;

import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.dtrules.interpreter.RName;
import com.dtrules.samples.bookpreview.datamodel.DataObj;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

public class BookPreviewApp {
	//
	// These settings are loaded in from testfiles/settings.xml
	//
    int     db_delay     = 0;       // Simulated Database overhead per case.
	int 	threads 	 = 0;		// number of threads to use
	int 	numCases 	 = 0;		// number of cases to generate
	int 	save 		 = 0;       // Save cases whose id is divisible by this number 
					                //   1 would save every file, 3 would save every 3rd file.
	boolean console      = false;   // Write results to console.
	
	EvaluateJob ejob     = new EvaluateJobDTRules(); // Use DTRules to evaluate the rules.
	
	int     update       = 60;      // Look for updates every 60 seconds

	boolean printresults = false;	// Print ids of clients approved and denied if true.

	List<Integer> approvedClients = new ArrayList<Integer>() ;  // Approved clients
	List<Integer> deniedClients   = new ArrayList<Integer>() ;  // Denied clients
	
	Map<String,Integer> results = new HashMap<String,Integer>();
	
	// End of settings.
	
	String 		  path = System.getProperty("user.dir") + "/";
	Date 		  start = new Date();

	int 		  processed[];
	int 		  approved     = 0;
	int 		  denied       = 0;
	int           total        = 0;
	int           empty        = 0;
	
	GenCase 	  genCase 	   = new GenCase(this);
	String 		  ruleset;
	RName 		  rsName;
	RuleSet 	  rs;
	List<DataObj> jobs     = Collections.synchronizedList(new ArrayList<DataObj>());
	

	int 		  pulled       = 0;
	int 		  done         = 0;
	int           cacheloads   = 0;
	int           spaces       = 0;
	
	
	RulesDirectory rd;
	
	
	public synchronized void update(int thread, int total, int approved, int denied){
		processed[thread]     += approved+denied;
		this.denied   		  += denied;
		this.approved 		  += approved;
		this.total            += total;
		this.empty            += denied+approved == 0 ? 1 : 0;
	}
	
	public String getRulesDirectoryPath() {
		return path + "xml/";
	}

	public String getRuleSetName() {
		return "BookPreview";
	}

	public String getDecisionTableName() {
		return "Book_Access_Request";
	}

	public String getRulesDirectoryFile() {
		return "DTRules_BookPreview.xml";
	}

	public String getTestDirectory() {
		return path + "testfiles/";
	}

	public String getOutputDirectory() {
		return getTestDirectory() + "output/";
	}

	int jobsWaiting(){
		return jobs.size();
	}
		
	synchronized DataObj next() {
		pulled++;
		if(numCases>=50){
			if(pulled%(numCases/50) ==0){
				System.out.print(".");
				if(pulled%(numCases/5) ==0){
					int c = cacheloads;
					cacheloads = 0;
					if(c > 0 ){
						if(spaces == 0 ){
							spaces = c + 5;
						}
					}else if (spaces == 0){
						spaces = 5;
					}
					
					for(int i=0;i<spaces-c;i++) System.out.print(".");
					System.out.printf("%" + (numCases+"").length() +"d ",pulled);
					
					Date now = new Date();
					long t   = now.getTime() - start.getTime();
					System.out.printf(" -- Run Time: %8d seconds\n", t/1000);
				}
			}
		}
		if(pulled > numCases)return null;
		if(jobs.size()<50){
			genCase.fill(); 
		}
		return jobs.remove(0);
	}

	public static void main(String[] args) {
		BookPreviewApp bookPreviewApp = new BookPreviewApp();
		bookPreviewApp.run();
	}

	public void run() {		
		
		try {
			LoadSettings.loadSettings(this);
			System.out.println("Rules: "+ejob.getName());
			System.out.println("Executing  "+threads+" threads");
			System.out.println("Processing "+numCases+" cases");
			System.out.println("Capturing jobs with numbers divisable by "+save);
			System.out.println("Console Results "+(console?"On":"Off")+"\n");
			
			processed = new int [threads+1];
			
			// Delete old output files
			File dir = new File(getOutputDirectory());
			if (!dir.exists()) {
				dir.mkdirs();
			}
			File oldOutput[] = dir.listFiles();
			for (File file : oldOutput) {
				file.delete();
			}

			// Allocate a RulesDirectory. This object can contain many different
			// Rule Sets.
			// A Rule set is a set of decision tables defined in XML,
			// the Entity Description Dictionary (the EDD, or schema) assumed by
			// those tables, and
			// A Mapping file that maps data into this EDD.

			rd = new RulesDirectory(path, getRulesDirectoryFile());

			// Select a particular rule set and create a session to load the
			// data and evaluate
			// that data against the rules within this ruleset.
			ruleset  = getRuleSetName();
			rsName   = RName.getRName(ruleset);
			rs       = rd.getRuleSet(rsName);
			
			rs.newSession(); // Force the creation of the EntityFactory
			
			genCase.setLevel(10); 
			
			if (rs == null) {
				System.out.println("Could not find the Rule Set '" + ruleset
						+ "'");
				throw new RuntimeException("Undefined: '" + ruleset + "'");
			}

			{
				Thread ts[] = new Thread[threads];
				for (int i = 0; i < threads; i++) {
					ts[i] = new RunThread(i + 1, this);
				}
				
				System.out.print("\n\nNow Processing Jobs: \n\n");
				
				for (Thread t : ts)
					t.start();

				int sleep_ms = 1000;		
				int time     = 0;		
				while (threads > 0) {
					Thread.sleep(sleep_ms);
					time += sleep_ms;
					if(update > 0 && time/1000 > update){
						System.out.print("u");
						rs.clearCache();
						cacheloads++;
						time = 0;
					}
				}
			}
			Date now = new Date();
			int dfcnt = numCases;
			int jobcnt = dfcnt;
			if (jobcnt == 0)
				jobcnt = 1;
			{
				double dt = (now.getTime() - (double) start.getTime()) / (jobcnt);
				
				System.out.printf("\n\nA Job is processed every: %.8f seconds.\n\n",dt/1000);
				System.out.println("\n");
				
				String [] keys = results.keySet().toArray(new String [0]);
				sort(keys);
				for(String key : keys){
				    System.out.printf("%60s %8d\n", key, results.get(key));
				}
			}
			{
				long dt = (now.getTime() - start.getTime());
				long sec = dt / 1000;
				long lms = dt - (sec * 1000);
				String ms = lms < 100 ? lms < 10 ? "00" + lms : "0" + lms : ""
						+ lms;
				System.out.println("\nTotal time: " + sec + "." + ms);
			}
						 
		} catch (Exception e) {
			throw new RuntimeException(e);
		}

	}
	
	public void sort(String[] v){
	    boolean done = false;
	    for(int i=0; !done && i <v.length-1; i++){
	        done = true;
	        for(int j=0; j<v.length-i-1; j++){
	            if(v[j].compareTo(v[j+1])>0){
    	            String h = v[j];
    	            v[j]     = v[j+1];
    	            v[j+1]   = h;
    	            done = false;
	            }
	        }
	    }
	}
	
	public void sort(List<Integer> array){
		boolean done = false;
		for(int i=0; !done && i < array.size()-1; i++){
			done = true;
			for(int j=0; j < array.size()-i-1; j++){
				Integer a = array.get(j);
				Integer b = array.get(j+1);
				if((a!=null && b!= null && a>b)|| a==null){
					array.set(j,   b);
					array.set(j+1, a);
					done = false;
				}
			}
		}
	}

	public List<Integer> getApprovedClients() {
		return approvedClients;
	}

	public List<Integer> getDeniedClients() {
		return deniedClients;
	}

	
	
}
