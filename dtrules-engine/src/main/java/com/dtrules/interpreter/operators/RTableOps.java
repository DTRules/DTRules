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

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTable;
import com.dtrules.session.DTState;
import com.dtrules.session.RSession;

public class RTableOps {
	 static {
	     	new GetWithKey();    new GetWithKeys(); 
	     	new SetWithKeys();   new SetWithKey();
	     	new GetKeysArray ();
	     	new NewTable();      new SetDescription();   
	     	new Translate ();
	     	new ClearTable();
	    }
     /**
      * ( table, index1, index2, ... lookup -- result )
      * Does a lookup (of however many keys) and returns the result.  An
      * Error will be thrown if any index but the last fails to return
      * an RTable object
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     public static class GetWithKeys extends ROperator {
         GetWithKeys(){ super("getwithkeys");}
         
         public void execute(DTState state) throws RulesException {
             int cnt = 0;
             int d = state.ddepth();
             while(state.getds(--d).type().getId()!=iTable)cnt++;
             RName []keys = new RName[cnt];
             for(int i=0;i<cnt;i++){
                keys[i]= state.datapop().rNameValue();
             }
             RTable rtable = state.datapop().rTableValue();
             state.datapush(rtable.getValue(keys));
         }
     }

     /**
      * ( table, key -- result )
      * Does a lookup of a single key and returns the result.  An
      * Error will be thrown if any index but the last fails to return
      * an RTable object
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     public static class GetWithKey extends ROperator {
         GetWithKey(){ super("getwithkey");}
         
         public void execute(DTState state) throws RulesException {
             IRObject key = state.datapop();
             RTable rtable = state.datapop().rTableValue();
             state.datapush(rtable.getValue(key));
         }
     }
     
     
     /**
      * (table key value --> ) Asserts into the table the given value using
      * the given key.
      * @author Paul Snow
      *
      */
     public static class SetWithKey extends ROperator {
         SetWithKey(){ super("setwithkey"); }
      
         public void execute(DTState state) throws RulesException {
             
             IRObject   v      = state.datapop();                // Get the value to store
             IRObject   key    = state.datapop().rNameValue();
             RTable     rtable = state.datapop().rTableValue();
             rtable.setValue(key,v);
             
         }
     }
     
     /**
      * ( table, index1, index2, ... , indexN value set -- )
      * Takes a table, and a set of indexes. The Nth index is used to
      * set the value.  None of the indexes can be a table, i.e. we assume
      * the deepest parameter to this call is the table.
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     public static class SetWithKeys extends ROperator {
         SetWithKeys(){ super("setwithkeys");}
         
         public void execute(DTState state) throws RulesException {
             int       cnt = 0;                             // We keep a count of the keys.
             IRObject  v   = state.datapop();               // Get the value to store
             int       d   = state.ddepth()-1;              // Get current depth of data stack less one for the value.
             
             while(state.getds(--d).type().getId()==iTable)cnt++;   // Count the keys (index1, index2, etc.)
             
             if(cnt != 1){
                 IRObject []keys = new IRObject[cnt];           // Get an array big enough to hold the keys
                 
                 for(int i=0;i<cnt;i++){                        // Get all the keys off the data stack
                    keys[i]= state.datapop();
                 }
                 
                 RTable rtable = state.datapop().rTableValue();
                 
                 rtable.setValue(state,keys, v);                // Set the value.
             }else{
                 IRObject   key    = state.datapop();
                 RTable     rtable = state.datapop().rTableValue();
                 rtable.setValue(key,v);
             }
             
             
         }
     }
     /**
      * ( table -- Array ) Returns an array holding all the
      * keys in a table.
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     public static class GetKeysArray extends ROperator {
         GetKeysArray(){ super("getkeysarray");}
         
         public void execute(DTState state) throws RulesException {
             RTable rtable = state.datapop().rTableValue();
             state.datapush(rtable.getKeys(state));
         }
     }
     /**
      * ( table -- ) Clear all the entries from the given table
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     public static class ClearTable extends ROperator {
         ClearTable(){ super("cleartable");}
         
         public void execute(DTState state) throws RulesException {
             RTable rtable = state.datapop().rTableValue();
             rtable.getTable().clear();
         }
     }
          
     /**
      *  ( Name Type -- Table ) newTable
      *  Name is a name for the table
      */
     public static class NewTable extends ROperator {
         NewTable(){ super("newtable");}
         
         public void execute(DTState state) throws RulesException {
             String type  = state.datapop().stringValue();
             RName  name  = state.datapop().rNameValue();
             RTable table = RTable.newRTable(state.getSession().getEntityFactory(),name,"");
             state.datapush(table);
         }
     }
     
     /**
      * ( Table description -- ) setDescription
      */
     public static class SetDescription extends ROperator {
         SetDescription() { super("setdescription"); }
         public void execute(DTState state) throws RulesException {
             RString description  = state.datapop().rStringValue();
             RTable  table        = state.datapop().rTableValue();
             table.setDescription(description);
         }
     }
     /**
      * ( Table keyArray boolean -- valueArray ) translate
      * All the keys provided by the keyArray are looked up in the
      * given Table.  If a key is not found in the table, it is 
      * ignored.  If a key is found, then that value is added to
      * the valueArray.
      * 
      * If the boolean is true, then duplicates are allowed in
      * the valueArray.  If boolean is false, then only unique 
      * values are returned.
      * 
      * @author paul snow
      * 
      */
     public static class Translate extends ROperator {
         Translate(){super("translate"); }
         public void execute(DTState state) throws RulesException {
             boolean duplicates = state.datapop().booleanValue();
             RArray  keys       = state.datapop().rArrayValue();
             RTable  table      = state.datapop().rTableValue();
             RArray valueArray = RArray.newArray(state.getSession(),duplicates,false);
             for(IRObject key : keys){
                 if(table.containsKey(key)){
                    IRObject o = table.getValue(key);
                    valueArray.add(o);
                    if (state.testState(DTState.TRACE)) {
                        state.traceInfo("addto", "arrayId", valueArray.getID() + "",
                                o.postFix());
                    }

                 }
             }
             state.datapush(valueArray);
         }
     }
     
}