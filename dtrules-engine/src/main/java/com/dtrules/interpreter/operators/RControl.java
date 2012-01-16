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
import java.util.Iterator;
import java.util.List;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;
/**
 * Defines math operators.
 * @author paul snow
 *
 */
@SuppressWarnings("unchecked")
public class RControl {
    static {
        new If();           new Ifelse();           new While();        
        new Forallr();      new Forall();           new For();          
        new Forr();         new Entityforall();     new Forfirst();  
        new Doloop();       new ForFirstElse();     new executeTable(); 
        new execute();      new Deallocate();       new Allocate();     
        new Localfetch();   new Localstore();       new PerformCatchError();
        new Lookup();       new PolicyStatements(); new ThrowException();
    }
    
    /**
     * ( body boolean -- ) executes the body if the boolean is true.
     * @author paul snow
     *
     */
    public static class If extends ROperator {
        If(){super("if");}

        public void execute(DTState state) throws RulesException {
            boolean  test = state.datapop().booleanValue();
            IRObject body = state.datapop();
            if(test)body.execute(state);
        }
    }
    /**
     * ( truebody falsebody test -- ) executes truebody if the boolean test is true, otherwise 
     * executes falsebody.
     * @author paul snow
     *
     */
    public static class Ifelse extends ROperator {
        Ifelse(){super("ifelse");}

        public void execute(DTState state) throws RulesException {
            boolean  test      = state.datapop().booleanValue();
            IRObject falsebody = state.datapop();
            IRObject truebody  = state.datapop();
            if(test){
                truebody.execute(state);
            }else{ 
                falsebody.execute(state);
            }
        }
    }
    
    /**
     * This is an internal operator which doesn't have any use outside of the
     * Rules Engine itself.  It is used to execute the conditions and actions 
     * for a table within the context as defined in the table.
     * 
     * ( DecisionTableName -- ) Takes the DecisionTable name and looks it up
     * in the decisiontable entity.  It then executes the table within that
     * decision table.  Because of this extra lookup, this path shouldn't be
     * used unless there actually is a context to be executed.
     * 
     * @author paul snow
     *
     */
    public static class executeTable extends ROperator {
        executeTable(){super("executetable");}

        public void execute(DTState state) throws RulesException {
            RName dtname = state.datapop().rNameValue();
            state.getSession().getEntityFactory().getDecisionTable(dtname).executeTable(state);
        }
    }
    /**
     * ( RObject -- ? ) executes the object on the top of the data stack.
     * The behavior of this operator is defined by the object executed.  Usually
     * the data stack will be left clean.
     * @author paul snow
     *
     */
    public static class execute extends ROperator {
        execute(){super("execute"); }
        
        public void execute(DTState state) throws RulesException {
            state.datapop().getExecutable().execute(state);
        }
    }
    
    /**
     * ( body test -- ) executes test.  If test returns true, executes body.  
     * Repeats until the test returns false.
     * 
     * @author paul snow
     *
     */
    public static class While extends ROperator {
        While(){super("while");}

        public void execute(DTState state) throws RulesException {
            IRObject test = state.datapop();         // Get the test
            IRObject body = state.datapop();         // Get the body
            
            test.execute(state);                     // evaluate the test
            while(state.datapop().booleanValue()){   // If true, keep looping
                body.execute(state);                 // execute the body.
                test.execute(state);                 // check the test again.
            }
            
           
        }
    }
    /**
     * ( body array -- ) execute the body for each entity in the array.
     * executes backwards through the array, and the entity can be removed
     * from the array by the body.
     * 
     * @author paul snow
     *
     */
    public static class Forallr extends ROperator {
        Forallr(){super("forallr");}

        public void execute(DTState state) throws RulesException {

            List<IRObject> array  = state.datapop().arrayValue(); // Get the array
            int            length = array.size();                 // get Array length.
            IRObject       body   = state.datapop();              // Get the body
            
            for(int i=length - 1;i>=0;i--){                  // For each element in array,
                 IRObject o = (IRObject) array.get(i);
                 int      t = o.type().getId();
                 if(t== iNull)continue;
                 if(t!=iEntity){
                    throw new RulesException("Type Check", "Forallr", "Encountered a non-Entity entry in array: "+o);
                 }  
                 state.entitypush((IREntity) o);
                 body.execute(state);
                 state.entitypop();
            }
        }
    }
    
    /**
     * ( body array -- ) executes the body for each entity in the array.  Uses an 
     * Iterator and executes forward in the array.  The entity cannot be removed from
     * the array by the body.
     * 
     * @author paul snow
     *
     */
    public static class Forall extends ROperator {
        Forall(){super("forall");}

        public void execute(DTState state) throws RulesException {
            
            RArray   array = state.datapop().rArrayValue();
            IRObject body = state.datapop();        // Get the body
            
            for(IRObject o : array){
                 int      t = o.type().getId();
                 if(t== iNull)continue;
                 if(t!=iEntity){
                     throw new RulesException("Type Check", "Forallr", "Encountered a non-Entity entry in array: "+o);
                 }  
                 state.entitypush((IREntity) o);
                 body.execute(state);
                 state.entitypop();
            }
        }
    }
    
    /**
     * ( body array -- ) Pushes each element on to the data stack, then executes the body.
     * @author paul snow
     *
     */
    public static class For extends ROperator {
        For(){super("for");}

        public void execute(DTState state) throws RulesException {
            RArray   list = state.datapop().rArrayValue();
            IRObject body = state.datapop();        // Get the body
            for(IRObject o : list){
                 state.datapush(o);
                 body.execute(state);
            }
        }
    }
    /**
     * ( body array  -- ) pushes each element on to the data stack, then executes the body.
     * Because the array is evaluated in reverse order, the element can be removed from the
     * array if you care to.
     * 
     * @author paul snow
     *
     */
    public static class Forr extends ROperator {
        Forr(){super("forr");}

        public void execute(DTState state) throws RulesException {
            
            List<IRObject> array  = state.datapop().arrayValue(); // Get the array
            int            length = array.size();                 // get Array length.
            IRObject       body   = state.datapop();              // Get the body
            
            for(int i=length-1;i>=0;i++){                    // For each element in array,
                 IRObject o = (IRObject) array.get(i);
                 state.datapush((IRObject) o);
                 body.execute(state);
            }
        }
    }
    /**
     * ( body entity -- ) executes the body for each key value pair in the Entity.
     * 
     * @author paul snow
     *
     */
    public static class Entityforall extends ROperator {
        Entityforall(){super("entityforall");}

        public void execute(DTState state) throws RulesException {
            IREntity        entity  = state.datapop().rEntityValue();  // Get the entity
            IRObject        body    = state.datapop();                    // Get the body
            Iterator<RName> keys    = entity.getAttributeIterator();      // Get the Attribute Iterator
            
            while(keys.hasNext()){                         // For each attribute 
                RName     n = (RName) keys.next();
                IRObject  v = entity.get(n);
                if(v!=null){
                    state.datapush(n);
                    state.datapush(v);
                    body.execute(state);
                }    
           }
        }
    }
    /**
     * ( body test array -- )
     * Each entity within the array is placed on the entity stack.  The test is evaluated, which should
     * return the boolean.  If true, the body is executed, and the loop stops. 
     * @author paul snow
     * Jan 10, 2007
     *
     */
    public static class Forfirst extends ROperator {
        Forfirst(){super("forfirst");}

        public void execute(DTState state) throws RulesException {
            RArray              array = state.datapop().rArrayValue();
            IRObject            test  = state.datapop();
            IRObject            body  = state.datapop();
            Iterator<IRObject>  ie    =  array.getIterator();
            while(ie.hasNext()){
                state.entitypush(ie.next().rEntityValue());
                test.execute(state);
                if(state.datapop().booleanValue()){
                    body.execute(state);
                    state.entitypop();
                    return;
                }
                state.entitypop();
            }
        }
    }

    /**
     * ( body1 body2 test array -- )
     * Each entity within the array is placed on the entity stack.  The test is 
     * evaluated, which should return the boolean.  If true, the body1 is executed, 
     * and the loop stops.  If no match is found, body2 is executed.  Kinda a 
     * default operation. 
     * @author paul snow
     * Jan 10, 2007
     *
     */
    public static class ForFirstElse extends ROperator {
        ForFirstElse(){super("forfirstelse");}

        public void execute(DTState state) throws RulesException {
            RArray   array = state.datapop().rArrayValue();
            IRObject test  = state.datapop();
            IRObject body2 = state.datapop();
            IRObject body1 = state.datapop();
            for(IRObject obj : array) {
                IREntity e = obj.rEntityValue();
                state.entitypush(e);
                test.execute(state);
                if(state.datapop().booleanValue()){
                    body1.execute(state);
                    state.entitypop();
                    return;
                }
                state.entitypop();
            }
            body2.execute(state);
        }
    }
    
    /**
     * ( body start increment limit -- ) Starts the index with the value "start".  If the
     * increment is positive and start is below the limit, the index
     * is pushed and body is executed, then the index is incremented
     * by increment.  If the increment is negative and stat is above
     * the limit, then the index is pushed and the body is executed,
     * then the index is decremented by the increment.  This continues
     * until the limit is reached.
     * 
     * @author paul snow
     * Jan 10, 2007
     *
     */
    public static class Doloop extends ROperator {
        Doloop(){super("doloop");}

        public void execute(DTState state) throws RulesException {
            int         limit     = state.datapop().intValue();
            int         increment = state.datapop().intValue();
            int         start     = state.datapop().intValue();
            IRObject    body      = state.datapop();
            if(increment>0){
                for(int i = start;i<limit;i+=increment){
                    state.cpush(RInteger.getRIntegerValue(i));
                    body.execute(state);
                    state.cpop();
                }
            }else{
                for(int i = start;i>limit;i+=increment){
                    state.cpush(RInteger.getRIntegerValue(i));
                    body.execute(state);
                    state.cpop();
                }
            }
        }
    }
    /**
     * ( value -- )
     * Allocates storage for a local variable.  The value is the initial
     * value for that variable.  In all reality, this is just a push of
     * a value to the control stack.
     * @author paul snow
     *
     */
    public static class Allocate extends ROperator {
        Allocate(){super("allocate"); alias("cpush");}

        public void execute(DTState state) throws RulesException {
            IRObject v = state.datapop();
            int index = state.cdepth()-state.getCurrentFrame();
            if(state.testState(DTState.TRACE)){
                state.traceInfo("allocate", "index",""+index, "value",v.postFix(),null);
            }
            state.cpush(v);
        }
    }  
    
    /**
     * ( -- value )
     * Deallocates storage for a local variable.  In all reality, this is
     * just a pop of a value from the control stack.
     * 
     * @author paul snow
     *
     */
    public static class Deallocate extends ROperator {
        Deallocate(){super("deallocate"); alias("cpop");}
        public void execute(DTState state) throws RulesException {
            state.datapush(state.cpop());
        }
    }
    /**
     * ( index -- IRObject ) fetches the value from the local variable
     * specified by the given index.  This is an offset from the currentframe. 
     * @author paul snow
     *
     */
    public static class Localfetch extends ROperator {
        Localfetch(){super("local@");}

        public void execute(DTState state) throws RulesException {
            int      index = state.datapop().intValue();
            IRObject value = state.getFrameValue(index);
            if(state.testState(DTState.TRACE) && state.testState(DTState.VERBOSE)){
                state.traceInfo("local_fetch", "index",index+"","value",value.stringValue(),null);
            }
            state.datapush(value);
        }
    }
    /**
     * (value index -- ) stores the value into the local variable specified
     * by the given index.  The index is an offset from the currentframe.
     * @author paul snow
     *
     */
    public static class Localstore extends ROperator {
        Localstore(){super("local!");}

        public void execute(DTState state) throws RulesException {
            int      index = state.datapop().intValue();
            IRObject value = state.datapop();
            if(state.testState(DTState.TRACE)){
                state.traceInfo("local_store", "index",index+"","value",value.stringValue(),null);
            }
            state.setFrameValue(index, value);
        }
    }

    /**
     * performCatchError (table error_table error_entity -- )
     * executes the given table.  If a RulesException is thrown, a RulesException
     * is created and put into the context.  Then the error_table is called.
     * <br><br>
     * Intended to support something like the following syntax:
     * <br><br>
     * perform TableX and on an error, add the ErrorDetails to the context and call Error_handling_table
     * @author paul snow
     *
     */
    public static class PerformCatchError extends ROperator {
        PerformCatchError(){super("performcatcherror");}

        private IRObject p(String v) { return RString.newRString(v==null?"":v); }
        private RName    n(String x) { return RName.getRName(x);}
        
        public void execute(DTState state) throws RulesException {
            RName    error        = state.datapop().rNameValue();
            RName    table        = state.datapop().rNameValue();
            
            try{
                state.find(table).execute(state);
            }catch(NullPointerException e){
                throw new RulesException("undefined", "PerformCatchError", 
                        "The table '"+table.stringValue()+"' is undefined");
            }catch(RulesException e){
                IRSession     session       = state.getSession();
                EntityFactory ef            = session.getEntityFactory();   
                IREntity      errorEntity   = ef.findRefEntity(error).clone(session).rEntityValue();
                state.entitypush(errorEntity);
                
                // If any of the following puts fail (because the given entity doesn't define them), then
                // simply carry on your merry way.  The user can define these fields if they need them,
                // and they don't need to define them if they don't need them.
                
                try { errorEntity.put(null, n("errortype"),     p(e.getErrortype()));                         }catch(RulesException ex){}
                try { errorEntity.put(null, n("location"),      p(e.getLocation()));                          }catch(RulesException ex){}
                try { errorEntity.put(null, n("message"),       p(e.getMessage()));                           }catch(RulesException ex){}
                try { errorEntity.put(null, n("decisionTable"), p(e.getDecisionTable()));                     }catch(RulesException ex){}
                try { errorEntity.put(null, n("formal"),        p(e.getErrortype()));                         }catch(RulesException ex){}
                try { errorEntity.put(null, n("postfix"),       p(e.getPostfix()));                           }catch(RulesException ex){}
                try { errorEntity.put(null, n("filename"),      p(e.getFilename()));                          }catch(RulesException ex){}
                try { errorEntity.put(null, n("section"),       p(e.getSection()));                           }catch(RulesException ex){}
                try { errorEntity.put(null, n("number"),        RInteger.getRIntegerValue(e.getNumber()));    }catch(RulesException ex){}
                
            }
        }
    }

    /**
     * ( name -- value ) Looks up and returns the value of the given attribute.
     * Even if the value found is executable, it is still pushed to the 
     * data stack.  If no value is found, an undefined exception is thrown.
     * 
     * @author paul snow
     *
     */
    public static class Lookup extends ROperator {
        Lookup(){super("lookup");}

        public void execute(DTState state) throws RulesException {
            RName name = state.datapop().rNameValue();
            IRObject value = state.find(name);
            if(value == null){
                throw new RulesException(
                        "undefined",
                        "Lookup",
                        "Could not find a value for "+name.stringValue()+" in the current context."
                );
            }
            state.datapush(value);
        }
    }

    /**
     * ( -- array ) Returns the list of policy statements in this decision table that
     * resulted in the execution of this particular action.
     * 
     * @author paul snow
     *
     */
    public static class PolicyStatements extends ROperator {
        PolicyStatements(){super("policystatements");}

        public void execute(DTState state) throws RulesException {
            RArray ra = RArray.newArray(state.getSession(),true,false);
            state.datapush(ra);
            if(!state.getCurrentTable().equals(state.getAnode().getrDecisionTable())){
                return;
            }

            ArrayList<Integer> columns = state.getAnode().getColumns();
            IRObject rps [] = state.getCurrentTable().getRpolicystatements();
            if(rps!= null) { 
                for(int column : columns){
                    if(column < rps.length){
                       IRObject ps = rps[column];
                       if(ps != null){
                          ps.execute(state);
                          ps = state.datapop();
                          if(ps != RNull.getRNull()){
                              ra.add(ps);
                              if (state.testState(DTState.TRACE)) {
                                  state.traceInfo("addto", "arrayId", ra.getID() + "",
                                          ps.postFix());
                              }
                          }
                       }
                    }
                }
                if(columns.size()==0){
                    IRObject ps = rps[0];
                    if(ps != null){
                       ps.execute(state);
                       ps = state.datapop();
                       if(ps != RNull.getRNull()){
                           ra.add(ps);
                           if (state.testState(DTState.TRACE)) {
                               state.traceInfo("addto", "arrayId", ra.getID() + "",
                                       ps.postFix());
                           }
                       }
                    }
                 
                }
            }
        }
    }
    
    /**
     * ( String -- ) throws a Rules Exception with the given message.
     * 
     * @author paul snow
     *
     */
    public static class ThrowException extends ROperator {
        ThrowException(){super("throwexception");}

        public void execute(DTState state) throws RulesException {
            IRObject value = state.datapop();
            throw new RulesException(
                        "throwException",
                        "ThrowExceiption.execute()",
                        value.stringValue());
        }
    }
}
