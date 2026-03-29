package com.dtrules.interpreter;

import java.util.ArrayList;
import java.util.HashMap;

public class RType {

	private static HashMap<String,RType> 	types    = new HashMap<String,RType>();
	private static ArrayList<RType>         typelist = new ArrayList<RType>();
	private static HashMap<String,RType> 	subtypes = new HashMap<String,RType>();
	
	private final String typename;
	
	private final int id;
	
	static {
		typelist.add(null);
	}
	
	/**
	 * Constructor building a RType name.
	 * @param name
	 */
	private RType(String name){
		typename = name;
		this.id  = typelist.size();
	}
	
	/**
	 * Returns true if this is a defined type
	 * @param name
	 * @return
	 */
	public static boolean isType(String name){
		name = name.toLowerCase();
		return types.containsKey(name);
	}
	
	/**
	 * Get the RType object for this type name.  Returns a null if the type isn't defined.
	 * @param name
	 * @return
	 */
	public static RType getType(String name){
		name = name.toLowerCase();
		return types.get(name);
	}

	/**
	 * Get the RType object for this type name.  Returns a null if the type isn't defined.
	 * @param name
	 * @return
	 */
	public static RType getType(int id){
		return typelist.get(id);
	}
	
	/**
	 * Define a new type with the given name.
	 * @param name
	 * @return
	 */
	public static RType newType(String name){
		name = name.toLowerCase();
		if(!isType(name)){
			RType newtype = new RType(name);
			types.put(name,newtype);
			typelist.add(newtype);
			return newtype;
		}else{
			return types.get(name);
		}
	}
		
	/**
	 * Add a subtype for this type.  Generally, subtypes are implementations
	 * of a given type.  
	 * @param type
	 */
	public void addSubType(RType type){
		if(!subtypes.containsKey(type.toString())){
			subtypes.put(type.toString(), type);
		}
	}
	
	/**
	 * get the Id for this type, which allows for quicker type checking.
	 * @return
	 */
	public int getId(){
		return id;
	}
	
	/**
	 * Get the RName for this type.
	 * @return
	 */
	public RName getName(){
		return RName.getRName(typename);
	}
	
	/**
	 * Get the string value for this type.
	 */
	public String toString(){
		return typename;
	}
}
