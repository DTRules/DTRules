/** 
 * Copyright 2004-2009 DTRules.com, Inc.
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
  
package com.dtrules.entity;

import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.Set;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.ARObject;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RXmlValue;
import com.dtrules.mapping.XMLNode;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;

/**
 * Entities serve as the dictionaries of the Rules Engine.  These
 * Dictionaries are typed Hashtables.  In other words, you can't 
 * just put any old object into an Entity.  There has to be an
 * attribute with the appropriate name and type in order to put
 * a particular object with that name into an Entity.
 * <br><br>
 * This structure catches many data entry and program structure 
 * errors, as well as provides for a number of convienent automatic 
 * data conversions (if the compiler writer cares to provide such
 * facilities).
 * 
 * @author Paul Snow
 *
 */
public class REntity extends ARObject implements IREntity {

    public void removeAttribute(RName attrib) {
        attributes.remove(attrib);
    }

    /** This attribute's name */
	final RName                     name; 
          boolean                   readonly;
    
    HashMap<RName,REntityEntry>     attributes; 
	ArrayList<IRObject>             values       = new ArrayList<IRObject>();
	String                          comment      = "";
	
	
    
	/**
     * @return the comment
     */
    public String getComment() {
        return comment;
    }

    /**
     * @param comment the comment to set
     */
    public void setComment(String comment) {
        this.comment = comment;
    }

    public Set<RName> getAttributeSet(){
	    return attributes.keySet();
	}
	
    /**
     * A readonly entity cannot be modified at run time.
     */
    public boolean isReadOnly(){ return readonly; }
    
    /**
     * All reference Entities (i.e. those from which we clone instances) have 
     * an id of zero.  Clones have an id that is non zero, and unique.
     */
    final int                             id;
    
    /**
     * Returns the ID number for this instance of the entity.  Entities with
     * an ID of zero are reference entities.
     * @return
     */
    public int getID(){ return id;} 
    
	/**
     * Returns an interator that provides all the names of all the attributes for this entity. 
	 */
	public Iterator<RName> getAttributeIterator(){
		return attributes.keySet().iterator();
	}
	/**
     * Checks to see if the given attribute is defined by this entity. 
     * @param attName
     * @return
	 */
	public boolean containsAttribute(RName attName){
		return attributes.containsKey(attName);
	}
	
    /**
     * Create a clone of an Entity.
     *
     */
    public REntity( boolean _readonly, REntity entity, IRSession s) throws RulesException{
        id = s.getUniqueID();
        readonly   = _readonly;
        name       = entity.name;
        attributes = entity.attributes;
        values     = new ArrayList<IRObject>(entity.values);
        
        put(name,this);                         //Patch up the self reference to point to self.
        put(mappingKey,RNull.getRNull());       //Clear the mapping Key

        for(int i=0;i<values.size();i++){
            IRObject value     = values.get(i);
            if(value == this){          // make a clone of everything one level down, but don't clone ourselves.            
                values.set(i,value);    // The clone references the same entity
            }else{
                values.set(i,value.clone(s));
            }
        }
   }
    
    /**
     * Regular Constructor.  This should only be called when building the EntityFactory.
     * However, we make it public so the EntityFactory can be defined in the Session
     * package.  We might like to reconsider that decision some time in the future.
     * 
     * @param _name
     */
	public REntity( int id, boolean _readonly, RName _name) {
        this.id = id;
        readonly   = _readonly;
		name       = _name;
        attributes = new HashMap<RName,REntityEntry>();
        this.addAttribute(_name, "", this, false, true, type(),null,"Self Reference","");  // Add a reference to self!
        this.addAttribute(mappingKey,"",RNull.getRNull(),false, true, iString,null,"Mapping Key","");
	}

	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#getName()
     */
	public RName getName(){
		return name;
	}
	
    /**
     * Adds this REntityEntry to this Entity.  If an EntityEntry already exists, it will
     * be replaced by this new one, no questions asked.  The assumption is that one is
     * editing the Attributes of the Entity, and the latest is the the right one to keep.
     * 
     * @param entry The new Attribute meta data.
     * 
     */
    public void addAttribute(REntityEntry entry){
       REntityEntry oldentry = getEntry(entry.attribute);
       if(oldentry!=null){                      // If the attribute already exists
           entry.index = oldentry.index;        //   replace the old one (keep their index)
       }else{
           entry.index = values.size();         // If the attribute is new, make a new
           values.add(RNull.getRNull());        //   value index.
       }
       if(entry.defaultvalue==null){            // Update with the default value.
           values.set(entry.index, RNull.getRNull());
       }else{
           values.set(entry.index, entry.defaultvalue);
       }
       
       attributes.put(entry.attribute,entry);   // Put the new Entity Entry into this Entity.
       
    }
    
	/**
	 * This adds an attribute into the REntity.  This is called by the Entity
	 * construction during the loading of the Entity Description Dictionary.
	 * 
	 * @param defaultvalue Default value for this tag
	 * @param writable If true, the attribute is writable by decision tables
	 * @param readable If true, the attribute is readable by decision tables
	 * @param type
	 * @return null if successful, and an Error string if it failed.
	 */
	public String addAttribute(RName attributeName, 
	        String   defaulttxt, 
	        IRObject defaultvalue,
	        boolean  writable,
	        boolean  readable,
	        int      type,
	        String   subtype,
	        String   comment,
	        String   input){
        REntityEntry entry = getEntry(attributeName);
        if(entry==null){
    		int index = values.size();
    		if(defaultvalue==null){
                values.add(RNull.getRNull());
            }else{
                values.add(defaultvalue);
            }
    		REntityEntry newEntry = new REntityEntry(
    		        this,
    		        attributeName,
    		        defaulttxt, 
    		        defaultvalue,
    		        writable,
    		        readable,
    		        type,
    		        subtype,
    		        index,
    		        comment,
    		        input);
    		attributes.put(attributeName,newEntry);
            return null;
        }
        if(entry.type!=type){
            String type1 ="";
            String type2 ="";
            try{
                type1="("+  RSession.typeInt2Str(entry.type)+") ";
            }catch(RulesException e){}
            try{
                type2="("+  RSession.typeInt2Str(type)+")";
            }catch(RulesException e){}
            
            return "The entity '"+name.stringValue()+
                   "' has an attribute '"+attributeName.stringValue()+
                   "' with two types: "+type1+" and "+ type2+ "\n";                    
        }
        return null;    // Entry already matches what we already have.
	}
	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#put(com.dtrules.interpreter.RName, com.dtrules.interpreter.IRObject)
     */
	public void put( RName attrib, IRObject value) throws RulesException {
		REntityEntry entry = (REntityEntry)attributes.get(attrib);
		if(entry==null)throw new RulesException("Undefined", "REntity.put()", "Undefined Attribute "+attrib+" in Entity: "+name);
		if(value.type()!= iNull && entry.type != value.type()){
            switch(entry.type) {
                case iInteger :         value = value.rIntegerValue();          break;
                case iDouble :          value = value.rDoubleValue();           break;
                case iBoolean :         value = value.rBooleanValue();          break;
                //case iDecisiontable : value = value.rDecisiontableValue();    break;
                case iEntity :          value = value.rEntityValue();           break;      
                //case iMark :          value = value.rMarkValue();             break;
                case iName :            value = value.rNameValue();             break;
                //case iOperator :      value = value.rOperatorValue();         break;
                case iString :          value = value.rStringValue();           break;
                case iTime :            value = value.rTimeValue();             break;                    
            }
		}
		values.set(entry.index,value);
	}
	
	/**
     * Looks up the name of an attribute,
     * and returns the associated value.  
     * If no value is defined, returns a null. 
	 */
	public IRObject get(String attribName)
	{
		return get(RName.getRName(attribName));
	}
	
	/**
     * Looks up the given name, and returns the associated value.  If
     * no value is defined, returns a null. 
	 */
	public IRObject get(RName attrib) {
	   REntityEntry entry = (REntityEntry)attributes.get(attrib);
	   if(entry==null)return null;
	   return (IRObject) values.get(entry.index);
	}
	
	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#get(int)
     */
	public IRObject get(int i) {
		return (IRObject) values.get(i);
	}
	
	/**
	 * Sets a value in the values array.  Should only be used
	 * RARELY outside of REntity.
	 * @param i
	 * @param v
	 */
	public void set(int i, IRObject v){
		values.set(i, v);
	}
	/**
     * Returns an object that describes all the information we track about an 
     * Entity Attribute (the key to get its value).  If the attribute is undefined,
     * then a  null is returned. 
	 */
	public REntityEntry getEntry(RName attrib) {
		   REntityEntry entry = (REntityEntry)attributes.get(attrib);
		   return entry;
	}	
	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#getValue(int)
     */
	public IRObject getValue(int index) {
		return (IRObject) values.get(index);
	}
	
	
	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#postFix()
     */
	public String postFix() {
		return "/"+name.stringValue()+" "+id+" createEntity ";
	}

	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#stringValue()
     */
	public String stringValue() {
		return name.stringValue();
	}

	/* (non-Javadoc)
     * @see com.dtrules.entity.IREntity#type()
     */
	public int type() {
		return iEntity;
	}

	public String toString(){
		String v = name.stringValue()+" = {";
        Iterator<RName> ia = getAttributeIterator();
        while(ia.hasNext()){
            RName    n = ia.next();
            IRObject o = get(n);
            if(o==null){                // Protect ourselves from nulls.
                o = RNull.getRNull();
            }
            v +=n.stringValue()+" = "+get(n).stringValue()+"  ";
        }
        v +="}";
        return v;
        
	}

    public IRObject clone(IRSession s) throws RulesException {
        if(readonly)return this;
        return new REntity(false,this,s);
    }

    /**
     * Returns itself
     */
    public IREntity rEntityValue() throws RulesException {
       return this;
    }
    
    public void writeXML(PrintStream p) throws RulesException {
        Iterator<RName> attribs = getAttributeIterator();
        p.println();
        while(attribs.hasNext()){
           RName attrib = attribs.next();
           if(attrib.equals(name))continue;     // Skip the self reference.
           REntityEntry entry = attributes.get(attrib);
           p.print("<entity attribute=\"");
           p.print(attrib.stringValue());
           p.print("\" type =\"");
           p.print(RSession.typeInt2Str(entry.type));
           p.print("\" cdd_default_value=\"");
           p.print(entry.defaulttxt);
           p.print("\" cdd_i_c=\"");
           p.print(entry.writable?"c":"i");
           p.print("\" parseStr=\"\">");
           p.print(name.stringValue());
           p.println("</entity>");
        }
    }
    RXmlValue rXmlValue;
    
    /**
     * Get XML Node for this entity
     */
    public RXmlValue getRXmlValue(){
        return rXmlValue;
    }
    
    /**
     * Set the XML Node for this entity
     */
    public void setRXmlValue(RXmlValue rXmlValue){
        this.rXmlValue = rXmlValue;
    }

    /**
     * If the Entity is associated with an XML Node, that node
     * is returned.  Otherwise, a null is returned.
     */
    public IRObject rXmlValue() throws RulesException {
        if(getRXmlValue()!= null) return getRXmlValue();
        return RNull.getRNull();
    }

    /* (non-Javadoc)
     * @see com.dtrules.interpreter.ARObject#xmlTagValue()
     */
    @Override
    public XMLNode xmlTagValue() throws RulesException {
        return rXmlValue.xmlTagValue();
    }
    
}
