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

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;

public class RTable extends ARObject {
    
	static RType type = RType.newType("table");

	private final int     id;
    private       RName   tablename;
    private       RString description;
    
    private final Map<IRObject, IRObject> table = new HashMap<IRObject,IRObject>();
    
    /**
     * Get the description of this table
     * @return the description
     */
    public RString getDescription() {
        return description;
    }

    /**
     * Set the description of this table
     */
    public void setDescription(RString description){
        this.description = description;
    }
    
    /**
     * Return the Unique ID for this table.
     * @return the id
     */
    public int getId() {
        return id;
    }

    /**
     * Return the HashMap for primary dimension of this table
     * @return the table
     */
    public Map<IRObject, IRObject> getTable() {
        return table;
    }

    /**
     * @return the tablename
     */
    public RName getTablename() {
        return tablename;
    }

    /**
     * Returns true if the key is contained in the RTable
     * @param key
     * @return
     */
    public boolean containsKey (IRObject key){
        return table.containsKey(key);
    }
    
    /**
     * Returns true if the value is contained in the RTable
     * @param value
     * @return
     */
    public boolean containsValue (IRObject value){
        return table.containsValue(value);
    }
    
    private RTable(EntityFactory ef, 
            RName  tablename, 
            String description) throws RulesException {
        this.tablename   = tablename;
        this.id          = ef.getUniqueID();
        this.description = RString.newRString(description);
    }
    /**
     * Factory method for creating an RTable
     * @param state
     * @param tablename
     * @param description
     * @param resultType
     * @return
     */
    static public RTable newRTable(EntityFactory ef, RName tablename, String description) throws RulesException{
        return new RTable(ef,tablename,description);
    }
    /**
     * This routine assumes that the string defines an Array of the 
     * form:
     *     {
     *       { key1 value1 }
     *       { key2 value2 }
     *       ...
     *       { keyn valuen }
     *     }
     * This routine compiles the given string, then calls the
     * set routine that takes an array of key value pairs and sets
     * them into the RTable    
     *     
     * @param values
     */
    public void setValues(IRSession session, String values)throws RulesException{
        RArray array = RString.compile(session, values, false).rArrayValue();
        setValues(array);
    }
    /**
     * This routine assumes that an Array of the 
     * form:
     *     {
     *       { key1 value1 }
     *       { key2 value2 }
     *       ...
     *       { keyn valuen }
     *     }
     * This routine takes an array of key value pairs and sets
     * them into the RTable    
     *     
     * @param values
     */
    public void setValues(RArray values) throws RulesException{
        for(IRObject irpair : values){
            RArray pair = irpair.rArrayValue();
            if(pair.size()!=2){
                throw new RulesException(
                        "Invalid_Table_Value",
                        "RTable.setValues",
                        "setValues expected an array of arrays giving pairs of values to assert into the Table");
            }
            IRObject key   = pair.get(0);
            IRObject value = pair.get(1);
            setValue(key, value);
        }
    }
    
    /**
     * Set a value with the given set of keys into the given table. 
     * @param keys
     * @param value
     * @throws RulesException
     */
    public void setValue(DTState state, IRObject[]keys, IRObject value) throws RulesException{
        IRObject v = this;
        for(int i=0;i<keys.length-1; i++){
            if(v.type()!=type){
                throw new RulesException("OutOfBounds","RTable","Invalid Number of Keys used with Table "+this.stringValue());
            }
            RTable   table = v.rTableValue();
            IRObject next  =  table.getValue(keys[i]);
            if(next == null){
                next = newRTable(state.getSession().getEntityFactory(),this.tablename,this.description.stringValue());
                table.setValue(keys[i], next);
            }
            v = (IRObject) next;
        }
        v.rTableValue().setValue(keys[keys.length-1], value);
    }
    
    public void setValue(IRObject key, IRObject value) throws RulesException{
        table.put(key, value);
    }
    
    public IRObject getValue(IRObject key) throws RulesException {
        IRObject v = this.table.get(key);
        if(v==null)return RNull.getRNull();
        return v;
    }
    
    public IRObject getValue(IRObject[]keys) throws RulesException {
        IRObject v = this;
        for(int i=0;i<keys.length; i++){
            if(v.type()!=type){
                throw new RulesException("OutOfBounds","RTable","Invalid Number of Keys used with Table "+this.stringValue());
            }
            RTable   table = v.rTableValue();
            IRObject next  =  table.getValue(keys[i]);
            if(next == null){
                return RNull.getRNull();
            }
            v = (IRObject) next;
        }
        return v;
    }
    
    public RArray getKeys (DTState state) throws RulesException {
        ArrayList <IRObject> keys = new ArrayList<IRObject>(table.keySet());
        int id = state.getSession().getUniqueID();
        return RArray.newArray(state.getSession(), true, keys,false);
    }
    
    public String stringValue() {
        if(tablename!=null) return tablename.stringValue();
        return "";
    }

	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}


    /* (non-Javadoc)
     * @see com.dtrules.interpreter.ARObject#rTableValue()
     */
    @Override
    public RTable rTableValue() throws RulesException {
        return this;
    }

    /* (non-Javadoc)
     * @see com.dtrules.interpreter.ARObject#tableValue()
     */
    public Map<IRObject, IRObject> tableValue() throws RulesException {
        return table;
    }

	
	public IRObject clone(IRSession s) throws RulesException {
		RTable newTable = newRTable(
				s.getEntityFactory(),tablename, 
				description.stringValue());
		newTable.getTable().putAll(table);
		return newTable;
	}
    
    
}
