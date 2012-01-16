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

package com.dtrules.interpreter.operators;

import java.util.ArrayList;
import java.util.List;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RMark;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.session.DTState;

public class RArrayOps {
	static {
		new Addto();
		new Addat();
		new Remove();
		new Removeat();
		new Getat();
		new Newarray();
		new Length();
		new Memberof();
		new Mark();
		new Arraytomark();
		new Copyelements();
		new DeepCopy();
		new Sortarray();
		new Sortentities();
		new Add_no_dups();
		new Clear();
		new Merge();
		new Randomize();
		new AddArray();
		new Tokenize();
		new Intersection();
		new Intersects();
		new FindMatch();
	}

	/**
	 * tokenize ( String1 String2 --> array ) String1 is a string to tokenize
	 * String2 is a regular expression defining these tokens array is an array
	 * of strings that results when String1 is tokenized.
	 * 
	 * @author ps24876
	 * 
	 */
	public static class Tokenize extends ROperator {
		Tokenize() {
			super("tokenize");
		}

		public void execute(DTState state) throws RulesException {

			String pattern = state.datapop().stringValue();
			IRObject obj1 = state.datapop();
			String v = "";
			if (obj1.type().getId() != IRObject.iNull) {
				v = obj1.stringValue().trim();
			}
			String[] results = v.split(pattern);

			RArray r = RArray.newArray(state.getSession(), false, false);
			for (String t : results) {
				r.add(RString.newRString(t));
				if (state.testState(DTState.TRACE)) {
					state.traceInfo("addto", "arrayId", r.getID() + "", t);
				}
			}
			state.datapush(r);
		}
	}

	/**
	 * addto( Array Value --> ) Addto Operator, adds an element to an array
	 * 
	 * @author Paul Snow
	 * 
	 */
	public static class Addto extends ROperator {
		Addto() {
			super("addto");
		}

		public void execute(DTState state) throws RulesException {
			IRObject value = state.datapop();
			RArray rarray = state.datapop().rArrayValue();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("addto", "arrayId", rarray.getID() + "",
						value.postFix());
			}
			rarray.add(value);
		}
	}

	/**
	 * findmatch
	 * if found:     ( name1 value1 name2 value2 name3 value3 array --> entity true) 
	 * if not found: ( name1 value1 name2 value2 name3 value3 array --> null   false) 
	 * 
	 * Looks through a list of entities, and returns the entity that matches
	 * the three given values.  If any of the names are null, then that one
	 * always matches.
	 * 
	 * @author Paul Snow
	 * 
	 */
	public static class FindMatch extends ROperator {
		FindMatch() {
			super("findmatch");
		}

		public void execute(DTState state) throws RulesException {
			RArray array = state.datapop().rArrayValue();
			IRObject v3 = state.datapop();
			IRObject n3 = state.datapop();
			IRObject v2 = state.datapop();
			IRObject n2 = state.datapop();
			IRObject v1 = state.datapop();
			IRObject n1 = state.datapop();
			
			for(IRObject ie : array){
				IREntity e = ie.rEntityValue();
				
				if(n3.type().getId() != RNull.type.getId()){
					IRObject v = e.get(n3.rNameValue());
					if(!v.equals(v3)){
						continue;
					}
				}
				if(n2.type().getId() != RNull.type.getId()){
					IRObject v = e.get(n2.rNameValue());
					if(!v.equals(v2)){
						continue;
					}
				}
				if(n1.type().getId() != RNull.type.getId()){
					IRObject v = e.get(n1.rNameValue());
					if(!v.equals(v1)){
						continue;
					}
				}
				state.datapush(e);
				state.datapush(RBoolean.getRBoolean(true));
				return;
			}
			state.datapush(RNull.getRNull());
			state.datapush(RBoolean.getRBoolean(false));
			return;
			
		}
	}

	
	
	/**
	 * addat( Array int Value --> ) Addat Operator, adds an element to an array
	 * at the given location
	 */
	public static class Addat extends ROperator {
		Addat() {
			super("addat");
		}

		public void execute(DTState state) throws RulesException {
			IRObject value = state.datapop();
			int position = state.datapop().intValue();
			RArray rarray = state.datapop().rArrayValue();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("addat", "arrayId", rarray.getID() + "",
						"index", position + "", value.postFix());
			}
			rarray.add(position, value);

		}
	}

	/**
	 * remove( Array Value --> boolean ) Remove Operator, removes all elements
	 * from an array that match the value. Returns a true if at least one
	 * element was removed, and a false otherwise.
	 */
	public static class Remove extends ROperator {
		Remove() {
			super("remove");
		}

		public void execute(DTState state) throws RulesException {
			IRObject value = state.datapop();
			RArray rarray = (RArray) state.datapop();
			List<IRObject> array = rarray.arrayValue();
			boolean removed = false;
			if (value != null) {
				for (int i = 0; i < array.size();) {
					if (value.equals((IRObject) array.get(i))) {
						if (state.testState(DTState.TRACE)) {
							state.traceInfo("removed", "arrayId",
									rarray.getID() + "", value.postFix());
						}
						array.remove(i);
						removed = true;
					} else {
						i++;
					}
				}
			}
			state.datapush(RBoolean.getRBoolean(removed)); // Return indicater
															// that something
															// was removed.
		}
	}

	/**
	 * removeat( Array int --> boolean ) Removeat Operator, removes an element
	 * from an array at the given location. Returns true upon success
	 */
	public static class Removeat extends ROperator {
		Removeat() {
			super("removeat");
		}

		public void execute(DTState state) throws RulesException {
			int position = state.datapop().intValue();
			RArray rarray = (RArray) state.datapop();
			if (position >= rarray.size()) {
				state.datapush(RBoolean.getRBoolean(false));
			}

			List<IRObject> array = rarray.arrayValue();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("removed", "arrayId", rarray.getID() + "",
						"position", position + "", null);
			}

			array.remove(position);
			if (state.testState(DTState.TRACE)) {
                state.traceInfo("removedat", "arrayId",
                        rarray.getID() + "", position+"");
            }
			state.datapush(RBoolean.getRBoolean(true));
		}
	}

	/**
	 * getat( Array int --> value) Getat Operator, gets an element from an array
	 * at the given location
	 */
	public static class Getat extends ROperator {
		Getat() {
			super("getat");
		}

		public void execute(DTState state) throws RulesException {
			int position = state.datapop().intValue();
			List<IRObject> array = state.datapop().arrayValue();
			state.datapush((IRObject) array.get(position));
		}
	}

	/**
	 * newarray( --> Array) Newarray Operator, returns a new empty array
	 */
	public static class Newarray extends ROperator {
		Newarray() {
			super("newarray");
		}

		public void execute(DTState state) throws RulesException {
			IRObject irobject = RArray.newArray(state.getSession(), true, false);
			state.datapush(irobject);
		}
	}

	/**
	 * length( Array --> int) Length Operator, returns the size of the array
	 */
	public static class Length extends ROperator {
		Length() {
			super("length");
		}

		public void execute(DTState state) throws RulesException {
			List<IRObject> array = state.datapop().arrayValue();
			state.datapush(RInteger.getRIntegerValue(array.size()));
		}
	}

	/**
	 * memberof( Array Value --> boolean) Memberof Operator, returns true if the
	 * element found in the array
	 */
	public static class Memberof extends ROperator {
		Memberof() {
			super("memberof");
		}

		public void execute(DTState state) throws RulesException {
			IRObject value = state.datapop();
			List<IRObject> array = state.datapop().arrayValue();
			boolean found = false;
			if (value != null) {
				for (int i = 0; i < array.size(); i++) {
					if (value.equals((IRObject) array.get(i))) {
						found = true;
						break;
					}
				}
			}
			state.datapush(RBoolean.getRBoolean(found));
		}
	}

	/**
	 * copyelements( Array --> newarray) Copyelements Operator, returns the copy
	 * of the array
	 */
	public static class Copyelements extends ROperator {
		Copyelements() {
			super("copyelements");
		}

		public void execute(DTState state) throws RulesException {
			RArray rarray = state.datapop().rArrayValue();
			RArray newRArray = rarray.clone(state.getSession()).rArrayValue();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("copyelements", "id", rarray.getID() + "",
						"newarrayid", newRArray.getID() + "", null);
				for (IRObject v : newRArray) {
					state.traceInfo("addto", "arrayId", rarray.getID() + "",
							v.postFix());
				}
			}

			state.datapush(newRArray);

		}
	}

	/**
	 * DeepCopy ( Array --> newarray) Copies the array, and makes copies of all
	 * of the elements as well.
	 */
	public static class DeepCopy extends ROperator {
		DeepCopy() {
			super("deepcopy");
		}

		public void execute(DTState state) throws RulesException {
			RArray rarray = state.datapop().rArrayValue();
			RArray newRArray = rarray.deepCopy(state.getSession())
					.rArrayValue();

			if (state.testState(DTState.TRACE)) {
				state.traceInfo("copyelements", "id", rarray.getID() + "",
						"newarrayid", newRArray.getID() + "", null);
				for (IRObject v : newRArray) {
					state.traceInfo("addto", "arrayId", rarray.getID() + "",
							v.postFix());
				}
			}

			state.datapush(newRArray);

		}
	}

	/**
	 * sortarray( Array boolean --> ) Sortarray Operator, sorts the array
	 * elements (asc is boolean is true)
	 */
	public static class Sortarray extends ROperator {
		Sortarray() {
			super("sortarray");
		}

		public void execute(DTState state) throws RulesException {
			boolean asc = state.datapop().booleanValue();
			int direction = asc ? 1 : -1;

			RArray rarray = state.datapop().rArrayValue();
			List<IRObject> array = rarray.arrayValue();

			if (state.testState(DTState.TRACE)) {
				state.traceInfo("sort", "length", array.size() + "", "arrayId",
						rarray.getID() + "", asc ? "true" : "false");
			}

			IRObject temp = null;
			int size = array.size();
			for (int i = 0; i < size - 1; i++) {
				for (int j = 0; j < size - 1 - i; j++) {
					if (((IRObject) array.get(j + 1)).compare((IRObject) array
							.get(j)) == direction) {
						temp = (IRObject) array.get(j);
						array.set(j, (IRObject) array.get(j + 1));
						array.set(j + 1, temp);
					}
				}
			}
		}
	}

	/**
	 * randomize ( Array --> ) Randomizes the order in the given array.
	 */
	public static class Randomize extends ROperator {
		Randomize() {
			super("randomize");
		}

		public void execute(DTState state) throws RulesException {
			List<IRObject> array = state.datapop().arrayValue();
			IRObject temp = null;
			int size = array.size();
			for (int i = 0; i < 10; i++) {
				for (int j = 0; j < size; j++) {
					int x = state.rand.nextInt(size);
					temp = (IRObject) array.get(j);
					array.set(j, (IRObject) array.get(x));
					array.set(x, temp);
				}
			}
		}
	}

	/**
	 * sortentities( Array field boolean --> ) Sortentities Operator,
	 */
	public static class Sortentities extends ROperator {
		Sortentities() {
			super("sortentities");
		}

		public void execute(DTState state) throws RulesException {
			boolean asc = state.datapop().booleanValue();
			RName rname = state.datapop().rNameValue();
			RArray rarray = state.datapop().rArrayValue();
			List<IRObject> array = rarray.arrayValue();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("sortentities", "length", array.size() + "",
						"by", rname.stringValue(), "arrayId", rarray.getID()
								+ "", asc ? "true" : "false");
			}
			REntity temp = null;
			int size = array.size();
			int greaterthan = asc ? 1 : -1;
			for (int i = 0; i < size; i++) {
				boolean done = true;
				for (int j = 0; j < size - 1 - i; j++) {
					try {
						IRObject v1 = ((REntity) array.get(j)).get(rname);
						IRObject v2 = ((REntity) array.get(j + 1)).get(rname);
						if (v1.compare(v2) == greaterthan) {
							temp = (REntity) array.get(j);
							array.set(j, (REntity) array.get(j + 1));
							array.set(j + 1, temp);
							done = false;
						}
					} catch (RuntimeException e) {
						throw new RulesException("undefined", "sort",
								"Field is undefined: " + rname);
					}
				}
				if (done)
					return;
			}
		}
	}

	/**
	 * add_no_dups( Array item --> ) Add_no_dups Operator, adds an element to an
	 * array if it is not present
	 */
	public static class Add_no_dups extends ROperator {
		Add_no_dups() {
			super("add_no_dups");
		}

		public void execute(DTState state) throws RulesException {

			IRObject value = state.datapop();
			RArray rArray = (RArray) state.datapop();
			for (IRObject v : rArray) {
				if (value.equals((IRObject) v)) {
					return;
				}
			}
			rArray.add(value);
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("addto", "arrayId", rArray.getID() + "",
						value.postFix());
			}

		}
	}

	/**
	 * clear( Array --> ) Clear Operator, removes all elements from the array
	 */
	public static class Clear extends ROperator {
		Clear() {
			super("clear");
		}

		public void execute(DTState state) throws RulesException {
			IRObject rarray = state.datapop();
			List<IRObject> array = rarray.arrayValue();
			array.clear();
			if (state.testState(DTState.TRACE)) {
				state.traceInfo("clear", "array", rarray.stringValue(), null);
			}
		}
	}

	/**
	 * merge( Array Array--> Array) Merge Operator, merges two array to one
	 * (Array1 elements followed by Array2 elements)
	 */
	public static class Merge extends ROperator {
		Merge() {
			super("merge");
		}

		public void execute(DTState state) throws RulesException {
			List<IRObject> array2 = state.datapop().arrayValue();
			List<IRObject> array1 = state.datapop().arrayValue();
			List<IRObject> newarray = new ArrayList<IRObject>();
			newarray.addAll(array1);
			newarray.addAll(array2);
			state.datapush(RArray.newArray(state.getSession(), false, newarray, false));
		}
	}

	/**
	 * Mark ( -- mark ) pushes a mark object onto the data stack.
	 * 
	 */
	public static class Mark extends ROperator {
		Mark() {
			super("mark");
		}

		public void execute(DTState state) throws RulesException {
			state.datapush(RMark.getMark());
		}
	}

	/**
	 * arraytomark ( mark obj obj ... -- array ) creates an array out of the
	 * elements on the data stack down to the first mark.
	 * 
	 */
	public static class Arraytomark extends ROperator {
		Arraytomark() {
			super("arraytomark");
		}

		public void execute(DTState state) throws RulesException {
			int im = state.ddepth() - 1;                                         // Index of top of stack

			while (im >= 0 && state.getds(im).type().getId() != iMark) im--;     // Index of mark
			
			RArray newarray = RArray.newArray(state.getSession(), true, false);  // Create a new array
			int newdepth = im++;                                                 // skip the mark (remember its index)
			
			while (im < state.ddepth()) {                                        // copy elements to the array
				IRObject o = state.getds(im++);
			    newarray.add(o);
				if (state.testState(DTState.TRACE)) {
	                state.traceInfo("addto", "arrayId", newarray.getID() + "",
	                        o.postFix());
	            }
			}
			
			while (newdepth < state.ddepth())
				state.datapop(); // toss elements AND mark
			
			state.datapush(newarray); // push the new array.
		}
	}

	/**
	 * ( array1 array2 boolean -- ) Adds all the elements of array1 to array2.
	 * If the boolean is true, then no duplicates are added to array2. If false,
	 * all elements (duplicate or not) are added to array2
	 * 
	 */
	public static class AddArray extends ROperator {
		AddArray() {
			super("addarray");
		}

		public void execute(DTState state) throws RulesException {
			boolean dups = state.datapop().booleanValue();
			RArray a2 = state.datapop().rArrayValue();
			RArray a1 = state.datapop().rArrayValue();

			for (IRObject o : a1) {
				if (dups || !a2.contains(o)) {
					a2.add(o);
				}
			}

		}
	}

	/**
	 * ( array1 array2 -- array3 ) Returns the intersection of Array1 and
	 * Array2. Returns an empty array if no members of array1 are found in
	 * array2. Duplicates are removed.
	 */
	public static class Intersection extends ROperator {
		Intersection() {
			super("intersection");
		}

		public void execute(DTState state) throws RulesException {

			RArray array1 = state.datapop().rArrayValue();
			RArray array2 = state.datapop().rArrayValue();

			RArray ret = RArray.newArray(state.getSession(), false, false);

			for (IRObject v : array1) {
				if (array2.contains(v)) {
					ret.add(v);
					if (state.testState(DTState.TRACE)) {
		                state.traceInfo("addto", "arrayId", ret.getID() + "",
		                        v.postFix());
		            }
				}
			}

			state.datapush(ret);
		}
	}

	/**
	 * ( array1 array2 -- boolean ) Returns true if any member of array1 is in
	 * array2
	 */
	public static class Intersects extends ROperator {
		Intersects() {
			super("intersects");
		}

		public void execute(DTState state) throws RulesException {

			RArray array1 = state.datapop().rArrayValue();
			RArray array2 = state.datapop().rArrayValue();

			for (IRObject v : array1) {
				if (array2.contains(v)) {
					state.datapush(RBoolean.getRBoolean(true));
					return;
				}
			}

			state.datapush(RBoolean.getRBoolean(false));
		}
	}
}