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

import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

import com.dtrules.decisiontables.DTNode.Coordinate;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.ARObject;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RType;
import com.dtrules.interpreter.operators.ROperator;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IDecisionTableError;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.xmlparser.GenericXMLParser;

/**
 * Decision Tables are the classes that hold the Rules for a set of Policy 
 * implemented using DTRules.  There are three types: <br><br>
 * 
 * BALANCED -- These decision tables expect all branches to be defined in the condition table <br>
 * ALL      -- Evaluates all the columns, then executes all the actions, in the order they
 *             are specified, for all columns whose conditions are met.<br>
 * FIRST    -- Effectively evaluates each column, and executes only the first Column whose
 *             conditions are met.<br>
 * @author paul snow
 * Mar 1, 2007
 *
 */
public class RDecisionTable extends ARObject {
    
	public static RType dttype = RType.newType("decisiontable");
	
    public static final String DASH = "-";      // Using a constant reduces our footprint, and increases our speed.
    
    private final  RName    dtname;             // The decision table's name.
    
    private        String   filename = null;    // Filename of Excel file where the table is defined,
                                                //   if this decision table is defined in Excel.

    enum UnbalancedType { FIRST, ALL };         // Unbalanced Table Types.
    
    public static enum Type { 
        BALANCED { void build(DTState state, RDecisionTable dt) {dt.compile(); dt.buildBalanced();                       dt.check(null);}},
        FIRST    { void build(DTState state, RDecisionTable dt) {dt.compile(); dt.buildUnbalanced(state, UnbalancedType.FIRST); dt.check(null);}},
        ALL      { void build(DTState state, RDecisionTable dt) {dt.compile(); dt.buildUnbalanced(state, UnbalancedType.ALL);   dt.check(null);}};
        abstract void build(DTState state, RDecisionTable dt);
    }    
  
    public Type  type = Type.BALANCED;          // By default, all decision tables are balanced.
    
    public static  final int MAXCOL = 16;       // The Maximum number of columns in a decision table.
    
                         int maxcol = 1;        // The number of columns in this decision table.
                         
                         int maxbalcol = 0;     // Number of balanced columns in this decision table. This is a computed thing, not used at 
                                                //    runtime, and only set if the balanced condition table or action table is asked for.
                         
    private final IRSession session;        	// We need to compile within a session, so we know how to parse dates (among other things)
	
    private final RuleSet ruleset;			    // A decision table must belong to a particular ruleset
    
    public  final Map<RName,String> fields = new HashMap<RName, String>(); // Holds meta information about this decision table.

	private boolean  compiled=false;            // The decision table isn't compiled
												//   until fully constructed.  And
												//   it won't be if compilation fails.

    String   [] contexts;                       // Contexts in which to execute this table.
    String   [] contextComments;                // Comments on the contexts.
    String   [] contextsPostfix;                // The Postfix for each context statement.
    String      contextsrc;                     // For Tracing...
    IRObject    rcontext;                       //  lists of entities.  It is best if this is done within the table than
                                                //  by the calling table.	

    String   [] initialActions;                 // A list of actions to be executed each time the 
                                                //   decision table is executed before the conditions
                                                //   are evaluated.
    String   [] initialActionsPostfix;          // Compiled Initial Actions
    String   [] initialActionsComment;          // Comment for Initial Actions
    IRObject [] rinitialActions;                // The compiled version of the initial action

	
	String [][] conditiontable;
	String [][] conditiontablebalanced;         // A balanced rendering of the Decision Table; We don't use this 
	                                            //   at runtime; it is mostly a UI thing.
	String   [] conditions;                     // The conditions in formal.  This is compiled to get the postfix
	String   [] conditionsPostfix;              // The conditions in postfix. derived from the formal
	String   [] conditionsComment;              // A comment per condition.
	IRObject [] rconditions;					// Each compiled condition
	
    String [][] actiontable;                    // Indicates which actions should be executed
    String [][] actiontablebalanced;            // A balanced rendering of the Decision Table; We don't use this
                                                //   at runtime; it is mostly a UI thing.
    String   [] actions;                        // The actions in the language specified
	String   [] actionsComment;                 // A free form comment for the action
	String   [] actionsPostfix;                 // The compiled postfix version of the action
	IRObject [] ractions;						// Each compiled action
	
	// While actions and conditions are 0 based arrays, the policy statements are 1 based.
	// The zero'th index is the default value (if no columns match).
	String   [] policystatements;               // The Policy statements as defined in the Decision Table.
	String   [] policyStatementsBalanced;       // A balanced rendering of the Policy Statements; We don't use this
	                                            //   at runtime; it is mostly a UI thing.
	String   [] policystatementsPostfix;        // Generally, a policy statement will be a string, but can have values
	IRObject [] rpolicystatements;              // The IRObject that returns a string for a policy statement.
	
	
	
	boolean  [] columnsSpecified  = null;       // Columns defined in the XML
	boolean  [] columnsUsed       = null;       // These are the columns really used in the executable form of the table.
	boolean  [] conditionsUsed    = null;       // We mark conditions which might be evaluated
	boolean  [] actionsUsed       = null;       // We mark actions which might be evaluated
	boolean  [] columnUnreachable = null;       // Mark actions unreachable if they don't end up in the decision tree
	boolean     hasNullColumn     = false;      // Not all tables have a null column.  This is true if 
	                                            //   this table has one.
	
	int starColumn      = -1;                   // Column where a star is found (defaulted to none).	
	int otherwiseColumn = -1;
	int alwaysColumn    = -1;
	
	boolean optimize = true;                    // Don't optimize all tables with a refernce to policestatements
	
	// Virtual field values.
    public static  RName    table_name = RName.getRName("Name");
    public static  RName    file_name = RName.getRName("File_Name");
    public static  RName    type_name = RName.getRName("type");	
	
	
	List<IDecisionTableError> errorlist = new ArrayList<IDecisionTableError>();
	DTNode decisiontree=null;

    private int numberOfRealColumns = 0;        // Number of real columns (as unbalanced tables can have
    											// far more columns than they appear to have).

	public boolean getHasNullColumn(){
	    return hasNullColumn;
	}
	
	/**
     * @return the type
     */
    public Type getType() {
        return type;
    }

    /**
     * @return the mAXCOL
     */
    public static int getMAXCOL() {
        return MAXCOL;
    }

    /**
     * @return the maxcol
     */
    public int getMaxcol() {
        return maxcol;
    }

    /**
     * @return the ruleset
     */
    public RuleSet getRuleset() {
        return ruleset;
    }

    /**
     * @return the fields
     */
    public Map<RName, String> getFields() {
        return fields;
    }

    /**
     * @return the initialActions
     */
    public String[] getInitialActions() {
        return initialActions;
    }

    /**
     * @return the rinitialActions
     */
    public IRObject[] getRinitialActions() {
        return rinitialActions;
    }

    /**
     * @return the initialActionsPostfix
     */
    public String[] getInitialActionsPostfix() {
        return initialActionsPostfix;
    }

    /**
     * @return the initialActionsComment
     */
    public String[] getInitialActionsComment() {
        return initialActionsComment;
    }
    
    /**
     * 
     * @return comments on each context statement.
     */
    public String[] getContextsComment() {
        return contextComments;
    }

    /**
     * @return the contexts
     */
    public String[] getContexts() {
        return contexts;
    }

    /**
     * @return the contextsPostfix
     */
    public String[] getContextsPostfix() {
        return contextsPostfix;
    }

    /**
     * @return the contextsrc
     */
    public String getContextsrc() {
        return contextsrc;
    }

    /**
     * @return the rcontext
     */
    public IRObject getRcontext() {
        return rcontext;
    }

    /**
     * @return the columnsSpecified
     */
    public boolean[] getColumnsSpecified() {
        return columnsSpecified;
    }

    /**
     * @return the columnsUsed
     */
    public boolean[] getColumnsUsed() {
        return columnsUsed;
    }

    /**
     * @return the conditionsUsed
     */
    public boolean[] getConditionsUsed() {
        return conditionsUsed;
    }

    /**
     * @return the actionsUsed
     */
    public boolean[] getActionsUsed() {
        return actionsUsed;
    }

    /**
     * @return the columnUnreachable
     */
    public boolean[] getColumnUnreachable() {
        return columnUnreachable;
    }

    /**
     * @return the errorlist
     */
    public List<IDecisionTableError> getErrorlist() {
        return errorlist;
    }

    public String [][] getActionTableBalanced(IRSession session ){
        if(actiontablebalanced == null){
            try {
                RDecisionTable dt        = getBalancedTable(session);
                actiontablebalanced      = dt.actiontable;
                conditiontablebalanced   = dt.conditiontable;
                policyStatementsBalanced = dt.policystatements;
            } catch (RulesException e) {   }
        }
        return actiontablebalanced;
    }

    public String [][] getConditionTableBalanced(IRSession session ) {
        if(conditiontablebalanced == null){
            try {
                RDecisionTable dt        = getBalancedTable(session);
                actiontablebalanced      = dt.actiontable;
                conditiontablebalanced   = dt.conditiontable;
                policyStatementsBalanced = dt.policystatements;
            } catch (RulesException e) {   }
        }
        return conditiontablebalanced;
    }

    private void whatsUsed(){
        int conditionCnt  = conditiontable.length>0?conditiontable[0].length:0;
        columnsSpecified  = new boolean [conditionCnt];
	    columnsUsed       = new boolean [conditionCnt];
	    columnUnreachable = new boolean [conditionCnt];
	    conditionsUsed    = new boolean [conditions.length];
	    actionsUsed       = new boolean [actions.length];
	    
	    for(int col=0; col < conditionCnt; col++){   // For each column
	        for(int row=0; row < conditiontable.length; row++) { // For each row
	            if(    conditiontable[row][col].equalsIgnoreCase("y")||
	                   conditiontable[row][col].equalsIgnoreCase("n")||
	                   conditiontable[row][col].equalsIgnoreCase("*")){
	                columnsSpecified[col]    = true;
	            }
	        }
	                // A bit tricky.  We LOOK at columns that seem to be used.
                    //  but if no actions are specified, we don't worry about them.
                    //  This is because dead columns will be dropped when we 
                    //  balance tables, and we will throw a bogus error if we don't
                    //  ignore such columns here.
	 
	        for(int row=0; row < actions.length; row++){
	            if( columnsSpecified[col] && actiontable[row][col].equalsIgnoreCase("x")){
	                columnsUsed[col]=true;                  // If there is an action specified, return it to the used
	                actionsUsed[row]=true;                  //   category (and mark the action as used as well).
	            }
	        }
	    }
	}	
	
	private void setUnreachable(){
	    for(int i=0; i < columnsUsed.length; i++){         // We assume the worst... If we are using the column,
	        columnUnreachable[i]=columnsUsed[i];           //  then we assume it is unreachable until proven 
	    }                                                  //  otherwise.
	    setUnreachable(decisiontree);                      // Now go look for each column in the decisiontree!
	}
	
	private void setUnreachable(DTNode node){              // This is a recursive search for each column... 
	    if(node == null ) return;                          // Somebody else's problem if nulls are in the tree...
	    if(node instanceof CNode){                         // If a condition node, recurse...
            setUnreachable( ((CNode)node).iftrue);         //    look at the true branches... 
            setUnreachable( ((CNode)node).iffalse);        //    look at the false branches...
            conditionsUsed[((CNode)node).conditionNumber] = true;   // And set that we might actually need to evaluate
	    }                                                  //             this condition.
	    if(node instanceof ANode){                         // If this is an Action Node... Look at its columns!    
	        for (int col : ((ANode)node).columns){         // Simply grab the columns that might lead to this action...
	            if(col <= columnUnreachable.length){       //    and mark them as reachable (i.e. not unreachable)
	                columnUnreachable[col-1]=false;        //    (but ignore all of this if the col number is out if range)
	            }
	        }
	        if(((ANode)node).columns.size()==0) {          // If no column got us here, then this is a null column
	            hasNullColumn = true;
	        }
	        for (int action : ((ANode)node).anumbers){     // Mark all the actions in this column as used.
	            actionsUsed[action]=true;
	        }
	    }
	    
	}
	
    public int getNumberOfRealColumns() {
        if(decisiontree==null)return 0;
        return decisiontree.countColumns();
    }

    /**
     * Check for errors in the decision table.  Returns the column
     * and row of a problem if one is found.  If nothing is wrong,
     * a null is returned.
     * @return
     */
    public Coordinate validate(){
       if(decisiontree==null){
           if(actions !=null && actions.length==0)return null;
            return new Coordinate(0,0);
       }
       return decisiontree.validate();
    }
    
    BalanceTable balanceTable = null;           // Helper class to build a balanced or optimized version
                                                //   of this decision table.
    
    public boolean isCompiled(){return compiled;}
    
    
    
	public String getFilename() {
        return filename;
    }

    public void setFilename(String filename) {
        this.filename = filename;
        fields.put(file_name,filename);
    }

    @Override
    public IRObject clone(IRSession s) throws RulesException {
        RDecisionTable dt = new RDecisionTable(s, dtname.stringValue());
        
        dt.numberOfRealColumns      = numberOfRealColumns;
        
        dt.contexts                 = contexts.clone();
        dt.contextsPostfix          = contextsPostfix.clone();
        dt.contextsrc               = contextsrc;
        dt.rcontext                 = rcontext != null? rcontext.clone(s):null;
        
        dt.rinitialActions          = rinitialActions.clone();
        dt.initialActions           = initialActions.clone();
        dt.initialActionsComment    = initialActionsComment.clone();
        dt.initialActionsPostfix    = initialActionsPostfix.clone();
        
        dt.conditiontable           = conditiontable.clone();
        dt.conditions               = conditions.clone();
        dt.conditionsPostfix        = conditionsPostfix.clone();
        dt.conditionsComment        = conditionsComment.clone();
        dt.rconditions              = rconditions.clone();
        
        dt.actiontable              = actiontable.clone();
        dt.actions                  = actions.clone();
        dt.actionsComment           = actionsComment.clone();
        dt.actionsPostfix           = actionsPostfix.clone();
        dt.ractions                 = ractions.clone();
        
        dt.policystatements         = policystatements.clone();
        dt.policystatementsPostfix  = policystatementsPostfix.clone();
        dt.rpolicystatements        = rpolicystatements.clone();
        
        return dt;
    }
    /**
     * Changes the type of the given decision table.  The table is rebuilt. 
     * @param type
     * @return Returns a list of errors which occurred when the type was changed.
	 */
	public void setType(Type type) {
       this.type = type;   
    }
	/**
	 * This routine compiles the Context statements for the 
	 * decision table into a single executable array.  
	 * It must embed into this array a call to executeTable 
	 * (which avoids this context building for the table).
	 */
	private void buildContexts(){
	    // Nothing to do if no extra contexts are specfied.
	   if(contextsPostfix==null || contextsPostfix.length==0) return;
       
	   // This is the call to executeTable against this decisiontable
	   // that we are going to embed into our executable array.
	   contextsrc = "/"+getName().stringValue()+" executeTable ";
       
	   boolean keep = false;
       for(int i=contextsPostfix.length-1;i>=0;i--){
           if(contextsPostfix[i]!=null){
               contextsrc = "{ "+contextsrc+" } "+contextsPostfix[i];
               keep = true;
           }    
       }
       if(keep == true){
           try {
              rcontext = RString.compile(session, contextsrc, true);
           } catch (RulesException e) {
              errorlist.add(
                    new CompilerError (
                            IDecisionTableError.Type.CONTEXT,
                            "Formal Compiler Error: "+e,
                            contextsrc,0));
           }
       }  	    
	}
		
    /**
     * Build this decision table according to its type.
     *
     */
	public void build(DTState state){
       errorlist.clear();
       decisiontree = null;
       buildContexts();
       /** 
        * If a context or contexts are specified for this decision table,
        * compile the context formal into postfix.
        */
       type.build(state, this);
    }
        
    /**
     * Return the name of this decision table.
     * @return
     */
    public RName getName(){
        return dtname;
    }
    
    /**
     * Renames this decision table.
     * @param session
     * @param newname
     * @throws RulesException
     */
    public void rename(IRSession session, RName newname)throws RulesException{
        session.getEntityFactory().deleteDecisionTable(dtname);
        session.getEntityFactory().newDecisionTable(newname, session);
    }
    
    /**
     * Create a Decision Table 
     * @param tables
     * @param name
     * @throws RulesException
     */
    
	public RDecisionTable(IRSession session, String name) throws RulesException{
        this.session = session;
        ruleset      = session.getRuleSet();
		dtname       = RName.getRName(name,true);
		
        EntityFactory ef = session.getEntityFactory();
        RDecisionTable dttable =ef.findDecisionTable(RName.getRName(name));
        if(dttable != null){
            new CompilerError(CompilerError.Type.TABLE,"Duplicate Decision Tables Found",0,0);
		}    
	}
	
	ROperator ps = ROperator.getInstance("policystatements");
	
	/**
	 * If the given array calls the policystatements operator, then we cannot
	 * optimize ALL tables.
	 */
	private boolean callsPolicyStatement(IRObject c, ArrayList<IRObject> list) {
	    if(list == null) list = new ArrayList<IRObject>();
	    if(list.contains(c)) return false;
	    try{
	        if(ps.equals(c)) {
	            return true;
	        }
	   
    	    if(c.type().getId() == iArray){
    	        list.add(c);
    	        RArray ar = c.rArrayValue();
    	        for(IRObject v : ar){
    	            boolean f = callsPolicyStatement(v,list);
    	            if(f) return f;
    	        }
    	    }
    	    
	    }catch(RulesException e) {} return false;
	   
	}
	
	/**
	 * Compile each condition and action.  We mark the decision table as
	 * uncompiled if any error is detected.  However, we still attempt to 
	 * compile all conditions and all actions.
	 */
	public List<IDecisionTableError> compile(){
	    try{
    		compiled          = true;                  // Assume the compile will work.		
    		rconditions       = new IRObject[conditionsPostfix.length];
    		ractions          = new IRObject[actionsPostfix.length];
    		rinitialActions   = new IRObject[initialActionsPostfix.length];
    		rpolicystatements = new IRObject[policystatementsPostfix.length];
    		
    		// The assumption is that if you compile the table, something changed.  We would
    		// need to recompute these in that case.
    		actiontablebalanced    = null;
    		conditiontablebalanced = null;
    		
    		for(int i=0; i< initialActions.length; i++){
                 try {
                     rinitialActions[i] = RString.compile(session, initialActionsPostfix[i],true);
                 } catch (Exception e) {
                     errorlist.add(
                             new CompilerError(
                                IDecisionTableError.Type.INITIALACTION,
                                "Postfix Interpretation Error: "+e,
                                initialActionsPostfix[i],
                                i
                             )
                     );            
                     compiled = false;
                     rinitialActions[i]=RNull.getRNull();
                 }
             }
    		 
    		for(int i=0;i<rconditions.length;i++){
    			try {
    				rconditions[i]= RString.compile(session, conditionsPostfix[i],true);
    			} catch (RulesException e) {
                    errorlist.add(
                       new CompilerError(
                          IDecisionTableError.Type.CONDITION,
                          "Postfix Interpretation Error: "+e,
                          conditionsPostfix[i],
                          i
                       )
                    );
                    compiled=false;
    				rconditions[i]=RNull.getRNull();
    			}
    		}
    		for(int i=0;i<ractions.length;i++){
    			try {
    				ractions[i]= RString.compile(session, actionsPostfix[i],true);
                    if(ractions[i] != null && type == Type.ALL && callsPolicyStatement(ractions[i],null)){
                        optimize = false;
                    }
                    
    			} catch (RulesException e) {
                    errorlist.add(
                            new CompilerError(
                               IDecisionTableError.Type.ACTION,
                               "Postfix Interpretation Error: "+e,
                               actionsPostfix[i],
                               i
                            )
                         );
                    compiled=false;
    				ractions[i]=RNull.getRNull();
    			}
    		}
    		
    		for(int i=0;i<policystatementsPostfix.length;i++){
                try {
                    rpolicystatements[i]= RString.compile(session, policystatementsPostfix[i],true);
                } catch (RulesException e) {
                    errorlist.add(
                            new CompilerError(
                               IDecisionTableError.Type.POLICYSTATEMENT,
                               "Postfix Interpretation Error: "+e,
                               policystatementsPostfix[i],
                               i
                            )
                         );
                    compiled=false;
                    rpolicystatements[i]=RNull.getRNull();
                }
            }
    		
	    }catch(Exception e){
            errorlist.add(
                    new CompilerError(
                       IDecisionTableError.Type.TABLE,
                       "Unexpected Exception Thrown: "+e,
                       0,
                       0
                    )
                 );
      
	  
	    }
		return errorlist;
	}
	
	/**
	 * Checks the compile of this decision table, setting the columns used and
	 * looks for unreachable columns.
	 */
	public void check(PrintStream out){
		
	    whatsUsed();
		setUnreachable();
		
        boolean header = false;

		for(int i=0; i< columnUnreachable.length; i++){
		    if(columnUnreachable[i]){
		        if(out!= null && header == false){
                    out.println (getName().stringValue());
                    header = true;
                }
                if(out!=null)out.println("  *** Column "+(i+1)+" cannot be reached.");
                
		        //errorlist.add(                              // Print warnings about unreachable code.
                //        new CompilerError(
                //                ICompilerError.Type.TABLE,
                //                "Column "+(i+1)+" cannot be reached.",
                //                0,i
                //             )
                //        );
		    }
		}
				
		for(int i=0; i< conditionsUsed.length; i++){
		    if(rconditions[i]!=null && conditionsUsed[i]==false){
		        if(out!= null && header == false){
		            out.println (getName().stringValue());
		            header = true;
		        }
		        if(out!=null)out.println("      condition "+(i+1)+" is not used");
		    }
		}
		
		for(int i=0; i< actionsUsed.length; i++){
            if(ractions[i]!=null && actionsUsed[i]==false){
                if(out != null && header == false){
                    out.println (getName().stringValue());
                    header = true;
                }
                if(out!=null)out.println("      action "+(i+1)+" is not used");
            }
        }
	}
	
	public void execute(DTState state) throws RulesException {
		arrayExecute(state);
	}
	
	public void arrayExecute(DTState state) throws RulesException {
	    RDecisionTable last = state.getCurrentTable();
	    state.setCurrentTable(this);
	    state.traceTagBegin("decisiontable","name",dtname.stringValue());
	    try {
			int estk     = state.edepth();
			int dstk     = state.ddepth();
			int cstk     = state.cdepth();

			state.pushframe();
			
			if(rcontext==null){
			    if(state.testState(DTState.TRACE)){
			        try{
    			        state.traceTagBegin("execute_table");
    			           executeTable(state);
    			        state.traceTagEnd();
			        }catch(RulesException e){
			            state.traceTagEnd();
			            throw e;
			        }
			    }else{
			        executeTable(state);
			    }
			    
			}else{
			    if(state.testState(DTState.TRACE)){
			        state.traceTagBegin("context", "execute",contextsrc);
			        if(state.testState(DTState.VERBOSE)){
				        for(String context : this.contexts){
				            if(context != null && context.trim().length()>0){
				                state.traceInfo("formal",context);
				            }
				        }
			        }
			        state.traceTagBegin("execute_table");
			        try {
                        rcontext.execute(state);
                    } catch (RulesException e) {
                        state.traceTagEnd();
                        state.traceTagEnd();
                        e.setSection("Context", 0);
                        throw e;
                    }
                    state.traceTagEnd();
			        state.traceTagEnd();
			    }else{
			        rcontext.execute(state);
			    }    
			}
			state.popframe();
			
			if(estk!= state.edepth() ||
			   dstk!= state.ddepth() ||
			   cstk!= state.cdepth() ){
			    throw new RulesException("Stacks Not balanced","DecisionTables", 
			    "Error while executing table: "+getName().stringValue() +"\n" +
			     (estk!= state.edepth() ? "Entity Stack size before: "+estk+" after: "+state.edepth()+"\n":"")+
			     (dstk!= state.ddepth() ? "Data Stack size before: "+dstk+" after: "+state.ddepth()+"\n":"")+
			     (cstk!= state.cdepth() ? "Control Stack size before: "+cstk+" after: "+state.cdepth()+"\n":""));
			}
		} catch (RulesException e) {
	        try{ 
	        	state.traceTagEnd(); 
	        }catch (RuntimeException e2){}
			e.addDecisionTable(this.getName().stringValue(), this.getFilename());
			state.setCurrentTable(last);
			throw e;
		}
		state.traceTagEnd();
	    state.setCurrentTable(last);
	}
	
	/**
	 * A decision table is executed by simply executing the
	 * binary tree underneath the table.
	 */
	public void executeTable(DTState state) throws RulesException {
        boolean trace = state.testState(DTState.TRACE);
		
		if(compiled==false){
            throw new RulesException(
                "UncompiledDecisionTable",
                "RDecisionTable.execute",
                "Attempt to execute an uncompiled decision table: "+dtname.stringValue()
            );
        }
        
        int edepth    = state.edepth();  // Get the initial depth of the entity stack 
                                         //  so we can toss any extra entities added...
        if(trace){
            if(state.testState(DTState.VERBOSE)){
                state.traceTagBegin("entity_stack");
                for(int i=0;i<state.edepth();i++){
                    state.traceInfo("entity", "id",state.getes(i).getID()+"", state.getes(i).stringValue());
                }
                state.traceTagEnd();
            }
            state.traceTagBegin("initialActions");
            for( int i=0; rinitialActions!=null && i<rinitialActions.length; i++){
                try{
                   state.traceTagBegin("initialAction");
                   state.traceInfo("formal",initialActions[i]);
                   int dstk = state.ddepth();
                   rinitialActions[i].execute(state);
                   if(dstk != state.ddepth()){
                       throw new RulesException("datastackunbalanced", "initialActions", "Initial Action: "+(i+1)+" failed!");
                   }
                   state.traceTagEnd();
                }catch(RulesException e){
                    e.setSection("Initial Actions", i+1);
                    throw e;
                }
            }
            state.traceTagEnd();
            if(decisiontree!=null)decisiontree.execute(state);
	        state.traceTagEnd();
	        state.traceTagBegin("execute_table");
        }else{
            for( int i=0; rinitialActions!=null && i<rinitialActions.length; i++){
                state.setCurrentTableSection("InitialActions", i);
                try{
                    rinitialActions[i].execute(state);
                 }catch(RulesException e){
                     e.setSection("Initial Actions", i+1);
                     throw e;
                 }
            }
            if(decisiontree!=null)decisiontree.execute(state);
        }    
        while(state.edepth() > edepth)state.entitypop();     // Pop off extra entities. 
	}

	/**
	 * Builds (if necessary) the internal representation of the decision table,
	 * then validates that structure.
	 * @return true if the structure builds and is valid; false otherwise.
	 */
	public List<IDecisionTableError> getErrorList(DTState state)  {
       if(decisiontree==null){
           errorlist.clear();
           build(state);
	   }
	   return errorlist;
	}	
	
    
    
    
	/**
	 * Builds the decision tree, which is a binary tree of "DTNode"'s which can be executed
     * directly.  This defines the execution of a Decision Table.
     * <br><br>
     * The way we build this binary tree is we walk down each column, tracing
	 * that column's path through the decision tree.  Once we are at the end of the column,
	 * we add on the actions.  This algorithm assumes that a decision table describes
	 * a complete decision tree, i.e. there is no set of possible condition states which 
     * are not explicitly handled by the decision table.
	 *
	 */
	void buildBalanced() {
		if(conditiontable[0].length == 0 ||           // If we have no conditions, or
		   conditiontable[0][0].equals("*")){         // If *, we just execute all actions
		   decisiontree = ANode.newANode(this,0);	  //   checked in the first column             
		   return;
		}
       
		decisiontree = new CNode(this,0,0, rconditions[0]);     // Allocate a root node.
       
        for(int col=0;col<maxcol;col++){                        // For each column, we are going to run down the
                                                                //   column building that path through the tree.
			boolean laststep = conditiontable[0][col].equalsIgnoreCase("y");	// Set the test for the root condition.
			CNode   last     = (CNode) decisiontree;            // The last node will start as the root.					

			boolean star = false;                               // Once you find a star, you can't do something else.
			for(int i=1; i<conditiontable.length; i++){         // Now go down the rest of the conditions.
				String t = conditiontable[i][col];              // Get this conditions truth table entry.
				
				
				boolean yes  = t.equalsIgnoreCase("y");
				boolean no   = t.equalsIgnoreCase("n");
				if(star){
				    new CompilerError(IDecisionTableError.Type.TABLE,"You can't follow a '*' with a '"+t+"' ",i,col);
				}
				star = t.equalsIgnoreCase("*");
				
				boolean invalid = false;
				
				if(yes || no ){                                // If this condition should be considered...
					CNode here=null;
					try {
						if(laststep){
							here = (CNode) last.iftrue;
						}else{
							here = (CNode) last.iffalse;
						}
						
						if(here == null){                       // Missing a CNode?  Create it!
							here = new CNode(this,col,i,rconditions[i]);
							if(laststep){
								last.iftrue  = here;
							}else{
								last.iffalse = here;
							}
						}
						
					} catch (RuntimeException e) {
                        invalid = true;        
					}
					if(invalid || here.conditionNumber != i ){
                        errorlist.add(
                                new CompilerError(
                                   IDecisionTableError.Type.TABLE,
                                   "Condition Table Compile Error ",
                                   i,col
                                )
                        );
                        return;
					}
					last     = here;
					laststep = yes;
				}
            }    
			if(laststep){										// Once we have traced the column, add the actions.
				last.iftrue=ANode.newANode(this,col);	
			}else{
				last.iffalse=ANode.newANode(this,col);
			}
            
		}
        DTNode.Coordinate rowCol = decisiontree.validate();
        if(rowCol!=null){
            errorlist.add(
               new CompilerError(IDecisionTableError.Type.TABLE,"Condition Table isn't balanced.",rowCol.row,rowCol.col)
            );        
            compiled = false;
        }
	}
	
    boolean newline=true;
    
    private void printattrib(PrintStream p, String tag, String body){
        if(!newline){p.println();}
        p.print("<"); p.print(tag); p.print(">");
        p.print(body);
        p.print("</"); p.print(tag); p.print(">");
        newline = false;
    }
    
    private void openTag(PrintStream p,String tag){
        if(!newline){p.println();}
        p.print("<"); p.print(tag); p.print(">");
        newline=false;
    }
    
    /**
     * Write the XML representation of this decision table to the given outputstream.
     * @param o Output stream where the XML for this decision table will be written.
     */
    public void writeXML(PrintStream p){
        p.println("<decision_table>");
        newline = true;
        printattrib(p,"table_name",dtname.stringValue());
        Iterator<RName> ifields = fields.keySet().iterator();
        while(ifields.hasNext()){
            RName name = ifields.next();
            printattrib(p,name.stringValue(),fields.get(name));
        }
        openTag(p, "conditions");
        for(int i=0; i< conditions.length; i++){
            openTag(p, "condition_details");
            printattrib(p,"condition_number",(i+1)+"");
            printattrib(p,"condition_description",GenericXMLParser.encode(conditions[i]));
            printattrib(p,"condition_postfix",GenericXMLParser.encode(conditionsPostfix[i]));
            printattrib(p,"condition_comment",GenericXMLParser.encode(conditionsComment[i]));
            p.println();
            newline=true;
            for(int j=0; j<maxcol; j++){
               p.println("<condition_column column_number=\""+(j+1)+"\" column_value=\""+conditiontable[i][j]+"\" />");
            }
            p.println("</condition_details>");
        }
        p.println("</conditions>");
        openTag(p, "actions");
        for(int i=0; i< actions.length; i++){
            openTag(p, "action_details");
            printattrib(p,"action_number",(i+1)+"");
            printattrib(p,"action_description",GenericXMLParser.encode(actions[i]));
            printattrib(p,"action_postfix",GenericXMLParser.encode(actionsPostfix[i]));
            printattrib(p,"action_comment",GenericXMLParser.encode(actionsComment[i]));
            p.println();
            newline=true;
            for(int j=0; j<maxcol; j++){
               if(actiontable[i][j].length()>0){
                   p.println("<action_column column_number=\""+(j+1)+"\" column_value=\""+actiontable[i][j]+"\" />");
               }
            }
            p.println("</action_details>");
        }
        p.println("</actions>");
        p.println("</decision_table>");
    }
    
    
	/**
	 * All Decision Tables are executable.
	 */
	public boolean isExecutable() {
		return true;
	}
    
	/**
	 * The string value of the decision table is simply its name.
	 */
	public String stringValue() {
        String number = fields.get("ipad_id"); 
        if(number==null)number = "";
		return number+" "+dtname.stringValue();
	}
	
	/**
	 * The string value of the decision table is simply its name.
	 */
	public String toString() {
		return stringValue();
	}
    
	/**
     * Return the postFix value 
	 */
    public String postFix() {
        return dtname.stringValue();
    }

    /**
	 * The type is Decision Table.
	 */
	public RType type() {
		return dttype;
	}

	/**
	 * @return the actions
	 */
	public String[] getActions() {
		return actions;
	}
	
	/**
	 * @return the actiontable
	 */
	public String[][] getActiontable() {
		return actiontable;
	}

	/**
	 * @return the conditions
	 */
	public String[] getConditions() {
		return conditions;
	}

	/**
	 * @return the conditiontable
	 */
	public String[][] getConditiontable() {
		return conditiontable;
	}
	
	public String getDecisionTableId(){
		return fields.get(RName.getRName("table_number"));
	}
	
	public void setDecisionTableId(String decisionTableId){
		fields.put(RName.getRName("table_number"),decisionTableId);
	}
	
	public String getPurpose(){
		return fields.get(RName.getRName("purpose"));
	}
	
	public void setPurpose(String purpose){
		fields.put(RName.getRName("purpose"),purpose);
	}
	
	public String getComments(){
		return fields.get(RName.getRName("comments"));
	}
	
	public void setComments(String comments){
		fields.put(RName.getRName("comments"),comments);
	}
	
	public String getReference(){
		return fields.get(RName.getRName("policy_reference"));
	}
	
	public void setReference(String reference){
		fields.put(RName.getRName("policy_reference"),reference);
	}

	/**
	 * @return the dtname
	 */
	public String getDtname() {
		return dtname.stringValue();
	}


	/**
	 * @return the ractions
	 */
	public IRObject[] getRactions() {
		return ractions;
}
	/**
	 * @param ractions the ractions to set
	 */
	public void setRactions(IRObject[] ractions) {
		this.ractions = ractions;
	}

	/**
	 * @return the rconditions
	 */
	public IRObject[] getRconditions() {
		return rconditions;
	}

	/**
	 * @param rconditions the rconditions to set
	 */
	public void setRconditions(IRObject[] rconditions) {
		this.rconditions = rconditions;
	}

	/**
	 * @param actions the actions to set
	 */
	public void setActions(String[] actions) {
		this.actions = actions;
	}

	/**
	 * @param actiontable the actiontable to set
	 */
	public void setActiontable(String[][] actiontable) {
		this.actiontable = actiontable;
	}

	/**
	 * @param conditions the conditions to set
	 */
	public void setConditions(String[] conditions) {
		this.conditions = conditions;
	}

	/**
	 * @param conditiontable the conditiontable to set
	 */
	public void setConditiontable(String[][] conditiontable) {
		this.conditiontable = conditiontable;
	}


	/**
	 * @return the actionsComment
	 */
	public final String[] getActionsComment() {
		return actionsComment;
	}


	/**
	 * @param actionsComment the actionsComment to set
	 */
	public final void setActionsComment(String[] actionsComment) {
		this.actionsComment = actionsComment;
	}


	/**
	 * @return the actionsPostfix
	 */
	public final String[] getActionsPostfix() {
		return actionsPostfix;
	}


	/**
	 * @param actionsPostfix the actionsPostfix to set
	 */
	public final void setActionsPostfix(String[] actionsPostfix) {
		this.actionsPostfix = actionsPostfix;
	}


	/**
	 * @return the conditionsComment
	 */
	public final String[] getConditionsComment() {
		return conditionsComment;
	}


	/**
	 * @param conditionsComment the conditionsComment to set
	 */
	public final void setConditionsComment(String[] conditionsComment) {
		this.conditionsComment = conditionsComment;
	}


	/**
	 * @return the conditionsPostfix
	 */
	public final String[] getConditionsPostfix() {
		return conditionsPostfix;
	}


	/**
	 * @param conditionsPostfix the conditionsPostfix to set
	 */
	public final void setConditionsPostfix(String[] conditionsPostfix) {
		this.conditionsPostfix = conditionsPostfix;
	}

    /**
     * A little helpper function that inserts a new column in a table
     * of strings organized as String table [row][column];  Inserts blanks
     * in all new entries, so this works for both conditions and actions.
     * @param table     
     * @param col
     */
    private static void insert(String[][]table, int maxcol, final int col){
        for(int i=0; i<maxcol; i++){
            for(int j=15; j> col; j--){
                table[i][j]=table[i][j-1];
            }
            table[i][col]=" ";
        }   
    }
	/**
     * Insert a new column at the given column number (Zero based) 
     * @param col The zero based column number for the new column
     * @throws RulesException
	 */
    public void insert(int col) throws RulesException {
        if(maxcol>=16){
            throw new RulesException("TableTooBig","insert","Attempt to insert more than 16 columns in a Decision Table");
        }
        insert(conditiontable,maxcol,col);
        insert(actiontable,maxcol,col);
    }
    
    /**
     * Balances an unbalanced decision table.  The additional columns have
     * no actions added.  There are two approaches to balancing tables.  One
     * is to have executed all columns whose conditions are met.  The other is
     * to execute only the first column whose conditions are met.  This 
     * routine executes all columns whose conditions are met.
     */
    public void buildUnbalanced(DTState state, UnbalancedType type) {
        if( 
                conditiontable.length == 0    ||
                conditiontable[0].length == 0 ||           // If we have no conditions, or
                conditiontable[0][0]==null)   {            // we have nothing, or
         	   return;
            }	
       if( 
           conditiontable.length == 0    ||
           conditiontable[0].length == 0 ||           // If we have no conditions, or
           conditiontable[0][0]==null    ||           // we have nothing, or
            conditiontable[0][0].equals("*")){        // If *, we just execute all actions
    	   decisiontree = ANode.newANode(this,0);	  //   checked in the first column             
    	   return;
       }	
       
       if(conditions.length<1){  
           errorlist.add(
                   new CompilerError(
                           IDecisionTableError.Type.CONDITION,
                           "You have to have at least one condition in a decision table", 0,0)
           );        
       }

       /**
        * 
        */
       CNode top = new CNode(this,1,0,this.rconditions[0]);
     
       for(int col=0;col<maxcol;col++){                             // Look at each column.
           boolean nonemptycolumn = false;
           for(int row=0; row<conditions.length; row++){
               String v      = conditiontable[row][col];                     // Get the value from the condition table
               if(v.equals(DASH) || v.equals(" ")){
                   v = DASH;
                   conditiontable[row][col]=DASH;
               }else{
                   nonemptycolumn = true;
               }
           }
           if(nonemptycolumn){    
             try {                          
                int numerrs = errorlist.size()+1;                   // Ignore all but the first error in a column 
                processCol(type,top,0,col,-1);                      // Process a column.
                while(errorlist.size()>numerrs){                    // Throw away all but one.
                    errorlist.remove(errorlist.size()-1);
                }
       
             } catch (Exception e) {
                /** Any error detected is recorded in the errorlist.  Nothing to do here **/                                            
             }
           }
       }
       ANode defaults;
       defaults = new ANode(this);
       addDefaults(top,defaults);                                   // Add defaults to all unmapped branches
       decisiontree = optimize(state, top);                         // Optimize the given tree.
    }     

    /**
     * Replace any untouched branches in the tree with a pointer
     * to the defaults for this table.  We only replace nulls.
     * @param node
     * @param defaults
     * @return
     */
    private DTNode addDefaults(DTNode node, ANode defaults){
        if(node == null ) return defaults; 
        if(node instanceof ANode)return node;
        CNode cnode = (CNode)node;
        cnode.iffalse = addDefaults(cnode.iffalse,defaults);
        cnode.iftrue  = addDefaults(cnode.iftrue, defaults);
        return node;
    }
    
    /**
     * At this level, we just check to make sure the table is okay
     * to optimize.  optimize2 does the real work of optimization, if
     * it is okay to do so.
     * 
     * @param node
     * @return
     */
    private DTNode optimize(DTState state, DTNode node){
    	
    	if(!optimize){                 // We don't want to optimize if
    		return node;               //   we are an ALL table with a 
    	}                              //   reference to policystatements
    	
    	return optimize2(state,node);
    }
    
    /**
     * Replaces the given DTNode with the optimized DTNode.
     * @param node
     * @return
     */
    private DTNode optimize2(DTState state, DTNode node){
    		      	
    	//   with the column numbers.
    	ANode opt = node.getCommonANode(state);	
        if(opt!=null){
            return opt;
        }
        CNode cnode = (CNode) node;
        cnode.iftrue  = optimize2(state, cnode.iftrue);
        cnode.iffalse = optimize2(state, cnode.iffalse);
        if(cnode.iftrue.equalsNode(state, cnode.iffalse)){
            cnode.iftrue.addNode(cnode.iffalse);
            return cnode.iftrue;
        }
        return cnode;
    }
    
    /**
     * Build a path through the decision tables for a particular column.
     * This routine throws an exception, but the calling routine just ignores it.
     * That way we don't flood the error list with lots of duplicate errors.
     * @param here
     * @param row
     * @param col
     * @param star -- Have we encountered a star as a column value yet.
     * @return
     */
    private DTNode processCol(UnbalancedType code, DTNode here, int row, int col, int istar) throws Exception{
        
        if (starColumn >=0 && col>starColumn){   
            int rowX = 0;
            
            for(;conditiontable[rowX][col] == DASH;rowX++);
            
            errorlist.add(
                    new CompilerError (
                         IDecisionTableError.Type.TABLE,
                         "Only one 'star' column is allowed, and it must be the last column.","*",rowX));
        }
        
        if(row >= conditions.length){                           // Ah, end of the column!
            
            ANode thisCol = ANode.newANode(this, col);          // Get a new ANode for this column
            thisCol.setStar(istar>=0);
            
            // Star columns can do one of two things.  They can fill all unused paths through
            // the decision table (the "Otherwise" or "default" option), or they can fill all paths 
            // (used and unused) through the decision table (the "always" option).
            if(istar>=0){
                
                starColumn = col;             // Record the star column for error checking purposes.
                
                boolean otherwise =    conditions[istar].trim().equalsIgnoreCase("otherwise")
                                    || conditions[istar].trim().equalsIgnoreCase("default");
                
                boolean always    =    conditions[istar].trim().equalsIgnoreCase("always");
                
                if (!always && !otherwise){
                    errorlist.add(
                            new CompilerError (
                                 IDecisionTableError.Type.CONDITION,
                                 "A condition row with a star ('*') must have a condition with a value"+
                                 " of 'default', 'otherwise', or 'always'","*",0));
                                 
                }
                
                if(here == null){               // If here is null, then this is an unused path.
                    return thisCol;             // This will be the only path out for "Otherwise", but 
                }                               // "Always" is going to cover it too.
                
                // Okay, this path is covered by some path.  We will leave it unchanged for "otherwise".
                if(otherwise){
                    otherwiseColumn = col;      // Remember which column is the "otherwise" column.
                    return here;
                }    
                
                // Okay, this path is covered by some path, so we will add to it for "always".
                if (always){
                    alwaysColumn = col;         // Remember which column is the "always" column.
                    thisCol.addNode((ANode)here);
                    return thisCol;
                }
                
            }
            
            
            if(here!=null && code == UnbalancedType.FIRST){           // If we execute only the First, we are done!
               return here;
            }

            if( here!=null && code == UnbalancedType.ALL){            // If Some path lead here, fold the
                thisCol.addNode((ANode)here);           			  //    old stuff in with this column.
                return thisCol;
            }
            
            return thisCol;                                           // Return the mix!
        }

        String v      = conditiontable[row][col];                     // Get the value from the condition table
        
        boolean dcare = v == DASH;                                    // Standardize Don't cares.
        boolean yes   = v.equalsIgnoreCase("y");
        boolean no    = v.equalsIgnoreCase("n");
                
        if(istar>=0 && (yes | no )){
            errorlist.add(
               new CompilerError (
                    IDecisionTableError.Type.CONDITION,
                    "Cannot follow a '*' with a '"+v+"' at row "+(row+1)+" column "+(col+1),v,0));
        }

        if(v.equalsIgnoreCase("*")){
            istar = row;
        }
        
        if(istar<0 && !yes && !no && !dcare){
            errorlist.add(
                new CompilerError (
                        IDecisionTableError.Type.CONDITION,
                        "Bad value in Condition Table '"+v+"' at row "+(row+1)+" column "+(col+1),v,0));
        }
        
        if((here==null || here.getRow()!= row ) && dcare){            // If we are a don't care, but not on a row
            return processCol(code,here,row+1,col,istar);             //   that matches us, we skip this row for now.
        }
        
        if(istar>=0){                                                 // If a star, then return the action node  
            DTNode t = processCol(code,here,row+1,col,istar);         //   for this position.
            t.setStar(true);
            return t;
        } 
        
        if(here==null){                                               // If this node is null,
            here = new CNode(this,col,row,rconditions[row]);          //   a condition node, create it!
        }else if ( here.getRow()!= row ){                             // If this is the wrong node, and I need 
            CNode t = new CNode(this,col,row,rconditions[row]);       //   a condition node, create a new one and insert it.
            t.iffalse = here;                                         // Put the node I have on the false tree
            t.iftrue  = here.cloneDTNode();                           //   and its clone on the true path.
            here = t;                                                 // Continue with the new node.  
        }
        
        if(yes || dcare){                                                   // If 'y' or a don't care,
            DTNode next = ((CNode) here).iftrue;                            // Recurse on the True Branch.
            DTNode t    = processCol(code,next,row+1,col,-1);
            ((CNode) here).iftrue = t;
            if(yes && t.getStar()){
                errorlist.add(
                   new CompilerError (
                        IDecisionTableError.Type.CONDITION,
                        "Cannot follow a 'Y' with a '*' at row "+(row+1)+" column "+(col+1),v,0));
            }
        }
        if (no || dcare){                                                   // If 'n' or a don't care,  
            DTNode next = ((CNode) here).iffalse;                           // Recurse on the False branch.  Note that
            DTNode t    = processCol(code,next,row+1,col,-1);
            ((CNode) here).iffalse = t;
            if(no && t.getStar()){                                          
                errorlist.add(
                   new CompilerError (
                        IDecisionTableError.Type.CONDITION,
                        "Cannot follow a 'N' with a '*' at row "+(row+1)+" column "+(col+1),v,0));
            }
        }
        
        return here;                                                  // Return the Condition node.
    }
    
    /**
     * In the case of an unbalanced decision table, this method returns a balanced
     * decision table using one of the two unbalanced rules:  FIRST (which executes only
     * the first column whose conditions are matched) and ALL (which executes all columns
     * whose conditions are matched).  If the decision table is balanced, this method returns
     * an "optimized" decision table where all possible additional "don't cares" are inserted.
     * 
     * @return
     */
    RDecisionTable getBalancedTable(IRSession session) throws RulesException{
        if(balanceTable==null)balanceTable = new BalanceTable(this);
        return balanceTable.balancedTable(session);
    }
    
    public BalanceTable getBalancedTable() throws RulesException {
        return new BalanceTable(this);
    }
    
    public Iterator<RDecisionTable> DecisionTablesCalled(){
        ArrayList<RDecisionTable> tables = new ArrayList<RDecisionTable>();
        ArrayList<RArray>         stack  = new ArrayList<RArray>();
        for(int i=0;i<ractions.length;i++){
           addTables(ractions[i],stack,tables);
        }
        return tables.iterator();
    }
    
    private void addTables(IRObject action,List<RArray> stack, List<RDecisionTable> tables){
        if(action==null)return;
        if(action.type().getId()==iArray){
            RArray array = (RArray)action;
            if(stack.contains(array))return;    // We have already been here.
            stack.add(array);
            try {     // As this is an array, arrayValue() will not ever throw an exception
                Iterator<?> objects = array.arrayValue().iterator();
                while(objects.hasNext()){
                    addTables((IRObject) objects.next(),stack,tables);
                }
            } catch (RulesException e) { }
        }
        if(action.type().getId()==iDecisiontable && !tables.contains(action)){
            tables.add((RDecisionTable)action);
        }
    }
    /**
     * Returns the list of Decision Tables called by this Decision Table
     * @return
     */    
    ArrayList<RDecisionTable> decisionTablesCalled(){
        ArrayList<RDecisionTable> calledTables = new ArrayList<RDecisionTable>();

        addlist(calledTables, rinitialActions);
        addlist(calledTables, rconditions);
        addlist(calledTables, ractions);
        
        return calledTables;
    }
    /**
     * We do a recursive search down each IRObject in these lists, looking for
     * references to Decision Tables.  We only add references to Decision Tables
     * to the list of called tables if the list of called tables doesn't yet have
     * that reference.
     * 
     * @param calledTables
     * @param list
     */
    private void addlist(ArrayList<RDecisionTable> calledTables, IRObject [] list){
        for(int i=0; i<list.length; i++){
            ArrayList<RDecisionTable> tables = new ArrayList<RDecisionTable>();
            ArrayList<RArray>         stack  = new ArrayList<RArray>();
            getTables(stack, tables, list[i]);
            for(RDecisionTable table : tables){
                if(!calledTables.contains(table))calledTables.add(table);
            }
        }
    }
    /**
     * Here we do a recursive search of all the constructs in an IROBject.  This
     * is because some IRObjects are arrays, so we search them as well.
     * @param obj
     * @return
     */
    private ArrayList<RDecisionTable> getTables(ArrayList<RArray>stack, ArrayList<RDecisionTable> tables, IRObject obj){
        
        if(obj instanceof RDecisionTable) tables.add((RDecisionTable) obj);
        if(obj instanceof RArray && !stack.contains(obj)){
            stack.add((RArray) obj);
            for(IRObject obj2 : (RArray) obj){
                getTables(stack,tables,obj2);
            }
        }
        return tables;
    }

    public String[] getPolicystatements() {
        return policystatements;
    }
    
    public String [] getPolicyStatementsBalanced(IRSession session) {
        if(policyStatementsBalanced == null){
            try {
                RDecisionTable dt        = getBalancedTable(session);
                actiontablebalanced      = dt.actiontable;
                conditiontablebalanced   = dt.conditiontable;
                policyStatementsBalanced = dt.policystatements;
            } catch (RulesException e) {   }
        }
        return policyStatementsBalanced;
   
    }

    public String[] getPolicystatementsPostfix() {
        return policystatementsPostfix;
    }

    public IRObject[] getRpolicystatements() {
        return rpolicystatements;
    }

    public DTNode getDecisiontree() {
        return decisiontree;
    }
    /**
     * This method provides field values given a field name.  A few "virtual" field names are
     * supported so as to avoid having to have special code to access these values.  They include
     * the Table_Name, File_Name, 
     * @param fieldname
     * @return
     */
    public String getField(String fieldname){
        RName fn = RName.getRName(fieldname);
        if(fn.equals(table_name)){
            return dtname.stringValue();
        }else if(fn.equals(type_name)){
            switch(type){
                case ALL :      return "ALL";
                case FIRST :    return "FIRST";
                case BALANCED : return "BALANCED";
                default :       return "UNDEFINED";
            }
        }
        return fields.get(fn);
    }
    
}
