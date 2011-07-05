package com.dtrules.samples.chipeligibility;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.GregorianCalendar;
import java.util.Random;

import com.dtrules.xmlparser.XMLPrinter;

public class TestCaseGen {
								// This is the default number of how many test cases to generate.
	static int cnt = 1000;		// You can pass a different number on the commandline.
	
	Random 		       rand 		 = new Random(1013);
	XMLPrinter 	       xout 		 = null;
	String 	 		   path 		 = System.getProperty("user.dir")+"/testfiles/";
	ArrayList<Integer> clients 		 = new ArrayList<Integer>();
	ArrayList<Integer> relationships = new ArrayList<Integer>();   
	
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
	
	
	void addchild(int clientId, int ageParent){
		ArrayList<Integer> sibs = new ArrayList<Integer>();
		
		while(sibs.size()==0 || chance(40)){
			int childId = genClient(ageParent-15);
			sibs.add(childId);
			relationships.add(rId);
			xout.opentag("relationship","id",rId);
			xout.printdata("source", "id", (Integer) clientId, null);
			xout.printdata("target", "id", (Integer) childId,  null);
			xout.printdata("type","parent");
			xout.closetag();
			rId++;
			relationships.add(rId);
			xout.opentag("relationship","id",rId);
			xout.printdata("source", "id", (Integer) childId,  null);
			xout.printdata("target", "id", (Integer) clientId, null);
			xout.printdata("type","child");
			xout.closetag();
			rId++;
		}
		for(int sib1 : sibs){
			for(int sib2 : sibs ){
				if(sib1 != sib2){
					relationships.add(rId);
					xout.opentag("relationship","id",rId);
					xout.printdata("source", "id", (Integer) sib1,  null);
					xout.printdata("target", "id", (Integer) sib2, null);
					xout.printdata("type","sibling");
					xout.closetag();
					rId++;
				}
			}
		}
	}
	
	void addIncome(){
		int p = randint(100);
		int i;
		for(i=0; p>0 && i < odds.length; i++) p -= odds[i];
		if(i==odds.length)i--;
		xout.opentag("income");
		xout.printdata("type",incomes[i]);
		xout.printdata("amount",randint(1000));
		xout.printdata("earned",earned[i]);
		xout.closetag();
	}
	
	int genClient(int ageLimit){
		int clientId = cId++;
		int age = randint(ageLimit);
		String gender = flip()?"male":"female"; 
		clients.add(clientId);
		xout.opentag("client","id",clientId);
		xout.printdata("age", age);
		xout.printdata("gender",gender);
		xout.printdata("livesAtResidence",chance(90)?"true":"false");
		boolean citizen = chance(90);
		xout.printdata("validatedCitizenship",citizen);
		boolean pregnant = gender.equals("female")&&age>15&&chance(30);
		xout.printdata("pregnant",pregnant);
		xout.printdata("expectedChildren",pregnant&&chance(80)?1: pregnant&&chance(80)?2: pregnant?3:0);
		xout.printdata("validatedImmigrationStatus",((!citizen)&&chance(50))?"true":"false");
		xout.printdata("disabled",chance(10)?"true":"false");
		xout.printdata("applying",age<22? (chance(90)?"true":"false"):"false");
		boolean uninsured = chance(50);
		xout.printdata("uninsured",uninsured?"true":"false");
		if(uninsured){
			cal.setTime(currentdate);
			int months = randint(4);
			cal.add(Calendar.MONTH, -months);
			xout.printdata("lostInsuranceDate",sdf.format(cal.getTime()));
		}
		xout.printdata("eligibleForMedicaid",chance(50)?"true":"false");

		int icnt = randint(4);
		for(int i = 0; age > 18 && i < icnt; i++){
			addIncome();
		}
		xout.closetag();
		
		if(age>15 && chance(20)){
			addchild(clientId, age);
		}else if(age>30 && chance(50)){
			addchild(clientId, age);
		}
		return clientId;
	}
	
	void generate(){
		
		clients 		 = new ArrayList<Integer>();
		relationships    = new ArrayList<Integer>();   
		
		xout.opentag("job","id",jobId);
		jobId++;
		
		xout.printdata("currentdate",sdf.format(currentdate));
		xout.printdata("effectivedate",sdf.format(nextMonth));
		xout.printdata("program", "CHIP");
		
		xout.closetag();
		
		int nc = randint(4)+3;
		for(int i=1; i <= nc; i++){
			genClient(75);
		}
		xout.opentag("case","id",caseId);
		caseId++;
		xout.printdata("county_cd",counties[randint(counties.length)]);
		xout.opentag("clients","number",clients.size());
		for(int c :clients){
			xout.printdata("clients","id",c,null);
		}
		xout.closetag();
		xout.opentag("relationships");
		for(int r : relationships){
			xout.printdata("relationship","id",r,null);
		}
		xout.closetag();
		
		xout.closetag();
	}
	
	String filename(String name, int max, int num){
		int    len = (max+"").length();
		String cnt = num+"";
		while(cnt.length()<len){ cnt = "0"+cnt; }
		return path+name+"_"+cnt+".xml";
	}
	
	void generate(String name, int numCases){
		try{
			System.out.println("Clearing away old tests");
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
			System.out.println("Generating "+numCases+" Tests");
			int inc = 100;
			if(inc < 100) inc = 1;
			if(inc < 1000) inc = 10;
			int lines = inc*10;
			
			for(int i=1;i<=numCases; i++){
				if(i>0 && i%inc   ==0 )System.out.print(i+" ");
				if(i>0 && i%lines ==0 )System.out.print("\n");
				xout = new XMLPrinter("chip_case",new FileOutputStream(filename(name,numCases,i)));
				generate();
				xout.close();
			}
		} catch (FileNotFoundException e) {
			System.out.println(e.toString());
			e.printStackTrace();
		}
	}
	
	public static void main(String args[]){
		  
		if(args.length>0){
			try {
				cnt = Integer.parseInt(args[0]);
			} catch (NumberFormatException e) {
				System.err.println("Could not parse '"+args[0]+"' as a number.\n" +
						"Usage:  TestCaseGen <number>");
			}
		}
		TestCaseGen tcg = new TestCaseGen();
		tcg.generate("test",cnt);
	}
}
