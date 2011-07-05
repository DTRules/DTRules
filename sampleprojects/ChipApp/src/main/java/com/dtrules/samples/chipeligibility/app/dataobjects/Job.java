package com.dtrules.samples.chipeligibility.app.dataobjects;

import java.util.ArrayList;
import java.util.Date;
import java.util.List;

public class Job {
	static int		ids=1;
	Integer			id;
	Case			the_case;
	Date 			currentdate;
	Date 			effectivedate;
	List<Result> 	results = new ArrayList<Result>();;
	String			program;
	
	public Job(){
		id = ids++;
	}
	
	public Integer getId() {
		return id;
	}
	public Case getCase() {
		return the_case;
	}
	public void setCase(Case the_case) {
		this.the_case = the_case;
	}
	public Date getCurrentdate() {
		return currentdate;
	}
	public void setCurrentdate(Date currentdate) {
		this.currentdate = currentdate;
	}
	public Date getEffectivedate() {
		return effectivedate;
	}
	public void setEffectivedate(Date effectivedate) {
		this.effectivedate = effectivedate;
	}
	public List<Result> getResults() {
		return results;
	}
	public void setResults(List<Result> results) {
		this.results = results;
	}
	public String getProgram() {
		return program;
	}
	public void setProgram(String program) {
		this.program = program;
	}
	
}
