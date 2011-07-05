package com.dtrules.samples.chipeligibility.app;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.List;
import java.util.Random;

import com.dtrules.samples.chipeligibility.app.dataobjects.Case;
import com.dtrules.samples.chipeligibility.app.dataobjects.Client;
import com.dtrules.samples.chipeligibility.app.dataobjects.Income;
import com.dtrules.samples.chipeligibility.app.dataobjects.Job;
import com.dtrules.samples.chipeligibility.app.dataobjects.Relationship;

public class GenCase  {
								// This is the default number of how many test cases to generate.
	static int cnt = 1000;		// You can pass a different number on the commandline.
	
	Random 		       rand 		 = new Random(1013);
	
	ChipApp            app;
	int				   level;        // Number of jobs we keep in the queue.
	
	String             incomes[]     = {"WA","SSI","SS","CS","TIP","DIV","SET"};
	int                odds[]        = {45  , 10  ,10  , 15 ,10   , 5   , 5};
	boolean            earned[]      = {true,true,true,false,true,false,false};
	
	String             counties[]    = { "UK", "OT", "AA", "AK", "BC", "BK", "BT", 
			                             "CR", "CO",  "TX",  "LO",  "AR" };
	
	SimpleDateFormat   sdf           = new SimpleDateFormat("MM/dd/yyyy");
	Date        	   currentdate   = new Date();
	Date               nextMonth;
	int                jobId         = 1;   // job ID
	int                rId           = 1;	// relationship ID
	int                cId           = 1;	// client ID
	int                caseId        = 1;   // case ID
	GregorianCalendar  cal 			 = (GregorianCalendar) GregorianCalendar.getInstance();
	{
		cal.setTime(currentdate);
		cal.add(Calendar.MONTH, 1);
		cal.set(Calendar.DAY_OF_MONTH, 1);
		nextMonth = cal.getTime();
	}
	
	GenCase(ChipApp app){
		this.app = app;
	}
	
	
	int randint(int limit){ 
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

	/**
	 * Expects an integer array of the form : [20,30,15,5,25,5]
	 * Were all the numbers add up to 100.   
	 * 
	 * A random number is generated where r >= 0 && r < 100.  
	 * Then the array is examined, and the index into the array is returned. 
	 * (in our example, 
	 * 0 is returned if r < 20, 
	 * 1 is returned if r < 20+30,  
	 * 2 if r < 20+30+15, 
	 * etc.)
	 *    
	 * @param odds
	 * @return
	 */
	int indexedOdds(int odds[]){
		int sum = 0;
		int r = randint(100);
		for(int i = 0; i < odds.length; i++){
			sum += odds[i];
			if(r < sum)return i;
		}
		return odds.length-1;
	}
	
	void addchild(Job job, Client parent, int ageParent){
		ArrayList<Client> sibs = new ArrayList<Client>();
		
		while(job.getCase().getClients().size()<10 && sibs.size()==0 || chance(40)){
			Client child = genClient(job, ageParent-15);
			sibs.add(child);
			Relationship r = new Relationship();
			job.getCase().getRelationships().add(r);
			r.setSource(parent);
			r.setTarget(child);
			r.setType("parent");
			
			r = new Relationship();
			job.getCase().getRelationships().add(r);
			r.setSource(child);
			r.setTarget(parent);
			r.setType("child");
		}
		for(Client sib1 : sibs){
			for(Client sib2 : sibs ){
				if(sib1 != sib2){
					Relationship r = new Relationship();
					job.getCase().getRelationships().add(r);
					r.setSource(sib1);
					r.setTarget(sib2);
					r.setType("sibling");
				}
			}
		}
	}
	
	void addIncome(Job job,Client client){
		int i = indexedOdds(odds);
		Income income = new Income();
		client.getIncomes().add(income);
		income.setType(incomes[i]);
		income.setAmount(randint(800)+200);
		income.setEarned(earned[i]);
	}
	
	Client genClient(Job job, int ageLimit){
		Client client = new Client();
		job.getCase().getClients().add(client);
		int age = randint(ageLimit);
		String gender = flip()?"male":"female"; 
		client.setAge(age);
		client.setGender(gender);
		client.setLivesAtResidence(chance(90)?true:false);
		boolean citizen = chance(90);
		client.setValidatedCitizenship(citizen);
		boolean pregnant = gender.equals("female")&&age>15&&chance(30);
		client.setPregnant(pregnant);
		if(pregnant){
			client.setExpectedChildren(chance(80)?1: chance(80)?2:3);
		}else{
			client.setExpectedChildren(0);
		}
		if(!citizen){
			client.setValidatedImmigrationStatus(chance(50)?true:false);
		}
		client.setDisabled(chance(10)?true:false);
		client.setApplying(age<22? (chance(90)?true:false):false);
		boolean uninsured = chance(50);
		client.setUninsured(uninsured);
		if(uninsured){
			cal.setTime(currentdate);
			int months = randint(4);
			cal.add(Calendar.MONTH, -months);
			client.setLostInsuranceDate(cal.getTime());
		}
		client.setEligibleForMedicaid(chance(50)?true:false);

		int icnt = randint(4);
		for(int i = 0; age > 18 && i < icnt; i++){
			addIncome(job, client);
		}
		
		if(job.getCase().getClients().size()<10 && age>15 && chance(20)){
			addchild(job, client, age);
		}else if(job.getCase().getClients().size()<10 && age>30 && chance(50)){
			addchild(job, client, age);
		}
		return client;
	}
	
	Job generate(){
		Job job 	     = new Job();
		Case c 		     = new Case();
		job.setCase(c);
		
		job.setCurrentdate(currentdate);
		job.setEffectivedate(nextMonth);
		job.setProgram("CHIP");
		
		
		int nc = randint(4)+3;
		for(int i=1; i <= nc; i++){
			if(job.getCase().getClients().size()<10){
			   genClient(job,75);
			}
		}
		
		c.setCounty_cd(counties[randint(counties.length)]);
		
		return job;
	}
	
	public void setLevel(int i){
		level = i;
		if (i<100){ level = 100; }
	}
	
	/**
	 * This method is going to watch the queue in the ChipApp, and fill
	 * it with test cases until until full (i.e. has level many jobs in it).
	 */
	public void fill() {
		while(app.jobsWaiting()<level){
			Job job = generate();
			app.jobs.add(job);
		}
	}
	
}
