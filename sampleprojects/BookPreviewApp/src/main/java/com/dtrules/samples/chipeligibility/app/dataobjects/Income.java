package com.dtrules.samples.chipeligibility.app.dataobjects;

public class Income {
	static int 		ids=1;
	Integer			id;
	String			type;
	Integer			amount;
	Boolean			earned;
	
	public Income(){
		id=ids++;
	}
	
	public Integer getId() {
		return id;
	}

	public String getType() {
		return type;
	}
	public void setType(String type) {
		this.type = type;
	}
	public Integer getAmount() {
		return amount;
	}
	public void setAmount(Integer amount) {
		this.amount = amount;
	}
	public Boolean getEarned() {
		return earned;
	}
	public void setEarned(Boolean earned) {
		this.earned = earned;
	}
	
	
}
