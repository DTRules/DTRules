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

  
package com.dtrules.decisiontables;

import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IDecisionTableError;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;
import com.dtrules.xmlparser.IGenericXMLParser;
/**
 * The DTLoader loads decision tables into the EntityFactory.  Warning messages
 * are routed through a RSession object.
 * 
 * @author paul snow
 * Feb 12, 2007
 *
 */
@SuppressWarnings({"unchecked","unused"})
public class DTLoader implements IGenericXMLParser {

	private transient String 	_tag; 
	private transient String 	_body; 
	private transient Map       _attribs;
	
	private final EntityFactory ef;         // As tables are loaded, they need to be registered
                                            //   with the entity factory.
    private final IRSession     session;    // Even though we are not building anything within
                                            //   a session, we need a session to print warning messages.
    private final DTState       state;      // For error reporting.
    
    private int     context_cnt = 0;        // Count of contexts
    private int     ia_cnt      = 0;        // Count of initial actions.
	private int     c_cnt       = 0;        // Count of conditions
	private int     a_cnt       = 0;        // Count of actions
	private int     col_cnt     = 1;        // Count of columns (We assume at least one column)
	private int     col         = 0;        // Current column
	
	private boolean processing_fields = false;  // This boolean is only set within the <attribute_fields> tag.
	                                            // Any tag within this tag is interpreted as an attribute of
	                                            // the decision table.  The Table_type tag is treated special.
	/**
	 *  The limit here is many times too large for actual decision tables.
	 *  In fact, one should throw an error if a table has more than 16 columns,
	 *  or more than 16 actions.  Tables larger than that are too hard to 
	 *  understand to be useful.
	 **/  
	static final int LIMIT = 100;
	
	// Temp Space for collecting data for the decision tables
	String context_formal  []  = new String[LIMIT];
	String context_postfix []  = new String[LIMIT];
	String ia_formal []        = new String[LIMIT];
	String ia_postfix []       = new String[LIMIT];
	String ia_comment []       = new String[LIMIT];
	String c_formal[]          = new String[LIMIT];
	String c_comment[]         = new String[LIMIT];
	String c_postfix[]         = new String[LIMIT];
	String c_table[][]         = new String[LIMIT][LIMIT];
	String a_formal[]          = new String[LIMIT];
	String a_comment[]         = new String[LIMIT];
	String a_postfix[]         = new String[LIMIT];
	String a_table[][]         = new String[LIMIT][LIMIT];

    RDecisionTable dt = null;

    /**
     * In order to load decision tables into an EntityFactory, we create
     * a loader.  The tables are loaded into the entityfactory, and warning
     * messages will be routed through the session.
     * 
     * @param _session
     * @param _ef
     */
	public DTLoader(IRSession _session, EntityFactory _ef){
		session = _session;
        state   = _session.getState();
        ef      = _ef;
	}
	

	public void end_decision_tables()throws Exception {
		/**
		 * Now we build all the decision tables.
		 */
		Iterator<RName> it = ef.getDecisionTableRNameIterator();
		while(it.hasNext()){
            try {
                RDecisionTable dt = ef.getDecisionTable(it.next());
                dt.build(state);
                
			} catch (RulesException e) {
                state.traceInfo("error","Error building Decision Table "+dt+"\r\n"+e);
			}
		}
	}
	
	
	/**
	 * Handle the Decision Table setup and initialization.
	 */
	
	/**
	 * Decision Table Name
	 * @throws RulesException
	 */
	public void end_table_name() throws RulesException {
		dt = ef.newDecisionTable(_body, session);
		
		c_cnt         = 0;
		a_cnt         = 0;
		col_cnt       = 1;
		col           = 0;
		ia_cnt        = 0;
		context_cnt   = 0;
		
        c_table = new String[LIMIT][LIMIT];
        a_table = new String[LIMIT][LIMIT];

	}
	
	/**
	 * All done gathering the info for the table.  Now we need to 
	 * pack it away into the actual decision table.
	 *
	 */
	public void end_decision_table(){
	    /** Contexts do not have comments **/
        dt.contexts                 = new String[context_cnt];
        dt.contextsPostfix          = new String[context_cnt];
        
        /** Initial Actions have Comments, descriptions, and postfix **/
        dt.initialActionsComment    = new String[ia_cnt];
        dt.initialActions           = new String[ia_cnt];
        dt.initialActionsPostfix    = new String[ia_cnt];
 
        /** The count of columns for Conditions and actions have to match **/
        dt.maxcol                   = col_cnt;
        
        /** Conditions have a comment, description, Postfix, and a truth table **/
        dt.conditionsComment        = new String[c_cnt];
        dt.conditions               = new String[c_cnt];
		dt.conditionsPostfix        = new String[c_cnt];		
		dt.conditiontable           = new String[c_cnt][col_cnt];

		/** Actions have a comment, description, Postfix, and a action table **/
        dt.actionsComment           = new String[a_cnt];
        dt.actions                  = new String[a_cnt];
        dt.actionsPostfix           = new String[a_cnt];
        dt.actiontable              = new String[a_cnt][col_cnt];
        
        //Move over the information for the contexts
        for(int i=0;i<context_cnt;i++){
            dt.contexts[i]          = context_formal[i];
            dt.contextsPostfix[i]   = context_postfix[i];
        }
        
        //Move over the information for actions
        for(int i=0;i<ia_cnt;i++){
            dt.initialActions[i]        =ia_formal[i];
            dt.initialActionsPostfix[i] =ia_postfix[i];
            dt.initialActionsComment[i] =ia_comment[i];
        }
        
        //Move over the information for the conditions
		for(int i=0;i<c_cnt;i++){
			dt.conditions[i]         = c_formal[i];
			dt.conditionsComment[i]  = c_comment[i]==null?"":c_comment[i];
			dt.conditionsPostfix[i]  = c_postfix[i]==null?"":c_postfix[i];
			for(int j=0;j<col_cnt;j++){
                String v = c_table[i][j];
				dt.conditiontable[i][j]=   v==null ? " " : v ;
			}
		}
		
		//Move over the information for the actions
		for(int i=0;i<a_cnt;i++){
			dt.actions[i]        = a_formal[i];
			dt.actionsComment[i] = a_comment[i]==null?"":a_comment[i];
			dt.actionsPostfix[i] = a_postfix[i]==null?"":a_postfix[i];
			for(int j=0;j<col_cnt;j++){
                String v = a_table[i][j];
				dt.actiontable[i][j]=  v==null ? " " : v;
			}
		}
	}
	
	/**
	 * Set the filename on the Decision Table
	 */
	public void end_xls_file(){
	    dt.setFilename(_body);
	}
	
    /** Turn on field processing when the <attribute_field> tag is encountered
     * 
     */    
    public void begin_attribute_fields() {
        processing_fields = true;
    }
    /**
     * Turn off attribute field processing outside the </attribute_filed> tag
     */
    public void end_attribute_fields() {
        processing_fields = false;
    }
    
    /**
     * Mostly we just collect attribute fields.  But if the attribute field is the
     * table_type tag, then it has to have a value of FIRST, ALL, or BALANCED.
     */
	public void process_field() throws RulesException {
	    dt.fields.put(RName.getRName(_tag),_body);
		if(_tag.equalsIgnoreCase("type")){
		    if(_body.equalsIgnoreCase("first")){
                dt.setType(RDecisionTable.Type.FIRST);
            }else if (_body.equalsIgnoreCase("all")){
                dt.setType(RDecisionTable.Type.ALL);
            }else if (_body.equalsIgnoreCase("balanced")){
                dt.setType(RDecisionTable.Type.BALANCED);
            }else {
                throw new RulesException("Invalid","Decision Table Load","Bad Decision Table type Encountered: '"+_body+"'");
            }
		}    
    }
    	
	
	/*
	 * Handle each condition in turn. 
	 *
	 */
	
	public void begin_condition_details(){
		col=0;  // Start at the begining of the columns again.
	}
	
	public void end_context_description(){
	    context_formal[context_cnt] = _body;
	}
	
	public void end_context_postfix (){
	    context_postfix[context_cnt] = _body;
	}
	
	public void end_context_details (){
	    context_cnt++;
	}
	
	public void end_initial_action_description(){
	    ia_formal[ia_cnt] = _body;
	}

    public void end_initial_action_comment(){
        ia_comment[ia_cnt] = _body;
    }

	public void end_initial_action_postfix () {
	    ia_postfix[ia_cnt] = _body;
	}
	
	public void end_initial_action_details () {
	    ia_cnt++;
	}
	
	
	public void end_condition_description(){
		c_formal[c_cnt]=_body;
	}
	
	public void end_condition_postfix(){
		c_postfix[c_cnt]=_body;
	}
	
	public void end_condition_comment(){
		c_comment[c_cnt]=_body;
	}
	
	public void begin_condition_column() throws RulesException {
		if(col>=col_cnt){
		    col_cnt++;
		}
		int theCol = Integer.parseInt((String)_attribs.get("column_number"))-1;
		c_table[c_cnt][theCol]= (String) _attribs.get("column_value");
		col++;
	}
	
	public void end_condition_details(){
		c_cnt++;
	}
	
	/*
	 * Load Actions
	 */
	
	
	public void begin_action_details(){
		for(int i=0;i<col_cnt;i++){
			a_table[a_cnt][i]="";
		}
	}
	
	public void end_action_description(){
		a_formal[a_cnt]=_body;
	}
	
	public void end_action_postfix(){
		a_postfix[a_cnt]=_body;
	}
	
	public void end_action_comment(){
		a_comment[a_cnt]=_body;
	}
	
	public void begin_action_column() {
		int    a_col = Integer.parseInt((String)_attribs.get("column_number"))-1;
		if(a_col>=col_cnt){
			throw new RuntimeException("Decision Table Loader: Too many columns found in "+dt.toString());
		}
		String a_v   = (String)_attribs.get("column_value");
		
		a_table[a_cnt][a_col]= a_v;
	}

	public void end_action_details() {
		a_cnt++;
	}
	
	public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap attribs) throws IOException, Exception {
		
		_tag       = tag;
		_attribs   = attribs;
		_body      = null;
		String tagname = "begin_"+tag;
		try {
            Method m = methodCache.get(tagname);
            if(m==null){
                m = this.getClass().getMethod(tagname,(Class [])null);
                methodCache.put(tagname,m);
            }    
            m.invoke(this,(Object [])null);  			
		} catch (NoSuchMethodException e){   // Ignore undefined tags	
        } catch (InvocationTargetException e){  // Ignore errors thrown by Decision Table parsing
            throw new RuntimeException("An Invocation Target Exception was thrown processing the Begin XML tag "+tag+
                            "\nError states: "+e.getCause());
		} catch (Exception e) {
            state.traceInfo("error", e.getCause().getMessage());
			throw new RuntimeException("Error Parsing Decision Tables at begin tag: "+tag+"\r\n "+e.getMessage());
		}			
	}
   
	HashMap<String,Method> methodCache = new HashMap<String,Method>();
	
	public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap attribs) throws Exception, IOException {
	
		_tag       = tag;
		_attribs   = attribs;
        body = body.trim().replaceAll("[\\n\\r]"," ");        
		_body      = body;
		String tagname = "end_"+tag;
		try {
    			Class[] classArr = null;
    			Object[] objArr = null;
    			Method m = methodCache.get(tagname);
    			if(m==null){
    			    m = this.getClass().getMethod(tagname,classArr);
    			    methodCache.put(tagname, m);
    			}    
    			m.invoke(this, objArr);    
		} catch (NoSuchMethodException e){ 		// Ignore undefined tags
		    if(processing_fields){              // Unless we are in the attribute files section.
                process_field();
            }
		} catch (InvocationTargetException e){  // Ignore errors thrown by Decision Table parsing
            state.traceInfo("error", "An Invocation Target Exception was thrown processing the End XML tag "+tag+
                            "\nError states "+e.getCause());
		} catch (Exception e) {
            state.traceInfo("error", e.getCause().getMessage());
			throw new RuntimeException("Error Parsing Decision Tables at end tag: "+tag+" body: "+body+"\r\n"+e.getMessage());
		}	
		
	}

	/**
	 * Skip DTD stuff and other parsing things the Rules Engine doesn't 
	 * care about.
	 */
	public boolean error(String v) throws Exception {
		return true;
	}
	
	
}
