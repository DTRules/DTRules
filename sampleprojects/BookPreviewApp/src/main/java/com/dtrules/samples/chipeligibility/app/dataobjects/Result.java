package com.dtrules.samples.chipeligibility.app.dataobjects;

import java.util.ArrayList;
import java.util.List;

public class Result {
	static int 		ids=1;
	
	Integer			id;
	String			client_id;
	String			program;
	String			programLevel;
	Boolean			eligible;
	List<String>	notes = new ArrayList<String>();;
	Client			client;
	Integer			client_fpl;
	Integer			totalGroupIncome;
		
	public Integer getId() {
		return id;
	}
	public String getClient_id() {
		return client_id;
	}
	public void setClient_id(String client_id) {
		this.client_id = client_id;
	}
	public String getProgram() {
		return program;
	}
	public void setProgram(String program) {
		this.program = program;
	}
	public String getProgramLevel() {
		return programLevel;
	}
	public void setProgramLevel(String programLevel) {
		this.programLevel = programLevel;
	}
	public Boolean getEligible() {
		return eligible;
	}
	public void setEligible(Boolean eligible) {
		this.eligible = eligible;
	}
	public List<String> getNotes() {
		return notes;
	}
	public void setNotes(List<String> notes) {
		this.notes = notes;
	}
	public Client getClient() {
		return client;
	}
	public void setClient(Client client) {
		this.client = client;
	}
	public Integer getClient_fpl() {
		return client_fpl;
	}
	public void setClient_fpl(Integer client_fpl) {
		this.client_fpl = client_fpl;
	}
	public Integer getTotalGroupIncome() {
		return totalGroupIncome;
	}
	public void setTotalGroupIncome(Integer totalGroupIncome) {
		this.totalGroupIncome = totalGroupIncome;
	}
	public static int getIds() {
		return ids;
	}

}
