package com.dtrules.samples.chipeligibility.app.dataobjects;

public class Relationship {
	static int  ids=1;
	Integer		id;
	Client		source;
	Client		target;
	String		type;
	
	public Relationship(){
		id = ids++;
	}

	public Integer getId() {
		return id;
	}
	public Client getSource() {
		return source;
	}
	public void setSource(Client source) {
		this.source = source;
	}
	public Client getTarget() {
		return target;
	}
	public void setTarget(Client target) {
		this.target = target;
	}
	public String getType() {
		return type;
	}
	public void setType(String type) {
		this.type = type;
	}
	
	
}
