package com.dtrules.samples.chipeligibility.app.dataobjects;

import java.util.ArrayList;
import java.util.List;

public class Case {
	static int			ids=1;
	Integer				id;
	
	List<Client>		clients = new ArrayList<Client>();
	String				county_cd;
	List<Relationship>	relationships = new ArrayList<Relationship>();
	
	public Case(){
		id = ids++;
	}
	
	public List<Client> getClients() {
		return clients;
	}
	public Integer getId() {
		return id;
	}

	public void setClients(List<Client> clients) {
		this.clients = clients;
	}
	public String getCounty_cd() {
		return county_cd;
	}
	public void setCounty_cd(String county_cd) {
		this.county_cd = county_cd;
	}
	public List<Relationship> getRelationships() {
		return relationships;
	}
	public void setRelationships(List<Relationship> relationships) {
		this.relationships = relationships;
	}
	
}
