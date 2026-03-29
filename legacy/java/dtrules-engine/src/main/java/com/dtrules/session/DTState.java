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

package com.dtrules.session;

import java.io.OutputStream;
import java.io.PrintStream;
import java.util.Calendar;
import java.util.GregorianCalendar;
import java.util.HashMap;
import java.util.Iterator;
import java.util.Random;

import com.dtrules.decisiontables.ANode;
import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.xmlparser.XMLPrinter;

public class DTState {

    public Calendar        calendar;
    { // Work around for a Java Bug
      // http://bugs.sun.com/bugdatabase/view_bug.do?bug_id=5045774
      //
        try {
            calendar = new GregorianCalendar();
        } catch (Throwable t) {
        }
    }

    private RDecisionTable currentTable;
    private String         currentTableSection;
    private int            numberInSection;
    private ANode          anode;                                    // The
                                                                      // current
                                                                      // Anode
                                                                      // under
                                                                      // execution.

    /**
     * The interpreter is a stack based interpreter. The implementation is much
     * like a Postscript Interpreter.
     * 
     * The control stack is used to implement a stack frame for decision tables.
     * The control stack has no analog in a PostScript interpreter.
     * 
     * The Entity stack is used to define the context for associating attributes
     * with values. This is much like the PostScript Dictionary Stack.
     * 
     * The Data stack is used to pass data to operators and to return results
     * from operators.
     */

    private final int      stklimit     = 1000;

    IRObject               ctrlstk[]    = new IRObject[stklimit];
    IRObject               datastk[]    = new IRObject[stklimit];
    IREntity               entitystk[]  = new IREntity[stklimit];

    int                    ctrlstkptr   = 0;
    int                    datastkptr   = 0;
    int                    entitystkptr = 0;

    int                    frames[]     = new int[stklimit];
    int                    framestkptr  = 0;
    int                    currentframe = 0;

    final IRSession        session;

    public long            seed         = 0x711083186866559L;
    public Random          rand         = new Random(seed);
    
    /**
     * The default debugging printstream is Standard Out. The default error
     * printstream is Standard Out.
     */
    private XMLPrinter     out          = new XMLPrinter(System.out);
    private PrintStream    outPs        = System.out;
    private PrintStream    err          = System.out;

    public static final int DEBUG   = 0x00000001;
    public static final int TRACE   = 0x00000002;
    public static final int ECHO    = 0x00000004;
    public static final int VERBOSE = 0x00000008;

    private int             state   = 0;          // This is the Rules Engine
                                                  // State under which the engine
                                                  // is running. Values are
                                                  // defined by DEBUG, TRACE,
                                                  // etc.
    
    private HashMap<RName, Object> extendedState = new HashMap<RName,Object>();
    
    public PrintStream getErrorOut()            { return err;   }
    public PrintStream getDebugOut()            { return outPs; }
    public PrintStream getTraceOut()            { return outPs; }
    public XMLPrinter  getTraceXMLPrinter()     { return out; }

    /**
     * The Extended State returns an object that might be used to maintain state information
     * not provided for by the Graphics State, but is needed by extended or application
     * specific operators.
     * <br><br>
     * You should use your fully qualified class name as the key, for example: 
     *    <br> <br>
     *    public static RName  extendedState = RName.getRName("com.dtrules.operators.polyoperators");
     *    <br> <br>
     *    setExtendedState(extendedState,new PolyState());
     *    
     * @param key
     * @return
     * @author paul snow
     **/   
    public Object getExtendedState(RName key){
        return extendedState.get(key);
    }
    
    /**
     * The Extended State returns an object that might be used to maintain state information
     * not provided for by the Graphics State, but is needed by extended or application
     * specific operators.
     * <br><br>
     * You should use your fully qualified class name as the key (though this isn't enforced), for example: 
     *    <br> <br>
     *    public static RName  extendedState = RName.getRName("com.dtrules.operators.polyoperators");
     *    <br> <br>
     *    setExtendedState(extendedState,new PolyState());
     *    
     * @param key
     * @return
     * @author paul snow
     **/
    public void setExtendedState(RName key, Object object){
        extendedState.put(key,object);
    }
    
    /**
     * Set the output streams for debug and trace.
     */
    public void setOutput(OutputStream debugtrace, OutputStream error) {
        if (debugtrace != null) {
            outPs = new PrintStream(debugtrace);
            out = new XMLPrinter(outPs);
        }
        err = new PrintStream(error);
    }

    /**
     * Set the output streams for debug and trace.
     */
    public void setOutput(PrintStream debugtrace, PrintStream error) {
        outPs = debugtrace;
        out = new XMLPrinter(outPs);
        err = error;
    }

    /**
     * We always print the error stream. But this may not be true forever.
     * 
     * @param s
     * @return
     */
    public boolean error(String s) {
        err.print(s);
        return true;
    }
    
    /**
     * Start the Trace
     */
    public void traceStart() {
        setState(TRACE);
        out.opentag("DTRulesTrace");
    }

    /**
     * Stop the trace
     * 
     * @throws RulesException
     */
    public void traceEnd() throws RulesException {
        out.close();
        clearState(TRACE);
    }

    /**
     * Internal use. Begins a tagged trace section.
     * 
     * @param tag
     * @param attribs
     */
    public void traceTagBegin(String tag, HashMap<String, Object> attribs) {
        if (testState(TRACE)) {
            out.opentag(tag, attribs);
        }
    }

    /**
     * Internal use. Begins a tagged trace section.
     * 
     * @param tag
     */
    public void traceTagBegin(String tag) {
        if (testState(TRACE)) {
            out.opentag(tag);
        }
    }

    /**
     * Internal use. Begins a tagged trace section.
     * 
     * @param tag
     * @param name1
     * @param value1
     */
    public void traceTagBegin(String tag, String name1, String value1) {
        if (testState(TRACE)) {
            out.opentag(tag, name1, value1);
        }
    }

    /**
     * internal use. Begins a tagged trace section.
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     */
    public void traceTagBegin(String tag, String name1, String value1, String name2, String value2) {
        if (testState(TRACE)) {
            out.opentag(tag, name1, value1, name2, value2);
        }
    }

    /**
     * Internal use. Begins tagged section
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     */
    public void traceTagBegin(String tag, String name1, String value1, String name2, String value2, String name3,
            String value3) {
        if (testState(TRACE)) {
            out.opentag(tag, name1, value1, name2, value2, name3, value3);
        }
    }

    /**
     * Internal use. Prints some information into the trace file.
     * 
     * @param tag
     * @param body
     */
    public void traceInfo(String tag, String body) {
        if (testState(TRACE))
            out.printdata(tag, body);
    }

    /**
     * Internal use. Prints some information into the trace file.
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param body
     */
    public void traceInfo(String tag, String name1, String value1, String body) {
        if (testState(TRACE))
            out.printdata(tag, name1, value1, body);
    }

    /**
     * Internal use. Prints some information into the trace file.
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param body
     */
    public void traceInfo(String tag, String name1, String value1, String name2, String value2, String body) {
        if (testState(TRACE))
            out.printdata(tag, name1, value1, name2, value2, body);
    }

    /**
     * Internal use. Prints some information into the trace file.
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param body
     */
    public void traceInfo(String tag, String name1, String value1, String name2, String value2, String name3,
            String value3, String body) {
        if (testState(TRACE))
            out.printdata(tag, name1, value1, name2, value2, name3, value3, body);
    }

    /**
     * Internal use. Prints some information into the trace file.
     * 
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param name4
     * @param value4
     * @param body
     */
    public void traceInfo(String tag, String name1, String value1, String name2, String value2, String name3,
            String value3, String name4, String value4, String body) {
        if (testState(TRACE))
            out.printdata(tag, name1, value1, name2, value2, name3, value3, name4, value4, body);
    }

    /**
     * End the trace
     */
    public void traceTagEnd() {
        if (testState(TRACE)) {
            out.closetag();
        }
    }

    /**
     * Prints a string to the output file if DEBUG is on. If ECHO is set, then
     * the output is also echoed to Standard out. Returns true if it printed
     * something.
     */
    public boolean debug(String s) {
        if (testState(DEBUG)) {
            if (testState(ECHO) && outPs != System.out) {
                System.out.print(s);
            }
            out.printdata("dbg", s);
            return true;
        }
        return false;
    }

    /**
     * Prints a string to System.out. 
     */
    public void print(String s) {
        System.out.print(s);
    }
    
    /**
     * Prints the Data Stack, Entity Stack, and Control Stack to the debugging
     * output stream.
     */
    public void pstack() {
        if (testState(TRACE))
            return;
        try {
            out.opentag("pstack");

            out.opentag("datastk");
            for (int i = 0; i < ddepth(); i++) {
                out.printdata("ds", "depth", "" + i, getds(i).stringValue());
            }
            out.closetag();

            out.opentag("entitystk");
            for (int i = 0; i < ddepth(); i++) {
                out.printdata("es", "depth", "" + i, getes(i).stringValue());
            }
            out.closetag();

            out.closetag();
        } catch (RulesException e) {
            err.print("ERROR printing the stacks!\n");
            err.print(e.toString() + "\n");
        }
    }

    /**
     * Returns the count of the number of elements on the data stack.
     * 
     * @return data stack depth
     */
    public int ddepth() {
        return datastkptr;
    }

    /**
     * Returns the element on the data stack at the given depth If there are 3
     * elements on the data stack, getds(2) will return the top element. A stack
     * underflow or overflow will be thrown if the index is out of range.
     */
    public IRObject getds(int i) throws RulesException {
        if (i >= datastkptr) {
            throw new RulesException("Data Stack Overflow", "getds", "index out of range: " + i);
        }
        if (i < 0) {
            throw new RulesException("Data Stack Underflow", "getds", "index out of range: " + i);
        }
        return datastk[i];
    }

    /**
     * Returns the element on the entity stack at the given depth If there are 3
     * entities on the entity stack, getes(2) will return the top entity. A
     * stack underflow or overflow will be thrown if the index is out of range.
     */
    public IREntity getes(int i) throws RulesException {
        if (i >= entitystkptr) {
            throw new RulesException("Entity Stack Overflow", "getes", "index out of range: " + i);
        }
        if (i < 0) {
            throw new RulesException("Entity Stack Underflow", "getes", "index out of range: " + i);
        }
        return entitystk[i];
    }

    /**
     * While the state holds the stacks, the Session holds changes to Entities
     * and other Rules Engine objects. On rare occasions we need to get our
     * session, so we save it in the DTState.
     * 
     * @param rs
     */
    DTState(IRSession rs) {
        session = rs;
    }

    /**
     * Get Session
     * 
     * @return the session assocaited with this state
     */
    public IRSession getSession() {
        return session;
    }

    /**
     * Returns the index of the Entity Stack.
     * 
     * @return
     */
    public int edepth() {
        return entitystkptr;
    }

    /**
     * Pushes an IRObject onto the data stack.
     * 
     * @param o
     * @throws RulesException
     */
    public void datapush(IRObject o) throws RulesException {
        if (datastkptr >= 1000) {
            throw new RulesException("Data Stack Overflow", o.stringValue(), "Data Stack overflow.");
        }
        datastk[datastkptr++] = o;
        if (testState(VERBOSE)) {
            traceInfo("datapush", "attribs", o.postFix(), null);
        }
    }

    /**
     * Pops an IRObject off the data stack and returns that object.
     * 
     * @return
     * @throws RulesException
     */
    public IRObject datapop() throws RulesException {
        if (datastkptr <= 0) {
            throw new RulesException("Data Stack Underflow", "datapop()", "Data Stack underflow.");
        }
        IRObject rval = datastk[--datastkptr];
        datastk[datastkptr] = null;
        if (testState(VERBOSE)) {
            traceInfo("datapop", rval.stringValue());
        }
        return (rval);
    }

    /**
     * Pushes an entity on the entity stack.
     * 
     * @param o
     * @throws RulesException
     */
    public void entitypush(IREntity o) throws RulesException {
        if (entitystkptr >= 1000) {
            throw new RulesException("Entity Stack Overflow", o.stringValue(), "Entity Stack overflow.");
        }
        if((state & TRACE) > 0){
        	traceInfo("entitypush","entity",o.getName().stringValue(), "id",o.getID()+"",null);
        }
        entitystk[entitystkptr++] = o;
    }

    /**
     * Pops an Entity off of the Entity stack.
     * 
     * @return
     * @throws RulesException
     */
    public IREntity entitypop() throws RulesException {
        if (entitystkptr <= 0) {
            throw new RulesException("Entity Stack Underflow", "entitypop", "Entity Stack underflow.");
        }
        if((state & TRACE) > 0){
        	traceInfo("entitypop",null);
        }
        IREntity rval = entitystk[--entitystkptr];
        entitystk[entitystkptr] = null;
        return (rval);
    }

    /**
     * Returns the nth element from the top of the entity stack (0 returns the
     * top element, 1 returns the 1 from the top, 2 the 2 from the top, etc.)
     * 
     * @return
     * @throws RulesException
     */
    public IREntity entityfetch(int i) throws RulesException {
        if (entitystkptr <= i) {
            throw new RulesException("Entity Stack Underflow", "entityfetch", "Entity Stack underflow.");
        }
        IREntity rval = entitystk[entitystkptr - 1 - i];
        return (rval);
    }

    /**
     * Test to see if the given flag is set.
     * 
     * @param flag
     * @return
     */
    public boolean testState(int flag) {
        return (state & flag) != 0;
    }

    /**
     * Clear the given flag (By and'ing the not of the flag into the state)
     */
    public void clearState(int flag) {
        state &= flag ^ 0xFFFFFFFF;
    }

    /**
     * Set the given flag by or'ing the flag into the state
     * 
     * @param flag
     */
    public void setState(int flag) {
        state |= flag;
    }

    /**
     * Returns the current depth of the control stack.
     * 
     * @return
     */
    public int cdepth() {
        return ctrlstkptr;
    }

    /**
     * Returns the index to the currentframe.  
     * @return
     * @throws RulesException
     */
    public int getCurrentFrame() throws RulesException {
        return currentframe;
    }
    
    /**
     * Internal use. Pushes a frame onto the control stack from which local
     * variables can be allocated.
     * 
     * @throws RulesException
     */
    public void pushframe() throws RulesException {
        if (framestkptr >= stklimit) {
            throw new RulesException("Control Stack Overflow", "pushframe", "Control Stack Overflow.");

        }
        frames[framestkptr++] = currentframe;
        currentframe = ctrlstkptr;
    }

    /**
     * Internal use. Pops a frame from the control stack, reclaiming storage
     * used by local variables.
     * 
     * @throws RulesException
     */
    public void popframe() throws RulesException {
        if (framestkptr <= 0) {
            throw new RulesException("Control Stack Underflow", "popframe", "Control Stack underflow.");
        }
        ctrlstkptr = currentframe; // Pop off this frame,
        currentframe = frames[--framestkptr]; // Then set the currentframe back
                                              // to its previous value.
    }

    /**
     * Internal Use only. Get a value from a frame location.
     * 
     * @param i
     * @return
     * @throws RulesException
     */
    public IRObject getFrameValue(int i) throws RulesException {
        if (currentframe + i >= ctrlstkptr) {
            throw new RulesException("OutOfRange", "getFrameValue", "");
        }
        return getcs(currentframe + i);
    }

    /**
     * Internal Use. Set a value within the stack frame.
     * 
     * @param i
     * @param value
     * @throws RulesException
     */
    public void setFrameValue(int i, IRObject value) throws RulesException {
        if (currentframe + i >= ctrlstkptr) {
            throw new RulesException("OutOfRange", "getFrameValue", "");
        }
        setcs(currentframe + i, value);
    }

    /**
     * Internal use. Push an Object onto the control stack.
     * 
     * @param o
     */
    public void cpush(IRObject o) {
        ctrlstk[ctrlstkptr++] = o;
    }

    /**
     * Pop the top element from the control stack and return it.
     * 
     * @return
     */
    public IRObject cpop() {
        IRObject r = ctrlstk[--ctrlstkptr];
        ctrlstk[ctrlstkptr] = null;
        return r;
    }

    /**
     * Internal use. Pull a value off the control stack.
     * 
     * @param i
     * @return
     * @throws RulesException
     */
    public IRObject getcs(int i) throws RulesException {
        if (i >= ctrlstkptr) {
            throw new RulesException("Control Stack Overflow", "getcs", "index out of range: " + i);
        }
        if (i < 0) {
            throw new RulesException("Control Stack Underflow", "getcs", "index out of range: " + i);
        }
        return ctrlstk[i];
    }

    /**
     * Set a value at a location on the control stack.
     * 
     * @param i
     * @param v
     * @throws RulesException
     */
    public void setcs(int i, IRObject v) throws RulesException {
        if (i >= ctrlstkptr) {
            throw new RulesException("Control Stack Overflow", "setcs", "index out of range: " + i);
        }
        if (i < 0) {
            throw new RulesException("Control Stack Underflow", "getcs", "index out of range: " + i);
        }
        ctrlstk[i] = v;
    }

    /**
     * This method evalates a condition, or any other set of PostFix code that
     * produces a boolean value. The code must only add one element to the data
     * stack, and that element must have a valid boolean value.
     * 
     * @param c
     *            -- Condition to execute
     * @return -- Returns the boolean value of c
     * @throws RulesException
     */
    public boolean evaluateCondition(IRObject c) throws RulesException {
        int stackindex = datastkptr; // We make sure the object only produces
                                     // one boolean.
        c.execute(this); // Execute the condition.
        if (datastkptr - 1 != stackindex) {
            throw new RulesException("Stack Check Exception", "Evaluation of Condition", "Stack not balanced");
        }
        return datapop().booleanValue();
    }

    /**
     * This method executes an action, or any other set of Postfix code. This
     * code can have side effects, but it cannot change the depth of the data
     * stack.
     * 
     * @param c
     *            -- Object to execute
     * @throws RulesException
     */
    public void evaluate(IRObject c) throws RulesException {
        int stackindex = datastkptr;
        c.execute(this);
        if (datastkptr != stackindex) {
            throw new RulesException("Stack Check Exception", "Evaluation of Action", "Stack not balanced");
        }
    }

    /**
     * Looks up the entity stack for an instance of an entity with the given
     * entity name. If such an entity is on the entity stack (i.e. in the
     * current context) then this routine returns true, otherwise false. Note
     * that this routine doesn't care how many entities of that type are in the
     * context, but merely returns true if one of them is in the context.
     * 
     * @param entity
     * @return
     */
    public boolean inContext(String entity) {
        return inContext(RName.getRName(entity));
    }

    /**
     * Looks up the entity stack for an instance of an entity with the given
     * entity name. If such an entity is on the entity stack (i.e. in the
     * current context) then this routine returns true, otherwise false. Note
     * that this routine doesn't care how many entities of that type are in the
     * context, but merely returns true if one of them is in the context.
     * 
     * @param entity
     * @return
     */
    public boolean inContext(RName entity) {
        for (int i = 0; i < entitystkptr; i++) { // entity on the Entity Stack.
            IREntity e = entitystk[i];
            if (e.getName().equals(entity))
                return true;
        }
        return false;
    }

    /**
     * Looks up the entity stack for an Entity that defines an attribute that
     * matches the name provided. When such an Entity with an attribute that
     * matches the name is found, that Entity is returned. A null is returned if
     * no match is found, which means no Entity on the entity Stack defines an
     * attribute with a name that mathches the name provided.
     * 
     * @param name
     */
    public IREntity findEntity(RName name) throws RulesException {
        RName entityname = name.getEntityName(); // If the RName does not spec
                                                 // an Enttiy Name
        if (entityname == null) { // then we simply look for the RName in each
            for (int i = entitystkptr - 1; i >= 0; i--) { // entity on the
                                                          // Entity Stack.
                IREntity e = entitystk[i];
                if (e.containsAttribute(name))
                    return e;
            }
        } else { // Otherwise, we insist that the Entity name
            for (int i = entitystkptr - 1; i >= 0; i--) { // match as well as
                                                          // insist that the
                                                          // Entity
                IREntity e = entitystk[i]; // have an attribute that matches
                                           // this name.
                if (e.getName().equals(entityname)) {
                    if (e.containsAttribute(name)) {
                        return e;
                    }
                }
            }
        }

        return null; // No matach found? return a null.
    }

    /**
     * Looks up the Entity Stack and returns the value for the given named
     * attribute.
     * 
     * When getting data out of the rules Engine, it is useful to take string
     * values rather than RNames. This should never be done within the Rules
     * Engine where RNames should be the coin of the realm.
     * 
     * This routine simply returns a null if an error occurs or if the name is
     * undefined.
     */
    public IRObject find(String name) {
        try {
            return find(RName.getRName(name));
        } catch (RulesException e) {
            return null;
        }
    }

    /**
     * Looks up the entity stack and returns the entity which defines the value
     * of the given attribute.
     * 
     * When getting data out of the rules Engine, it is useful to take string
     * values rather than RNames. This should never be done within the Rules
     * Engine where RNames should be the coin of the realm.
     * 
     * This routine simply returns a null if an error occurs or if the name is
     * undefined.
     */
    public IREntity findEntity(String name) {
        try {
            return findEntity(RName.getRName(name));
        } catch (RulesException e) {
            return null;
        }
    }

    /**
     * Looks up the entity stack for a match for the RName. When a match is
     * found, the value is returned. A null is returned if no match is found.
     * 
     * @param name
     */
    public IRObject find(RName name) throws RulesException {
        IREntity entity = findEntity(name);
        if (entity == null)
            return null;
        return entity.get(name);
    }

    /**
     * Looks up the entity stack for a match for the RName. When a match is
     * found, the value is placed there and a true is returned. If no match is
     * found, def returns false.
     * 
     * @param name
     * @param value
     */
    public boolean def(RName name, IRObject value, boolean protect) throws RulesException {

        RName entityname = name.getEntityName();

        for (int i = entitystkptr - 1; i >= 0; i--) {
            IREntity e = entitystk[i];
            if (!e.isReadOnly() && (entityname == null || e.getName().equals(entityname))) {
                REntityEntry entry = e.getEntry(name);
                if (entry != null && (!protect || entry.writable)) {
                    if (testState(TRACE)) {
                        out.printdata("def", 
                        		"id",     e.getID(), 
                        		"entity", e.getName().stringValue(), 
                        		"name",   name.stringValue(), 
                        		value.postFix());
                    }

                    e.put(null, name, value);
                    return true;
                }
            }
        }
        return false;
    }

    /**
     * Get the current Decision Table under execution.
     * 
     * @return
     */
    public RDecisionTable getCurrentTable() {
        return currentTable;
    }

    /**
     * Internal Use. Set the current Decision Table under execution.
     * 
     * @param currentTable
     */
    public void setCurrentTable(RDecisionTable currentTable) {
        this.currentTable = currentTable;
    }

    /**
     * Get the current Decision Table section, i.e. Condition, Action, Context,
     * etc.
     * 
     * @return the currentTableSection
     */
    public String getCurrentTableSection() {
        return currentTableSection;
    }

    /**
     * Condition, Action, Context, etc.
     * 
     * @param currentTableSection
     *            the currentTableSection to set
     */
    public void setCurrentTableSection(String currentTableSection, int number) {
        this.currentTableSection = currentTableSection;
        this.numberInSection = number;
    }

    /**
     * Condition number, context number, initial Action number, etc. -1 means
     * not set
     * 
     * @return the numberInSection
     */
    public int getNumberInSection() {
        return numberInSection;
    }

    /**
     * Condition number, context number, initial Action number, etc. -1 means
     * not set
     * 
     * @param numberInSection
     *            the numberInSection to set
     */
    public void setNumberInSection(int numberInSection) {
        this.numberInSection = numberInSection;
    }

    /**
     * Return the current ANode under execution.
     * 
     * @return
     */
    public ANode getAnode() {
        return anode;
    }

    /**
     * Set the current ANode under execution.
     * 
     * @param anode
     */
    public void setAnode(ANode anode) {
        this.anode = anode;
    }

    @Override
    public String toString() {
        String s = "dstk["+datastkptr+"]: ";
        for(int i=0;i<datastkptr;i++){
            s += datastk[i].stringValue()+" ";
        }
        s += "\nentitycstk["+entitystkptr+"]: ";
        for(int i=0;i<entitystkptr;i++){
            s += entitystk[i].getName().stringValue()+"["+entitystk[i].getID()+"] ";
        }
        s += "\nctrlstk["+ctrlstkptr+"]: ";
        for(int i=0;i<ctrlstkptr;i++){
            s += ctrlstk[i].stringValue()+" ";
        }
        
        return s;
    }
    
}
