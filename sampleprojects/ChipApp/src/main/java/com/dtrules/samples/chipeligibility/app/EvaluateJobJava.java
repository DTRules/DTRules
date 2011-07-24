package com.dtrules.samples.chipeligibility.app;

import java.io.FileOutputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.GregorianCalendar;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

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


public class EvaluateJobJava  {	
	
	ChipApp app;
	int     t;
	
	int FPL_Base                = 867;
	int FPL_PerAdditionalPerson	= 300;

	
	EvaluateJobJava(int threadnum, ChipApp app){
		this.app = app;
		this.t   = threadnum;
	}
	
	String [] excludedIncomes = {
			 "CS", "SS"
	};
	
	String [] coveredCounties = {
			"AA", "AK", "BC", "BK", "BT", "CR", "CO",  "TX",  "LO",  "AR"
	};
	
	Map<Integer,Boolean>  clientEligibility  = new HashMap<Integer,Boolean>();
	Map<Integer,String[]> clientNotes        = new HashMap<Integer,String[]>();
	Map<Integer,Integer>  clientIncome       = new HashMap<Integer,Integer>();
	Map<Integer,Integer>  clientGroupCnt 	 = new HashMap<Integer,Integer>();
	Map<Integer,Integer>  clientTotalIncome  = new HashMap<Integer,Integer>();
	
	public String evaluate(Job job) {
				
		clientEligibility.clear();
		clientNotes.clear();
		clientIncome.clear();
		clientGroupCnt.clear();
		clientTotalIncome.clear();
		
        if(job.getProgram()=="CHIP"){
        	Calculate_Individual_Income(job);
        	Calculate_Group_Size(job);
        	Evaluate_Chip_Eligibility(job);
        	Evaluate_Results(job);
        }
    	
        return null;
     }
    
	public void Calculate_Individual_Income(Job job){
		for(Client client : job.getCase().getClients()){
			int clientid = client.getId();
			Integer amount = 0;
			for(Income income : client.getIncomes()){
				if(income.getEarned()){
					amount += income.getAmount();
				}else{
					boolean include = true;
					for(String ex : excludedIncomes){
						if(ex.equals(income.getType())){
							include = false;
							break;
						}
					}
					if(include){
						amount += income.getAmount();
					}
				}
			}
			clientIncome.put(client.getId(), amount);
		}
	}
	
	
	
	public void Calculate_Group_Size(Job job){
		for(Client thisClient : job.getCase().getClients()) if(thisClient.getApplying()){
			int totalgroupincome = 0;
			int groupcnt         = 0;
			for(Client client : job.getCase().getClients()){

				if(thisClient == client){
					groupcnt++;
					if(client.getPregnant()){
						groupcnt += client.getExpectedChildren();
					}

					if(client.getAge()>18){
					   totalgroupincome += clientIncome.get(client.getId());
					}
				}
			
				if(is("parent", job.getCase(), client,thisClient)){
					groupcnt++;
					if(client.getPregnant()){
						groupcnt += client.getExpectedChildren();
					}
					
					if(client.getAge()>18){
						totalgroupincome += clientIncome.get(client.getId());
					}
				}
				
				if(is("sibling", job.getCase(), client,thisClient)){
					groupcnt++;
					if(client.getPregnant()){
						groupcnt += client.getExpectedChildren();
					}
					
					if(client.getAge()>18){
						totalgroupincome += clientIncome.get(client.getId());
					}
				}
				
				if(is("child", job.getCase(), client,thisClient)){
					groupcnt++;
					
					if(client.getAge()>18){
						totalgroupincome += clientIncome.get(client.getId());
					}
				}
			}
			clientGroupCnt.put(thisClient.getId(), groupcnt);
			clientTotalIncome   .put(thisClient.getId(), totalgroupincome);
		}
	}
	
	public void Evaluate_Chip_Eligibility(Job job){
		// For all the clients whose applying == true.
		for (Client client : job.getCase().getClients()) if(client.getApplying()){
			int incomeGroupCnt = clientGroupCnt.get(client.getId());
			int FPL = FPL_Base + FPL_PerAdditionalPerson * (incomeGroupCnt - 1 );
			int FPL200 = FPL*2;
			List <String> notes = new ArrayList<String>();
			boolean eligible = true;
			
			if(!client.getValidatedCitizenship()
					&& !client.getValidatedImmigrationStatus()){
				notes.add("The Client is not a validated Citizen, nor a validated immigrant");
				eligible = false;
			}
			
			if(clientTotalIncome.get(client.getId())>FPL200){
				notes.add("The Client's total group income, "+
			    clientTotalIncome.get(client.getId()) + 
			    "is greater than 200 percent of the FPL "+
			    FPL200);
				eligible = false;
			}
			if(!client.getUninsured()){
				//notes.add("The Applying Client must not have insurance");
				//eligible = false;
			}
			
			if(client.getUninsured() &&
				client.getLostInsuranceDate() != null){
				
				long delta = job.getCurrentdate().getTime() -
						client.getLostInsuranceDate().getTime();
				long days = delta/(1000*60*60*24);
				
				if(days < 90){
					notes.add("The Client must be uninsured for 90 days or more to qualify for CHIP");
					eligible = false;
				}
			}

			boolean goodCounty = false;
			for(String county : coveredCounties){
				if(job.getCase().getCounty_cd().equals(county)){
					goodCounty = true;
					break;
				}
			}
			
			if(!goodCounty){
				notes.add("The Client is not in a covered county");
				eligible = false;
			}
			
			if(client.getAge()>18){
				notes.add("The Client must be 18 or younger to qualify for CHIP");
				eligible = false;
			}
			
			if(client.getEligibleForMedicaid()){
				notes.add("The Client is eligible for Medicaid, so they are not eligible for CHIP");
				eligible = false;
			}
			
			clientEligibility.put(client.getId(), eligible);
			notes.add("Client Income = " + clientTotalIncome.get(client.getId()));
		}
	}
    
	
	public boolean is(String relationship,Case thecase, Client source,Client target){
		
		for(Relationship r : thecase.getRelationships()){
			if(   r.getSource()==source 
			   && r.getTarget()==target 
			   && r.getType().equals(relationship)){
					return true;
			}
		}

		return false;
		
	}
	
	public void Evaluate_Results(Job job){
		if(clientEligibility.size()==0 && app.console){
        	System.out.println("No results for job " + job.getId());
		}
		int approved = 0;
		int denied   = 0;
		for (Client client : job.getCase().getClients()) if(client.getApplying()){
			Boolean e = clientEligibility.get(client.getId());
			synchronized (app) {
				if(e==null){
					System.out.println("Client "+client.getId()+" is null");
				}else if(e){
					app.getApprovedClients().add(client.getId());
					approved++;
				} else{
					app.getDeniedClients().add(client.getId());
					denied++;
				}
			}
		}
		
		app.update(t, job.getCase().getClients().size(), approved, denied);
	}
	
}
