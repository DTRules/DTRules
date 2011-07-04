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
	     	new Lookup();    new Set();              new GetKeys ();
	     	new NewTable();  new SetDescription();   new Translate ();
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
     static class Lookup extends ROperator {
         Lookup(){ super("lookup");}
         
         public void execute(DTState state) throws RulesException {
             int cnt = 0;
             int d = state.ddepth();
             while(state.getds(--d).type()!=iTable)cnt++;
             RName []keys = new RName[cnt];
             for(int i=0;i<cnt;i++){
                keys[i]= state.datapop().rNameValue();
             }
             RTable rtable = state.datapop().rTableValue();
             state.datapush(rtable.getValue(keys));
         }
     }
     
     /**
      * ( table, index1, index2, ... , indexN value set -- )
      * Takes a table, and a set of indexes. The Nth index is used to
      * set the value.
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     static class Set extends ROperator {
         Set(){ super("set");}
         
         public void execute(DTState state) throws RulesException {
             int       cnt = 0;                             // We keep a count of the keys.
             IRObject  v   = state.datapop();               // Get the value to store
             int       d   = state.ddepth();                // Get current depth of data stack.
             
             while(state.getds(--d).type()==iName)cnt++;    // Count the keys (index1, index2, etc.)
             
             RName []keys = new RName[cnt];                 // Get an array big enough to hold the keys
             
             for(int i=0;i<cnt;i++){                        // Get all the keys off the data stack
                keys[i]= state.datapop().rNameValue();
             }
             
             RTable rtable = state.datapop().rTableValue();
             
             rtable.setValue(state,keys, v);                // Set the value.
         }
     }
     /**
      * ( table getKeys -- Array ) Returns an array holding all the
      * keys in a table.
      * 
      * @author paul snow
      * Aug 14, 2007
      *
      */
     static class GetKeys extends ROperator {
         GetKeys(){ super("getkeys");}
         
         public void execute(DTState state) throws RulesException {
             RTable rtable = state.datapop().rTableValue();
             state.datapush(rtable.getKeys(state));
         }
     }
          
     /**
      *  ( Name Type -- Table ) newTable
      *  Name is a name for the table
      */
     static class NewTable extends ROperator {
         NewTable(){ super("newtable");}
         
         public void execute(DTState state) throws RulesException {
             String type  = state.datapop().stringValue();
             RName  name  = state.datapop().rNameValue();
             int    itype = RSession.typeStr2Int(type,"","");
             RTable table = RTable.newRTable(state.getSession().getEntityFactory(),name,"",itype);
             state.datapush(table);
         }
     }
     
     /**
      * ( Table description -- ) setDescription
      */
     static class SetDescription extends ROperator {
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
     static class Translate extends ROperator {
         Translate(){super("translate"); }
         public void execute(DTState state) throws RulesException {
             boolean duplicates = state.datapop().booleanValue();
             RArray  keys       = state.datapop().rArrayValue();
             RTable  table      = state.datapop().rTableValue();
             RArray valueArray = new RArray(state.getSession().getUniqueID(),duplicates,false);
             for(IRObject irkey : keys){
                 RName key = irkey.rNameValue();
                 if(table.containsKey(key)){
                    valueArray.add(table.getValue(key));
                 }
             }
             state.datapush(valueArray);
         }
     }
     
}