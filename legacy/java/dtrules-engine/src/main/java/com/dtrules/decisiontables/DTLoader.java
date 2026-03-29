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
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.IGenericXMLParser;
/**
 * The DTLoader loads decision tables into the EntityFactory.  Warning messages
 * are routed through a RSession object.
 * 
 * @author paul snow
 * Feb 12, 2007
 *
 */
public class DTLoader implements IGenericXMLParser {

	private transient String 	_tag; 
	private transient String 	_body; 
	private transient Map<String,String> _attribs;
	
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
	private int     ps_col      = 0;        // Current Policy Statement Column;  The count is the number of columns.
	
	private boolean processing_fields = false;  // This boolean is only set within the <attribute_fields> tag.
	                                            // Any tag within this tag is interpreted as an attribute of
	                                            // the decision table.  The Table_type tag is treated special.
	/**
	 *  We have no limits on the number of columns for a decision table.  But
	 *  from a practical point of view, 16 is a reasonable limit.
	 **/  
	
	// Temp Space for collecting data for the decision tables
	ArrayList<String> context_formal     = new ArrayList<String>();
    ArrayList<String> context_comments   = new ArrayList<String>();
	ArrayList<String> context_postfix    = new ArrayList<String>();
	ArrayList<String> ia_formal          = new ArrayList<String>();
	ArrayList<String> ia_postfix         = new ArrayList<String>();
	ArrayList<String> ia_comment         = new ArrayList<String>();
	ArrayList<String> c_formal           = new ArrayList<String>();
	ArrayList<String> c_comment          = new ArrayList<String>();
	ArrayList<String> c_postfix          = new ArrayList<String>();
	ArrayList<ArrayList<String>> c_table;
	ArrayList<String> a_formal           = new ArrayList<String>();
	ArrayList<String> a_comment          = new ArrayList<String>();
	ArrayList<String> a_postfix          = new ArrayList<String>();
	ArrayList<ArrayList<String>> a_table;
	ArrayList<String> ps_formal          = new ArrayList<String>();
	ArrayList<String> ps_postfix         = new ArrayList<String>();
	
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
		
        c_table = new ArrayList<ArrayList<String>>();
        a_table = new ArrayList<ArrayList<String>>();

	}
	
	/**
	 * All done gathering the info for the table.  Now we need to 
	 * pack it away into the actual decision table.
	 *
	 */
	public void end_decision_table(){
	    /** Contexts do not have comments **/
        dt.contexts                 = new String[context_cnt];
        dt.contextComments          = new String[context_cnt];
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
        
        /** PolicyStatements have a statement **/
        int realcnt = 0;
        for(int i=0; i<ps_formal.size(); i++){
            if(ps_formal.get(i)!=null && ps_formal.get(i).trim().length()>0){
                realcnt = i+1;
            }
        }
        dt.policystatements         = new String [realcnt];
        dt.policystatementsPostfix  = new String [realcnt];
            
        //Move over the information for the contexts
        adjust(context_formal,context_cnt);
        adjust(context_comments,context_cnt);
        adjust(context_postfix,context_cnt);
        for(int i=0;i<context_cnt;i++){
            dt.contexts[i]          = context_formal.get(i);
            dt.contextComments[i]   = context_comments.get(i);
            dt.contextsPostfix[i]   = context_postfix.get(i);
        }
        
        //Move over the information for actions
        for(int i=0;i<ia_cnt;i++){
            dt.initialActions[i]        =ia_formal.get(i);
            dt.initialActionsPostfix[i] =ia_postfix.get(i);
            dt.initialActionsComment[i] =ia_comment.get(i);
        }
        
        //Move over the information for the conditions
		for(int i=0;i<c_cnt;i++){
			dt.conditions[i]         = c_formal.get(i);
			dt.conditionsComment[i]  = c_comment.size()<=i || c_comment.get(i)==null?"":c_comment.get(i);
			dt.conditionsPostfix[i]  = c_postfix.size()<=i || c_postfix.get(i)==null?"":c_postfix.get(i);
			for(int j=0;j<col_cnt;j++){
                String v = " ";
                if(c_table.size()> i && c_table.get(i)!= null && c_table.get(i).size() > j){
                    v = c_table.get(i).get(j);
                    v = v==null ? " " : v;
                }
				dt.conditiontable[i][j] = v ;
			}
		}
		
		//Move over the information for the actions
		for(int i=0;i<a_cnt;i++){
			dt.actions[i]        = a_formal.get(i);
			dt.actionsComment[i] = a_comment.size()<= i || a_comment.get(i)==null?"":a_comment.get(i);
			dt.actionsPostfix[i] = a_postfix.size()<= i || a_postfix.get(i)==null?"":a_postfix.get(i);
			for(int j=0;j<col_cnt;j++){
                String v = a_table.get(i).get(j);
				dt.actiontable[i][j]=  v==null ? " " : v;
			}
		}
		
		adjust(ps_formal,col_cnt);
		adjust(ps_postfix,col_cnt);
		for(int i=0; i<realcnt;i++){
		    dt.policystatements[i] = ps_formal.get(i);
		    dt.policystatementsPostfix[i] = ps_postfix.get(i);
		}
        ps_formal.clear();
        ps_postfix.clear();
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
	
    public void end_context_description(){
        adjust(context_formal,context_cnt);
        context_formal.set(context_cnt, _body);
    }
    
	public void end_context_comment(){
	    adjust(context_comments,context_cnt);
	    context_comments.set(context_cnt, _body);
	}
	
	public void end_context_postfix (){
	    adjust(context_postfix,context_cnt);
	    context_postfix.set(context_cnt,_body);
	}
	
	public void end_context_details (){
	    context_cnt++;
	}
	
	public void end_initial_action_description(){
        adjust(ia_formal,ia_cnt);
	    ia_formal.set(ia_cnt,_body);
	}

    public void end_initial_action_comment(){
        adjust(ia_comment,ia_cnt);
        ia_comment.set(ia_cnt,_body);
    }

	public void end_initial_action_postfix () {
	    adjust(ia_postfix,ia_cnt);
	    ia_postfix.set(ia_cnt,_body);
	}
	
	public void end_initial_action_details () {
	    ia_cnt++;
	}
	
	
	public void end_condition_description(){
	    adjust(c_formal,c_cnt);
		c_formal.set(c_cnt,_body);
	}
	
	public void end_condition_postfix(){
	    adjust(c_postfix,c_cnt);
		c_postfix.set(c_cnt,_body);
	}
	
	public void end_condition_comment(){
	    adjust(c_comment, c_cnt);
	    
	    c_comment.set(c_cnt,_body);
	}
	
	public void begin_condition_column() throws RulesException {
		
		int theCol = Integer.parseInt((String)_attribs.get("column_number"));
		
		if(theCol > col_cnt){  // This check works because the column_number is one based. 
            col_cnt = theCol;
        }
		
		theCol--;              // Make our column index zero based.
		
		adjust(c_table,c_cnt);
		if(c_table.get(c_cnt)==null)c_table.set(c_cnt,new ArrayList<String>());
		adjust(c_table.get(c_cnt),theCol);
		
		c_table.get(c_cnt).set(theCol, (String) _attribs.get("column_value"));
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
	        adjust(a_table,a_cnt);
	        if(a_table.get(a_cnt)==null)a_table.set(a_cnt,new ArrayList<String>());
	        adjust(a_table.get(a_cnt),i);

	        a_table.get(a_cnt).set(i,"");
		}
	}
	
	public void end_action_description(){
	    adjust(a_formal,a_cnt);
		a_formal.set(a_cnt,_body);
	}
	
	public void end_action_postfix(){
        adjust(a_postfix,a_cnt);
		a_postfix.set(a_cnt,_body);
	}
	
	public void end_action_comment(){
        adjust(a_comment,a_cnt);
		a_comment.set(a_cnt,_body);
	}
	
	public void begin_action_column() {
		int    a_col = Integer.parseInt((String)_attribs.get("column_number"))-1;
		
		String a_v   = (String)_attribs.get("column_value");
        
		adjust(a_table,a_cnt);
        if(a_table.get(a_cnt)==null)a_table.set(a_cnt,new ArrayList<String>());
        adjust(a_table.get(a_cnt),a_col);
		
		a_table.get(a_cnt).set(a_col, a_v);
	}

	public void end_action_details() {
		a_cnt++;
	}
	
	public void begin_policy_statement(){
	    String num = _attribs.get("column");
	    ps_col = 0;
	    if(num != null && num.length()>0){
 	       try{ 
 	           ps_col = Integer.parseInt(num); 
 	       } catch(Exception e){
 	           System.err.println("Invalid column index: '"+num+"'");
 	       }
	    }
	    if(ps_col<0 || ps_col > col_cnt){
	        throw new RuntimeException("The column value for a Policy Statement '"+ps_col+"' is out of range");
	    }
	}
	
	public void end_policy_description(){
	    adjust(ps_formal,ps_col);
	    ps_formal.set(ps_col, _body);
	}

	public void end_policy_statement_postfix(){
        adjust(ps_postfix, ps_col);
        ps_postfix.set(ps_col, _body);
    }
	
	public void beginTag(
	        String[]               tagstk, 
	        int                    tagstkptr, 
	        String                 tag, 
	        HashMap<String,String> attribs  ) throws IOException, Exception {
		
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
		} catch (NoSuchMethodException e){      // Ignore undefined tags	
        } catch (InvocationTargetException e){  // Errors thrown by Decision Table parsing
            throw new RuntimeException("An Invocation Target Exception was thrown processing the Begin XML tag "+tag+
                            "\nError states: "+e.getCause());
		} catch (Exception e) {
            state.traceInfo("error", e.getCause().getMessage());
			throw new RuntimeException("Error Parsing Decision Tables at begin tag: "+tag+"\r\n "+e.getMessage());
		}			
	}
   
	HashMap<String,Method> methodCache = new HashMap<String,Method>();
	
	public void endTag(
	        String[]               tagstk, 
	        int                    tagstkptr, 
	        String                 tag, 
	        String                 body, 
	        HashMap<String,String> attribs    )throws Exception, IOException {
	
        _tag        = tag;
        _attribs    = attribs;
        body        = body.trim().replaceAll("[\\n\\r]"," ");        
        _body       = body;
		String tagname = "end_"+tag;
		
		try {
    		Class<Object>[]  classArr = null;
    		Object[]         objArr   = null;
    		Method           m        = methodCache.get(tagname);
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

	@SuppressWarnings({ "unchecked", "rawtypes" })
    private void adjust (ArrayList a, int i){
	    while(a.size()<= i)a.add(null);
	}
	
}
