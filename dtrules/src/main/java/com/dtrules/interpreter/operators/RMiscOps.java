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

import java.util.Date;

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTime;
import com.dtrules.session.DTState;
import com.dtrules.session.RSession;

public class RMiscOps {
    static {
        new RError();       new Debug();        new Traceon();
        new Traceoff();     new Ignore();       new Swap();         
        new Dup();			new Get();
        new Pop();          new Over();         new EntityName();
        new Entitypush();   new Entitypop();    new Entityfetch();
        new I();            new J();            new K();
        new ToR();          new FromR();        new Def();
        new Find();         new Print();        new Clone();
        new Xdef();         new PStack();       new Null();
        new Createentity(); new Cvi();          new Cvr();
        new Cvb();          new Cve();          new Cvs();
        new Cvn();          new Cvd();          new ActionString();
        new GetDescription();                   
    }

    /**
     * (  -- RNull ) Return the Null object
     */
    static class Null extends ROperator {
        Null() {
            super("null");
        }
        public void execute(DTState state) throws RulesException {
            state.datapush(RNull.getRNull());
        }
    }
 
    
    /**
     * ( entity -- RName ) Get the name of the given entity.
     */
    static class EntityName extends ROperator {
        EntityName() {
            super("entityname");
        }
        public void execute(DTState state) throws RulesException {
            IREntity entity = state.datapop().rEntityValue();
            state.datapush(entity.getName());
        }
    }
    /**
     * ( Exception message -- ) Throws a RulesException with the given message. 
     * @author paul snow
     * 
     *
     */
    static class RError extends ROperator {
        RError(){super("error");}

        public void execute(DTState state) throws RulesException {
            String message   = state.datapop().stringValue();
            String exception = state.datapop().stringValue();
            try {
                throw new RulesException(
                        "exception", 
                        "User Exception",
                        message);
            } catch (Exception e) {
                throw new RulesException("Type Check", 
                        "User Exception",
                        exception+":"+message);
            }
        }
    }

    /**
     * ( string -- ) Prints a debug message only if debug output is enabled.
     * @author paul snow
     *
     */
    static class Debug extends ROperator {
        Debug(){super("debug");}

        public void execute(DTState state) throws RulesException {
            String msg = state.datapop().stringValue();
            if(state.testState(DTState.DEBUG)){
                state.debug(msg);
            }
        }
    }

    /**
     * ( -- ) Turn on the trace flag.
     * @author paul snow
     *
     */
    static class Traceon extends ROperator {
        Traceon(){super("traceon");}

        public void execute(DTState state) throws RulesException {
            state.setState(DTState.TRACE);
        }
    }

    /**
     * ( -- ) Turn off the trace flag
     * @author paul snow
     *
     */
    static class Traceoff extends ROperator {
        Traceoff(){super("traceoff");}

        public void execute(DTState state) throws RulesException {
            state.clearState(DTState.TRACE);
        }
    }
    
    /**
     * ( flag -- ) Set the debug state
     * 
     * @author paul snow
     *
     */
    static class SetDebug extends ROperator {
        SetDebug(){super("setdebug");}

        public void execute(DTState state) throws RulesException {
            boolean flg = state.datapop().booleanValue();
            if(flg){
                state.setState(DTState.DEBUG);
            }else{
                state.clearState(DTState.DEBUG);
            }    
        }
    }

    /**
     * A Noop -- Does nothing.
     * @author paul snow
     *
     */
    static class Ignore extends ROperator {
        Ignore(){super("ignore"); alias("nop");}

        public void execute(DTState state) throws RulesException {
        }
    }


    /**
     * ( obj1 obj2 -- obj2 obj1 ) swaps the top two elements on the data stack
     * @author paul snow
     *
     */
    static class Swap extends ROperator {
        Swap(){super("swap"); alias("exch");}

        public void execute(DTState state) throws RulesException {
            IRObject obj1  = state.datapop();
            IRObject obj2  = state.datapop();
            state.datapush(obj1);
            state.datapush(obj2);
        }
    }

    /**
     * ( obj1 -- obj1 obj2 )
     * @author paul snow
     *
     */
    static class Dup extends ROperator {
        Dup(){super("dup");}

        public void execute(DTState state) throws RulesException {
            IRObject obj1  = state.datapop();
            state.datapush(obj1);
            state.datapush(obj1);
        }
    }

    static class Pop    extends ROperator {
        Pop(){
            super("pop");
            alias("drop");
        }

        public void execute(DTState state) throws RulesException {
            state.datapop();
        }
    }
    
    /**
     * (obj1 obj2 -- obj1 obj2 obj1 ) copies the element below the top.
     * @author paul snow
     *
     */
    static class Over    extends ROperator {
        Over(){super("over");}

        public void execute(DTState state) throws RulesException {
            IRObject obj1 = state.getds(state.ddepth()-2);
            state.datapush(obj1);
        }
    }

    /**
     * ( entity -- ) push the given entity onto the entity stack
     * @author paul snow
     *
     */
    static class Entitypush    extends ROperator {
        Entitypush(){super("entitypush");}

        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IREntity e;
            try{
               e = o.rEntityValue(); 
            }catch(RulesException ex){
               ex.addToMessage("entitypush could not convert a "+RSession.typeInt2Str(o.type())+" to an Entity"); 
               throw ex;
            }
            state.entitypush(e);            
            if(state.testState(DTState.TRACE)){
               state.traceInfo("entitypush", "value",e.stringValue(),"id",e.getID()+"",null);
            }
        }
    }

    /**
     * ( -- ) pops the top element from the entity stack and tosses
     * it into the bit bucket
     * @author paul snow
     *
     */
    static class Entitypop    extends ROperator {
        Entitypop(){super("entitypop");}

        public void execute(DTState state) throws RulesException {
            if(state.testState(DTState.TRACE)){
                state.traceInfo("entitypop","");
             }
            state.entitypop();
        }
    }

    /**
     * Gets the given value from the given entity and pushes it on the entity stack.
     * ( entity attribute -- value ) 
     * @author Paul Snow
     *
     */
    static class Get    extends ROperator {
        Get(){super("get");}
        public void execute(DTState state) throws RulesException {
        	RName    n = state.datapop().rNameValue();
            IREntity e = state.datapop().rEntityValue();
            IRObject v = e.get(n);
            state.datapush(v);
        }
    }

    /**
     * Pushes a copy of the top entity on the entity stack on to the
     * data stack.
     * ( index -- element ) 
     * @author paul snow
     *
     */
    static class Entityfetch    extends ROperator {
        Entityfetch(){super("entityfetch");}
        public void execute(DTState state) throws RulesException {
            int i = state.datapop().intValue();
            state.datapush(state.entityfetch(i));
        }
    }
    
    /**
     * Returns the top element from the control stack.
     * @author paul snow
     *
     */
    static class  I   extends ROperator {
        I(){super("i"); alias("r@");}

        public void execute(DTState state) throws RulesException {
            state.datapush( state.getcs(state.cdepth()-1));
        }
    }

    /**
     * Returns the second element from the control stack.
     * @author paul snow
     *
     */
    static class  J   extends ROperator {
        J(){super("j");}

        public void execute(DTState state) throws RulesException {
            state.datapush( state.getcs(state.cdepth()-2));
        }
    }
    
    /**
     * Returns the third element from the control stack.
     * @author paul snow
     *
     */
    static class  K   extends ROperator {
        K(){super("k");}

        public void execute(DTState state) throws RulesException {
            state.datapush( state.getcs(state.cdepth()-3));
        }
    }
    
    /**
     * ( obj -- ) Pops the top element from the data stack, and pushes
     * it to the Control stack.
     * @author paul snow
     *
     */
    static class ToR    extends ROperator {
        ToR(){super(">r");}

        public void execute(DTState state) throws RulesException {
            state.cpush(state.datapop());
        }
    }
    
    /**
     * ( -- obj ) pops the top element from the control stack, and pushes
     * it to the data stack.
     * @author paul snow
     *
     */
    static class FromR    extends ROperator {
        FromR(){super("r>");}

        public void execute(DTState state) throws RulesException {
            state.datapush(state.cpop());
        }
    }
    
    /**
     * ( name value -- )
     * Binds the name with the value in the highest entity on the
     * entity stack which is both writable, and has an entry with
     * a writiable name that matches.
     * 
     * @author paul snow
     *
     */
    static class  Def extends ROperator {
        Def(){super("def");}

        public void execute(DTState state) throws RulesException {
            IRObject value = state.datapop();
            RName    name  = state.datapop().rNameValue();
            boolean f = state.def(name, value, true);
            if(!f)throw new RulesException("Undefined",
                    "def", 
                    name+" is undefined");
        }
    }
    /**
     * ( name -- obj )
     * Looks up the name, and returns the value associated with
     * the name in the top most entity that defines the name.
     * Returns RNull if the name isn't found.
     * @author paul snow
     *
     */
    static class  Find   extends ROperator {
        Find(){super("find");}

        public void execute(DTState state) throws RulesException {
            RName    name = state.datapop().rNameValue();
            IRObject v    = state.find(name);
            if(v==null)throw new RulesException("Undefined",
                    "find", 
                    name+" is undefined");
        }
    }
    /**
     * ( obj -- ) Prints the top element of on the data stack to 
     * Standard Out.
     * @author paul snow
     *
     */    
    static class  Print   extends ROperator {
        Print(){super("print"); }

        public void execute(DTState state) throws RulesException {
            state.debug(state.datapop().toString());
        }
    }
    
    /**
     * ( obj1 -- obj2 ) Creates a clone of the given object.
     * @author paul snow
     *
     */
    static class Clone   extends ROperator {
        Clone(){super("clone");}

        public void execute(DTState state) throws RulesException {
            state.datapush(state.datapop().clone(state.getSession()));
        }
    }
    
    /**
     * ( value name -- )
     * Binds the name with the value in the highest entity on the
     * entity stack which is both writable, and has an entry with
     * a writiable name that matches.
     * 
     * @author paul snow
     *
     */
    static class  Xdef extends ROperator {
        Xdef(){super("xdef");}

        public void execute(DTState state) throws RulesException {
            RName    name  = state.datapop().rNameValue();
            IRObject value = state.datapop();
            boolean f = state.def(name, value, true);
            if(!f)
                if(state.find(name)==null){
                    throw new RulesException("Undefined",
                            "xdef", 
                            name+" is undefined");
                }else{
                    throw new RulesException("Write Protection",
                            "xdef",
                            name+" is Input only, and xdef attempted to write to it");
                }
        }
    }

    /**
     * ( -- ) Prints all the elements on all the stacks non-distructively.
     * This is purely a debugging aid.
     * @author paul snow
     *
     */
    static class  PStack   extends ROperator {
        PStack(){super("pstack");}

        public void execute(DTState state) throws RulesException {
            state.pstack();
        }
    }
    
    /**
     * ( RName -- ) Creates an instance of the entity with the given name.
     * 
     * @author paul snow
     *
     */
    static class  Createentity   extends ROperator {
        Createentity(){super("createentity");}
        
        public void execute(DTState state) throws RulesException {
            RName    ename  = state.datapop().rNameValue();
            IREntity entity = ((RSession) state.getSession()).createEntity(null, ename);
            state.datapush(entity);
        }
    }
    
    /**
     * ( Object -- Integer ) Converts to an Integer.  Returns a null if no valid integer
     * value exists for the object.
     * 
     * @author paul snow
     *
     */
    static class  Cvi   extends ROperator {
        Cvi(){super("cvi");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rIntegerValue();
            }catch(Exception e){}
            state.datapush(v); 
        }
    }
    /**
     * ( Object -- Double ) Converts to an Double.  Returns a null if no valid double
     * value exists for the object
     * 
     * @author paul snow
     *
     */
    static class  Cvr   extends ROperator {
        Cvr(){super("cvr"); }
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rDoubleValue();
            }catch(Exception e){}
            state.datapush(v); 
        }
    }
    /**
     * ( Object -- Boolean ) Converts to an Boolean.  Pushes a null if the
     * object cannot be converted to a boolean value.
     * 
     * @author paul snow
     *
     */
    static class  Cvb   extends ROperator {
        Cvb(){super("cvb");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rBooleanValue();
            }catch(Exception e){}
            state.datapush(v);    
        }
    }
    /**
     * ( Object -- Entity ) Converts to an Entity.
     * 
     * @author paul snow
     *
     */
    static class  Cve   extends ROperator {
        Cve(){super("cve");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rEntityValue();
            }catch(Exception e){}
            state.datapush(v); 
        }
    }
    /**
     * ( Object -- String ) Converts to a String. If the object is a null, nothing is done.
     * 
     * @author paul snow
     *
     */
    static class  Cvs   extends ROperator {
        Cvs(){super("cvs");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            if(o.type() == iNull){
                state.datapush(o);
            }else{
                state.datapush(RString.newRString(o.stringValue()));
            }
        }
    }
    /**
     * ( Object -- String ) Converts to a Name.  Returns a null if no valid
     * name object exists for the value.
     * 
     * @author paul snow
     *
     */
    static class  Cvn   extends ROperator {
        Cvn(){super("cvn");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rNameValue();
            }catch(Exception e){}
            state.datapush(v); 
        }
    }
    /**
     * ( Object -- Date ) Converts to an Date.  Returns null if no valid Date
     * representation exists for the Object.
     * 
     * @author paul snow
     *
     */
    static class  Cvd   extends ROperator {
        Cvd(){super("cvd");}
        
        public void execute(DTState state) throws RulesException {
            IRObject o = state.datapop();
            IRObject v = RNull.getRNull();
            try{
                v = o.rTimeValue();
            }catch(Exception e){
                Date d =    state.getSession().getDateParser().getDate(o.stringValue());
                if(d!=null){
                    v = RTime.getRTime(d);
                }
            }
            state.datapush(v);    
        }
    }

    /**
     * ( -- String ) Returns the Decision Table and Action number
     * 
     * @author paul snow
     *
     */
    static class  ActionString   extends ROperator {
        ActionString(){super("actionstring");}
        
        public void execute(DTState state) throws RulesException {
            state.datapush( RString.newRString(
                    state.getCurrentTable().getName().stringValue()+" "+
                    state.getCurrentTableSection()+" "+
                    (state.getNumberInSection()+1)));
        }
    }
    /**
     * ( -- String ) Returns the Decision Table and Action number
     * 
     * @author paul snow
     *
     */
    static class  GetDescription   extends ROperator {
        GetDescription(){super("getdescription");}
        
        public void execute(DTState state) throws RulesException {
            String section = state.getCurrentTableSection();
            String description = "";
            if(section.equalsIgnoreCase("action")){
                RDecisionTable table = state.getCurrentTable();
                description = table.getActionsComment()[state.getNumberInSection()];
            }else if(section.equalsIgnoreCase("condition")){
                RDecisionTable table = state.getCurrentTable();
                description = table.getConditionsComment()[state.getNumberInSection()];
            }
            state.datapush( RString.newRString(description));
        }
    }
   
}
