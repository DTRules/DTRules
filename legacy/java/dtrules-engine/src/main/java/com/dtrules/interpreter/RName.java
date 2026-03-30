/** 
 * Copyright 2004-2011 DTRules.com, Inc.
 * 
 * See http://DTRules.com for updates and documentation for the DTRules Rules Engine  
 *   
 * Licensed under the Apache License, Version 2.0 (the "License");  
 * you may not use this file except in compliance with the License.  
 * You may obtain a copy of the License at  
 *   
 *      http://www.apache.org/licenses/LICENSE-2.0  
 *   
 * Unless required by applicable law or agreed to in writing, software  
 * distributed under the License is distributed on an "AS IS" BASIS,  
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
 * See the License for the specific language governing permissions and  
 * limitations under the License.  
 **/
  
package com.dtrules.interpreter;

import java.util.Hashtable;
import java.util.regex.Pattern;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.DTState;

public class RName extends ARObject implements Comparable<RName>{
	
	static RType type = RType.newType("name");

	final RName   entity;
	final String  name;
	final boolean executable;
    final int     hashcode;
    final RName   partner;		// RNames are created in pairs, executable and non-executable.
    final String  postfix;
    
    /** 
     * I started with a HashMap here.  However, we can deadlock with multiple threads.  So
     * I tried two possible fixes, ConcurrentHashMap and Hashtable.  The latter gave me the
     * better performance with a Rule Set eating a large set of data.  We may look into this
     * further later... <br><br>
     * 
     * Results from my testing:
     * 
     * 		 Hashtable without caching:
     * 
     * 		 47.8  Multi_NY_Enrollment_Test
     * 
     *       HashMap
     *      
     *		 44.3 Multi_NY_Enrollment_Test
     *       
     *       ConcurrentHashMap 
     *       
     *		 44.6 Multi_NY_Enrollment_Test       
     *       
     *       Hashtable
     *       
     *       44.5 Multi_NY_Enrollment_Test
     */
    
//    static HashMap<String, RName> names = new HashMap<String,RName>();
//    static ConcurrentHashMap<String, RName> names = new ConcurrentHashMap<String,RName>();
  static Hashtable<String, RName> names = new Hashtable<String,RName>();
    
	/**
	 * This constructor should only be called by the other RName constructor.
	 * It is used to build its partner.  RNames are always created in pairs,
	 * one executable and one non-executable.
	 * 
	 * @param _name
	 * @param _executable
	 * @param _hashcode
	 * @param _partner
	 */
	private RName(RName _entity, String _name, boolean _executable, int _hashcode, RName _partner){
		entity         = _entity;
		name           = _name;
		executable     = _executable;
		hashcode       = _hashcode;
		partner        = _partner;
		postfix        = _executable ? _name : "/"+_name;
	}
	/**
	 * This constructor should only be called by the getRName method.  This
	 * constructor always creates two RNames, the one requested and its partner. 
	 * 
	 * @param _name
	 * @param _executable
	 * @param _hashcode
	 */
	private RName(RName _entity, String _name, boolean _executable, int _hashcode ){
		entity         = _entity;
		name           = _name;
		executable     = _executable;
		hashcode       = _hashcode;
        postfix        = _executable ? _name : "/"+_name;
		partner        = new RName(_entity, _name,!_executable,_hashcode,this);
	}
	
	static Pattern spaces = Pattern.compile(" ");
	/**
	 * When you don't really care about the executable nature of the 
	 * name, then use this accessor. We parse looking for the slash to
     * determine a literal, so by default (i.e. no slash given), we 
     * return executable names.  This constructor also pareses to handle 
     * the "dot" syntax.
	 * @exception RuntimeException is thrown if the Syntax is incorrect.
	 * @param _name String from which to create a name
	 * @return The Name object
	 */
	static public RName getRName(String _name){
        String cache = _name;
	    RName rname = names.get(_name);
        if(rname != null)return rname;
	    
        // Fix the name; trim and then replace internal spaces with '_'.
	    _name = spaces.matcher(_name.trim()).replaceAll("_");
        boolean executable = !(_name.indexOf('/')==0);
        if(!executable){
            _name = _name.substring(1); // Remove the slash.
        }
        int dot = _name.indexOf('.');
        
        if(dot>=0){
            String entity = _name.substring(0,dot);
            if(dot == 0 || dot+1 == _name.length() || _name.indexOf(dot+1,'.')>=0){
                throw new RuntimeException("Invalid Name Syntax: ("+_name+")");
            }
            String name   = _name.substring(dot+1);
            rname = getRName(RName.getRName(entity),name,executable);
            names.put(cache, rname);
            return rname;
        }
		rname = getRName(null, _name, executable);
		names.put(cache,rname);
		return rname;
	}
	
	/* (non-Javadoc)
     * @see java.lang.Comparable#compareTo(java.lang.Object)
     */
    @Override
    public int compareTo(RName o) {
        try{
            return this.compare(o);
        }catch(RulesException e){
            return 0;
        }
    }
    /**
	 * We cache the creation of RNames so as to not create new copies
	 * of RNames that we don't have to create. (RNames are reusable)
	 * <br><br>
	 * Thus one calls one of the getRName functions, and one cannot call our
	 * constructors directly.
	 * <br><br>
	 * Returns a null if a RName cannot be found/created.
	 * 
	 * @param _name
	 * @return
	 */
	static public RName getRName(String _name, boolean _executable) {
        // Fix the name; trim and then replace internal spaces with '_'.
        _name = _name.trim().replaceAll(" ","_");
		return getRName(null,_name,_executable);
	}

	static Pattern space = Pattern.compile(" ");
	/**
	 * We cache the creation of RNames so as to not create new copies
	 * of RNames that we don't have to create. (RNames are reusable)
	 * <br><br>
	 * Thus one calls one of the getRName functions, and one cannot call our
	 * constructors directly.
	 * 
	 * @param _name
	 * @return
	 */
	static public RName getRName(RName _entity, String _name, boolean _executable) {
	    RName rn = (RName) names.get(_name);
	    if(rn != null && _entity == null){
	        return _executable ? (RName) rn.getExecutable() : (RName) rn.getNonExecutable();
	    }
	    if(rn!=null)
	    // Fix the name; trim and then replace internal spaces with '_'.
		_name = space.matcher(_name).replaceAll("_");
		// If we already have the RName, we are done.
		String lname = _name.toLowerCase();
        String cname = _name;
        if(_entity!=null){
		   cname = _entity.stringValue().toLowerCase()+"."+lname;
        }
		rn = (RName) names.get(cname);
		if(rn == null ) {
			rn = new RName(_entity ,_name,_executable, lname.hashCode());
			names.put(cname,(RName)rn.getExecutable());
		}
		if(_executable) return (RName) rn.getExecutable();
		return (RName) rn.getNonExecutable();
	}
	/**
	 * Returns the entity component of the name, which is null if none
	 * was specfied when the name was created.
	 * 
	 * @return
	 */
	public RName getEntityName(){
		return entity;
	}
	
	/**
	 * Returns just the attribute component of the name
	 */
	public String getName(){
	    return name;
	}
	
	
	public boolean equals(Object arg0) {
        if(arg0.getClass()!=RName.class)return false; 
		boolean f = name.equalsIgnoreCase(((RName)arg0).name);
        return f;
	}

	public int hashCode() {
		return hashcode;
	}
	
	/**
	 * Compare this RName with another Rules Engine Object.  RNames
	 * are only equal to other RNames. 
	 */
	public boolean equals(IRObject o) {
		if(o.type()!= type)return false;
		return equals((Object)o);
	}
    static int cnt= 0;
	/**
	 * The execution of an RName looks up that RName in the Entity
	 * Stack.  If the object found there is an Array, it is pushed 
	 * to the Entity Stack.  If the object found there is not an
	 * array, and it is not executable, then it is pushed.  Otherwise
	 * (not an array, and executable) the object is executed.
	 */
	public void arrayExecute(DTState state) throws RulesException {	
		if(executable){
			cnt++;
	        IRObject o = state.find(this);		      // Does a lookup of the name on the Entity Stack
			if(o==null){
	            throw new RulesException("Undefined","RName","The Name '"+name+"' was not defined by any Entity on the Entity Stack");
			}
			if(o.isExecutable()){
				o.execute(state);
			}else{
				state.datapush(o);
			}
		}else{
			state.datapush(this);
		}
	}
	
	public void execute(DTState state) throws RulesException {
		getExecutable().arrayExecute(state);
	}
	
	public IRObject getExecutable() {
		return executable ? this : partner ;
	}
	
	public IRObject getNonExecutable() {
		return executable ? partner : this ;
	}
	
	public boolean isExecutable() {
		return executable;
	}
	
	
	/** 
	 * Returns the postfix version of the name.
	 */
	public String postFix() {
		return postfix;
	}
	
	/**
	 * Returns the value of the name unadorned by the leading slash, even
	 * if the name is a literal.
	 */
	public String stringValue() {
		if(entity!=null){
			return entity.stringValue()+"."+name;
		}
		return name;
	}
	
	/**
	 * Returns the nicest format for debugging, i.e. the Postfix version which
	 * has the slash if the name is a literal name.
	 */
	public String toString(){
		return postFix();
	}
	
	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}

    
    @Override
    public int compare(IRObject obj) throws RulesException {
        String v = obj.stringValue();
        return name.compareToIgnoreCase(v);    
    }
	/**
	 * Returns myself
	 */
	public RName rNameValue() throws RulesException {
		return this;
	}
	/* (non-Javadoc)
	 * @see com.dtrules.interpreter.ARObject#rStringValue()
	 */
	@Override
	public RString rStringValue() {
		return RString.newRString(name);
	}

}
