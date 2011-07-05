package com.dtrules.samples.chipeligibility.app.dataobjects;

import java.util.ArrayList;
import java.util.Date;
import java.util.List;

public class Client {
	static int		ids=1;
	Integer			id;
	
	Integer			age;
	String			gender;
	Boolean			validatedCitizenship;
	Boolean			livesAtResidence;
	List<Income>	incomes= new ArrayList<Income>();
	Integer			expectedChildren;
	Boolean			disabled;
	Boolean			applying;
	List<String>	notes= new ArrayList<String>();
	Boolean			eligible;
	String			programLevel;
	Integer			totalIncome;
	Integer			incomeGroupCount;
	Integer			totalGroupIncome;
	Integer			client_fpl;
	Boolean			uninsured;
	Boolean			validatedImmigrationStatus;
	Date			lostInsuranceDate;
	Boolean 		eligibleForMedicaid;
	Boolean			pregnant;
	
	public Client(){
		id = ids++;
	}
	
	public Integer getId(){
		return id;
	}
	public Integer getAge() {
		return age;
	}
	public void setAge(Integer age) {
		this.age = age;
	}
	public String getGender() {
		return gender;
	}
	public void setGender(String gender) {
		this.gender = gender;
	}
	public Boolean getValidatedCitizenship() {
		return validatedCitizenship;
	}
	public void setValidatedCitizenship(Boolean validatedCitizenship) {
		this.validatedCitizenship = validatedCitizenship;
	}
	public Boolean getLivesAtResidence() {
		return livesAtResidence;
	}
	public void setLivesAtResidence(Boolean livesAtResidence) {
		this.livesAtResidence = livesAtResidence;
	}
	public List<Income> getIncomes() {
		return incomes;
	}
	public void setIncomes(List<Income> incomes) {
		this.incomes = incomes;
	}
	public Boolean getPregnant() {
		return pregnant;
	}
	public void setPregnant(Boolean pregnant) {
		this.pregnant = pregnant;
	}
	public Integer getExpectedChildren() {
		return expectedChildren;
	}
	public void setExpectedChildren(Integer expectedChildren) {
		this.expectedChildren = expectedChildren;
	}
	public Boolean getDisabled() {
		return disabled;
	}
	public void setDisabled(Boolean disabled) {
		this.disabled = disabled;
	}
	public Boolean getApplying() {
		return applying;
	}
	public void setApplying(Boolean applying) {
		this.applying = applying;
	}
	public List<String> getNotes() {
		return notes;
	}
	public void setNotes(List<String> notes) {
		this.notes = notes;
	}
	public Boolean getEligible() {
		return eligible;
	}
	public void setEligible(Boolean eligible) {
		this.eligible = eligible;
	}
	public String getProgramLevel() {
		return programLevel;
	}
	public void setProgramLevel(String programLevel) {
		this.programLevel = programLevel;
	}
	public Integer getTotalIncome() {
		return totalIncome;
	}
	public void setTotalIncome(Integer totalIncome) {
		this.totalIncome = totalIncome;
	}
	public Integer getIncomeGroupCount() {
		return incomeGroupCount;
	}
	public void setIncomeGroupCount(Integer incomeGroupCount) {
		this.incomeGroupCount = incomeGroupCount;
	}
	public Integer getTotalGroupIncome() {
		return totalGroupIncome;
	}
	public void setTotalGroupIncome(Integer totalGroupIncome) {
		this.totalGroupIncome = totalGroupIncome;
	}
	public Integer getClient_fpl() {
		return client_fpl;
	}
	public void setClient_fpl(Integer client_fpl) {
		this.client_fpl = client_fpl;
	}
	public Boolean getUninsured() {
		return uninsured;
	}
	public void setUninsured(Boolean uninsured) {
		this.uninsured = uninsured;
	}
	public Boolean getValidatedImmigrationStatus() {
		return validatedImmigrationStatus;
	}
	public void setValidatedImmigrationStatus(Boolean validatedImmigrationStatus) {
		this.validatedImmigrationStatus = validatedImmigrationStatus;
	}
	public Date getLostInsuranceDate() {
		return lostInsuranceDate;
	}
	public void setLostInsuranceDate(Date lostInsuranceDate) {
		this.lostInsuranceDate = lostInsuranceDate;
	}
	public Boolean getEligibleForMedicaid() {
		return eligibleForMedicaid;
	}
	public void setEligibleForMedicaid(Boolean eligibleForMedicaid) {
		this.eligibleForMedicaid = eligibleForMedicaid;
	}
	
	
}
