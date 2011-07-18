package com.dtrules.samples.chipeligibility.app;

import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;

import com.dtrules.interpreter.RName;
import com.dtrules.samples.chipeligibility.app.dataobjects.Job;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

public class ChipApp {
	//
	// These settings are loaded in from testfiles/settings.xml
	//
	int 	threads 	= 0;		// number of threads to use
	int 	numCases 	= 0;		// number of cases to generate
	int 	save 		= 0;        // Save cases whose id is divisible by this number 
									//   1 would save every file, 3 would save every 3rd file.
	boolean trace 		= false;	// Create trace files (a coverage file will be generated if tracing)
									//   When used with save, only saved files are traced.
	boolean console     = false;    // Write results to console.
	
	int     update      = 60;       // Look for updates every 60 seconds
	
	// End of settings.
	
	String 		path = System.getProperty("user.dir") + "/";
	Date 		start = new Date();

	int 		processed[];
	int 		approved     = 0;
	int 		denied       = 0;
	int         total        = 0;
	int         empty        = 0;
	
	GenCase 	genCase 	 = new GenCase(this);
	String 		ruleset;
	RName 		rsName;
	RuleSet 	rs;
	List<Job> 	jobs        = Collections.synchronizedList(new ArrayList<Job>());
	

	int 		pulled 		= 0;
	int 		done 		= 0;
	
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
		return "CHIP";
	}

	public String getDecisionTableName() {
		return "Compute_Eligibility";
	}

	public String getRulesDirectoryFile() {
		return "DTRules.xml";
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
		
	synchronized Job next() {
		pulled++;
		if(pulled%(numCases/50) ==0){
			System.out.printf("%" + (numCases+"").length() +"d ",pulled);
			if(pulled%(numCases/5) ==0){
				Date now = new Date();
				long t   = now.getTime() - start.getTime();
				System.out.printf(" -- Run Time: %8d seconds\n", t/1000);
			}
		}
		if(pulled > numCases)return null;
		if(jobs.size()==0){
			genCase.fill(); 
		}
		return jobs.remove(0);
	}

	public static void main(String[] args) {
		ChipApp chipApp = new ChipApp();
		chipApp.run();
	}

	public void run() {		
		
		try {
			LoadSettings.loadSettings(this);
			
			System.out.println("Executing  "+threads+" threads");
			System.out.println("Processing "+numCases+" cases");
			System.out.println("Capturing jobs with numbers divisable by "+save);
			System.out.println("Tracing    "+(trace?"On":"Off"));
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
					if(time/1000 > update){
						System.out.println("Clear Cache");
						rs.clearCache();
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
				System.out.printf ("Number of Cases:          %8d\n",numCases);
                System.out.printf ("Cases without applicants: %8d\n",empty);
                System.out.printf ("Total clients in cases:   %8d\n",total);
				System.out.printf ("Total clients applying:   %8d\n",(approved+denied));
				System.out.printf ("Clients approved:         %8d\n",approved);
				System.out.printf ("Clients denied:           %8d\n",denied);
				System.out.println("\n");
				for(int i=1;i<processed.length;i++){
					System.out.printf("  Processed in thread %8d : %8d\n", i, processed[i]);
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
}
