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
  
package com.dtrules.interpreter;

import java.util.ArrayList;
import java.util.Collection;
import java.util.ConcurrentModificationException;
import java.util.Iterator;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
/**
 * Immplements Arrays for Decision Tables.  Because we can't tag references,
 * we do the same trick here that we do for RNames, i.e. we create an executable
 * and a non-executable version of each array.  However, we only create one
 * ArrayList.
 * <br><br>
 * If dups is true, we allow duplicate references in the array.  Sometimes it
 * is pleasent to have an array whose values are all unique.  In that case, create
 * an array with dups set to false. <br>
 * <br>
 * @author ps24876
 *
 */
@SuppressWarnings("unchecked")
public class RArray extends ARObject implements Collection<IRObject> {
    final   ArrayList<IRObject> array;
    final   RArray    pair;
    final   boolean   executable;
    final   boolean   dups;  
    final   int       id;
    
    
    
    
    
    /**
     * @see java.util.Collection#addAll(java.util.Collection)
     */
    public boolean addAll(Collection<? extends IRObject> arg0) {
        return array.addAll(arg0);
    }
    /**
     * @see java.util.Collection#clear()
     */
    public void clear() {
        array.clear();
        
    }
    /**
     * @see java.util.Collection#contains(java.lang.Object)
     */
    public boolean contains(Object arg0) {
        if(arg0 instanceof IRObject) {
            for(IRObject o: array){
                try{                            // Really this isn't possible.
                    if (o.equals((IRObject)arg0))return true;
                }catch(RulesException e){}
            }            
        }
        return false;
    }
    /**
     * @see java.util.Collection#containsAll(java.util.Collection)
     */
    public boolean containsAll(Collection<?> arg0) {
        return array.containsAll(arg0);
    }
    /**
     * @see java.util.Collection#isEmpty()
     */
    public boolean isEmpty() {
        return array.isEmpty();
    }
    /**
     * @see java.util.Collection#iterator()
     */
    public Iterator<IRObject> iterator() {
        return array.iterator();
    }
    /**
     * @see java.util.Collection#remove(java.lang.Object)
     */
    public boolean remove(Object arg0) {
        return array.remove(arg0);
    }
    /**
     * @see java.util.Collection#removeAll(java.util.Collection)
     */
    public boolean removeAll(Collection<?> arg0) {
        return array.removeAll(arg0);
    }
    /**
     * @see java.util.Collection#retainAll(java.util.Collection)
     */
    public boolean retainAll(Collection<?> arg0) {
        return retainAll(arg0);
    }
    /**
     * @see java.util.Collection#toArray()
     */
    public Object[] toArray() {
        return array.toArray();
    }
    /**
     * @see java.util.Collection#toArray(T[])
     */
    public <T> T[] toArray(T[] arg0) {
        return array.toArray( arg0);
    }
    /**
     * Returns the id of this array.  Used by debuggers to analyze trace files
     * @return ID of the array
     */
    public int getID(){ return id; }
    /**
     * Constructor to create the core structure for an RArray.  We keep two "headers"
     * to every array.  One is the executable header, the other is the non-executable
     * header.
     * @param bogus
     * @param exectuable
     */
    protected RArray(int id, boolean duplicates, ArrayList thearray, RArray otherpair, boolean executable){
    	this.id         = id;
        this.array      = thearray;
    	this.executable = executable;
    	this.pair       = otherpair;
    	this.dups       = duplicates;
    }
    /**
     * Creates a RArray
     * @param id        A unique ID for all arrays.  Get this from the RSession object
     * @param duplicates
     * @param executable
     */
    public RArray(int id, boolean duplicates, boolean executable){
       this.id         = id;
       array           = new ArrayList();
       this.executable = executable;
       dups            = duplicates;
       pair            = new RArray(id,dups, array, this, !executable);
    }
    /**
     * Create an RArray from an arraylist
     * @param id
     * @param duplicates
     * @param thearray
     * @param executable
     */
    public RArray(int id, boolean duplicates, ArrayList thearray, boolean executable){
        this.id         = id;
        this.array      = thearray;
    	this.executable = executable;
        dups            = duplicates;
        pair            = new RArray(id,dups,thearray,this,!executable);
     }    
    
    /**
     * Return an Iterator for this array.  Returns a generic Iterator because
     * that can be cast to a typed Iterator.
     * @return
     */
    public Iterator getIterator(){ return array.iterator(); }
    
    /**
     * Returns the iArray type for Array Objects
     */
	public int type() {
		return iArray;
	}
	/**
	 * Add an element to this Array
	 */
    public boolean add(IRObject v){
    	if(!dups && array.contains(v))return false;
    	return array.add(v);
    }
    /**
     * Add an element to a particular location in this array
     * @param index
     * @param v
     */
    public void add(int index,IRObject v){
    	array.add(index,v);
    }
    /**
     * Delete an element at a particular location from this array
     * @param index
     */
    public void delete(int index){
    	array.remove(index);
    }
    /**
     * Find and remove a given element.  Removes the first element
     * matching.
     * @param v
     */
    public void remove(IRObject v){
    	array.remove(v);
    }
    /**
     * Get the element at the given index
     * @param index
     * @return
     * @throws RulesException
     */
    public IRObject get(int index) throws RulesException{
    	if(index<0 || index>= array.size()){
            throw new RulesException("Undefined","RArray","Index out of bounds");
    	}
    	return (IRObject) array.get(index);
    }
    /**
     * @see IRObject#arrayValue()
     */
	public ArrayList<IRObject> arrayValue() throws RulesException {
		return array;
	}
	/**
	 * @see IRObject#compare(IRObject)
	 */
	public boolean equals(IRObject o) throws RulesException {
		if(o.type() != iArray) return false;
		return ((RArray)o).array == array;
	}
	
	/**
	 * Generate the postfix for this array, while indicating
	 * the postfix element which generated the error, assuming
	 * the index yields that location.
	 * @param index
	 * @return
	 */
	private String generatePostfix(int index){
		String ps = "";
	       for(int i=0;i<array.size();i++){
	           if (i==index) ps += " ERROR==> ";
	           ps += array.get(i).postFix()+" ";
	           if (i==index) ps += " <== ";
	       }
	       return ps;
	}
	/**
	 * Implements the execution behavior of an RArray
	 */
	public void execute(DTState state) throws RulesException {
        int cnt = 0;  // A debugging aid.
        for(IRObject obj : this){
			if(obj.type()==iArray || !obj.isExecutable()){
				state.datapush(obj);
			}else{
			    try{
				   obj.execute(state);
			    }catch(ConcurrentModificationException e){
			       String ps = generatePostfix(cnt);
			       RulesException re = new RulesException("access error",array.get(cnt).postFix(),e.toString()+"\r\n"+
			    		   "This happens generally when you have attempted to modify an array\r\n "+
			    		   "which is in the context because you are iterating over its contents\r\n"+
			    		   "with a ForAll operator."+
			    		   "");
			       re.setPostfix(ps);
			       throw re;
			    }catch(RuntimeException e){
			       String ps = generatePostfix(cnt);
			       RulesException re = new RulesException("runtime error","RArray",e.toString());
			       re.setPostfix(ps);
			       throw re;
			    }catch(RulesException e){
			       String ps = generatePostfix(cnt);
			       e.setPostfix(ps);
			       throw e;
			    }
			}
            cnt++;
		}
		
	}
	/**
	 * @see IRObject#getExecutable()
	 */
	public IRObject getExecutable() {
		if(executable)return this;
		return pair;
	}
	/**
	 * @see IRObject#getNonExecutable()
	 */
	public IRObject getNonExecutable() {
		if(!executable)return this;
		return pair;
	}
	/**
	 * @see IRObject#isExecutable()
	 */
	public boolean isExecutable() {
		return executable;
	}
	/**
	 * @see IRObject#postFix()
	 */
	public String postFix() {
		StringBuffer result = new StringBuffer();
		result.append(executable?"{":"[");
		for (IRObject obj : array){
			result.append(obj.postFix());
			result.append(" ");
		}
        result.append(executable?"}":"]");
		return result.toString();
	}
	/**
	 * @see IRObject#stringValue()
	 */
	public String stringValue() {
		StringBuffer result = new StringBuffer();
		result.append(isExecutable()?"{ ":"[ ");
		for(IRObject obj : array){
			result.append(obj.stringValue());
			result.append(" ");
		}
		result.append(isExecutable()?"}":"]");
		return result.toString();
	}
	/**
	 * To string implementation for RArray object
	 */
	public String toString() {
		return stringValue();
	}
    
	/**
	 * returns the clone of this object
	 */
	public IRObject clone(IRSession session) {
		ArrayList<IRObject> newArray = new ArrayList<IRObject>();
		newArray.addAll(array);
		return new RArray(session.getUniqueID(), dups, newArray, executable);
	}

	/**
     * returns the clone of this object and of its elements
     */
    public IRObject deepCopy(IRSession session) throws RulesException {
        ArrayList<IRObject> newArray = new ArrayList<IRObject>();
        newArray.addAll(array);
        for(int i=0;i<newArray.size();i++){
            IRObject element = newArray.get(i);
            newArray.add(i,element.clone(session));
        }
        return new RArray(session.getUniqueID(), dups, newArray, executable);
    }

    /**
     * @see IRObject#rArrayValue()
     */
    public RArray rArrayValue() throws RulesException {
        return this;
    }
    
    /**
     * Returns the size of the array.
     */
    public int size()
    {
    	return this.array.size();
    }
}
