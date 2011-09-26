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

  
package com.dtrules.entity;

import java.io.PrintStream;
import java.util.Collection;
import java.util.Iterator;
import java.util.List;
import java.util.Set;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RType;
import com.dtrules.interpreter.RXmlValue;
import com.dtrules.session.IRSession;

public interface IREntity extends IRObject{

    /**
     * Get XML Node
     */
    public RXmlValue getRXmlValue();
    
    /**
     * Set the XML value associated with this entity
     */
    public void setRXmlValue(RXmlValue rXmlValue);
    
    /**
     * This is the attribute name used by DTRules to map data from 
     * XML files into Rules Engine Entitites.
     * 
     */
    public  final RName mappingKey = RName.getRName("mapping*key");
    /**
     * Returns if the Entity is read only or not
     */
    public  boolean isReadOnly();
    /**
     * Returns an iterator for the attributes of an REntity.  Each Key
     * is itself an RName.
     */
    public abstract Iterator<RName> getAttributeIterator();

    /**
     * Remove the attribute of the given name from this Entity.  Should only be
     * used by Rules Maintanence code, not by runtime code.
     * @param attrib attribute name to be removed.
     */
    public abstract void removeAttribute(RName attrib);
    
    /**
     * Add the attribute defined by this REntityEntry to this Entity.  The assumption
     * is that everything in the entry is set up correctly, and only the index to the
     * value needs to be adjusted (and then only if the attribute isn't defined already).
     * 
     * @param entry Entry providing the meta data for a new attribute.
     */
    public void addAttribute(REntityEntry entry);
     
    /**
     * This adds an attribute into the REntity.  This is called by the Entity
     * construction during the loading of the Entity Description Dictionary.
     * 
     * @param defaultvalue
     * @param writable
     * @param type
     * @return Error string if the add failed, or a null on success.
     */
    public String addAttribute( 
            RName attributeName, 
            String defaulttxt, 
            IRObject defaultvalue,
            boolean writable, 
            boolean readable, 
            RType type, 
            String subtype, 
            String comment, 
            String input,
            String output);
  
    /**
     * Returns the name of this entity. 
     */
    public abstract RName getName();

    /**
     * Here is where we set a value of an attribute within an Entity.  We take the
     * attribute name, look up that attribute and insure that the value type matches
     * the type expected by the attribute entry.  All types accept a RNull value.
     * 
     * Some data conversions require the session.
     * 
     * @param attrib
     * @param value
     * @throws RulesException
     */
    public abstract void put(IRSession session, RName attrib, IRObject value)
            throws RulesException;

    /**
     * Returns an IRObject if this Entity defines an attribute of the given name.
     * Otherwise it returns a null.  This method should be avoided if a RName is
     * easily available.
     * @param attrib
     * @return
     */
    public IRObject get(String attribName);
    
    /**
     * Returns an IRObject if this Entity defines an attribute of the given name.
     * Otherwise it returns a null.
     * @param attrib
     * @return
     */
    public abstract IRObject get(RName attrib);

    /**
     * Returns the indexed value of a key/value pair.  Sometimes we look at the 
     * EntityEntry before we grab the object.  This avoids an extra hash lookup.
     * @param i
     * @return
     */
    public abstract IRObject get(int i);

    /**
     * This method returns the Entry for an attribute.  An Entry allows the caller
     * to see the type and other information about the attribute in addition to its
     * value.  If the attribute is undefined, a null is returned.
     * @param attrib
     * @return
     */
    public abstract REntityEntry getEntry(RName attrib);

    /**
     * This gets all the entity entries for an entity
     * @return
     */
	public Collection<REntityEntry> getEntries();
    /**
     * Then sometimes they change their mind and want the value from an REntityEntry
     * @param index
     * @return
     */
    public abstract IRObject getValue(int index);
    
    /**
     * And sometimes they just want a bunch of objects
     * @return
     */
    public abstract List<IRObject> getValues();
 
    /**
     * Returns the ID number for this instance of the entity.  Entities with
     * an ID of zero are reference entities.
     * @return
     */
    public int getID();
    
    /**
     * Checks to see if the given attribute is defined by this entity. 
     * @param attName
     * @return
     */
    public boolean containsAttribute(RName attName);
    
    /**
     * Sets a value into the values array.  Should not be called directly,
     * but only through DTState.  We should refactor to enforce this.
     */
    public void set(int i, IRObject v);
    
    /**
     * Writes the XML representation of this Entity to the given printstream.
     * All the attributes are written along with their default values.  The self
     * referential attribute is not written.
     * @param p printstream to which the XML representation of this Entity is written.
     */
    public void writeXML(PrintStream p) throws RulesException;
    
    /**
     * Provides a set of keys for the attributes defined by this entity.
     * @return
     */
    public Set<RName> getAttributeSet();
    
    /**
     * Get the comment on this entity
     * @return
     */
    public String getComment();
    
    /*
     * Set the comment on this entity
     */
    public void setComment(String comment);
}