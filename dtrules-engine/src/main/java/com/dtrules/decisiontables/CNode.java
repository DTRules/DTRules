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

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.session.DTState;
/**
 * The Condition Node evaluates conditions within a Decision
 * table.
 * @author Paul Snow
 *
 */
public class CNode implements DTNode {

    final int            column;                  //Column that created this node
    final int            conditionNumber;         //NOPMD 
    final IRObject       condition;
    final RDecisionTable dtable;                  // Pointer back to the Decision Table for debug purposes
    boolean              star;
    
    DTNode iftrue  = null;
    DTNode iffalse = null;

    
    
    /**
     * Clone this CNode and all the CNodes referenced by this CNode
     */
    public CNode cloneDTNode (){
        CNode newCNode = new CNode(dtable,column,conditionNumber,condition);
        newCNode.iftrue  = iftrue  == null ? null : iftrue.cloneDTNode(); 
        newCNode.iffalse = iffalse == null ? null : iffalse.cloneDTNode();
        return newCNode;
    }
    
    public int getRow(){ return conditionNumber; }
    /**
     * Two CNodes are equal if their true paths and their
     * false paths are the same.
     * And those CommonANodes have to be equal.
     */
	public boolean equalsNode(DTState state, DTNode node) {
        if(node instanceof CNode){
            CNode cnode = (CNode)node;
            if(cnode.conditionNumber != conditionNumber){
                return false;
            }
            return(cnode.iffalse.equalsNode(state, iffalse) && 
               cnode.iftrue.equalsNode(state, iftrue));
        }else{
            ANode me  = getCommonANode(state);  // Get this CNode's commonANode.
            if(me==null)return false;           // If none exists, it can't be equal!
            ANode him = getCommonANode(state);  // Get the other DTNode's commonANode
            if(him==null)return false;          // If none exists, it can't be equal!
            return me.equalsNode(state, him);   // Return true if this node matches that node!
        }    
    }
	
    /**
     * To have a CommonANode, every path through the CNode (i.e. both
     * the true path and the false path) has to have the same commonANode.
     * So both iftrue and iffalse have to have a commonANode, and those have
     * to match.
     */
    public ANode getCommonANode(DTState state) {
        if(state.testState(DTState.TRACE) &&                // If we are tracing, don't do this.
                iftrue.equalsNode(state, iffalse)){         // If both true/false paths are the same, 
            ANode trueSide  = iftrue.getCommonANode(state); //   then we will merge them.  Get the 
            ANode falseSide = iffalse.getCommonANode(state);
            trueSide.addNode(falseSide);                    
            if(trueSide instanceof ANode){                  // We add them together so we maintain the tracking
                ((ANode) trueSide).addNode((ANode)falseSide);   //   of columns tested.
            }
            return trueSide;  
        }
        return null;                                        // If they don't match, I have to return false!
    }

  
	
	CNode(RDecisionTable dt, int column, int conditionNumber, IRObject condition){
        this.column          = column;
        this.conditionNumber = conditionNumber; 
        this.condition       = condition;
        this.dtable          = dt;
    }
	
	public int countColumns(){
	    int t = iftrue.countColumns();
	    int f = iffalse.countColumns();
	    return t+f;
	}
	    
	public void execute(DTState state) throws RulesException{
        boolean result;
        try {
            if(state.testState(DTState.TRACE)){
            	if(state.testState(DTState.VERBOSE)){
	            	state.traceTagBegin("Condition", "n",(conditionNumber+1)+"");
	                    state.traceInfo("Formal", dtable.getConditions()[conditionNumber]);
	                    state.traceInfo("Postfix", dtable.getConditionsPostfix()[conditionNumber]);
	                    state.traceTagBegin("execute");
	                    result = state.evaluateCondition(condition);
	                    state.traceTagEnd();
	                    state.traceInfo("result", "v",(result?"Y":"N"),null);
	                state.traceTagEnd();
            	}else{
            		 result = state.evaluateCondition(condition);
            		 state.traceInfo("Condition", "n",(conditionNumber+1)+"","v",(result?"Y":"N"),null);
            	}
            }else{
                result = state.evaluateCondition(condition);
            }    		
    		
        } catch (RulesException e) {
            e.isFirstAction();  // Avoids a bogus "first action"
            if(state.testState(DTState.TRACE)){
                state.traceTagEnd();
                state.traceInfo("result", "v", "ERROR",null);
                state.traceTagEnd();
            }
            e.setSection("Condition",conditionNumber+1);
            e.setFormal(dtable.getConditions()[conditionNumber]);
            throw e;
        } catch (Exception e){
            RulesException re = new RulesException(e.getClass().getName(), e.getStackTrace()[0].getClassName(), e.getMessage());
            re.isFirstAction();  // Avoids a bogus "first action"
            if(state.testState(DTState.TRACE)){
                state.traceTagEnd();
                state.traceInfo("result", "v", "ERROR",null);
                state.traceTagEnd();
            }
            re.setSection("Condition",conditionNumber+1);
            re.setFormal(dtable.getConditions()[conditionNumber]);
            throw re;
        }

        if(result){
            iftrue.execute(state);
        }else{
            iffalse.execute(state);
        }

	}

	public Coordinate validate() {
		if(iftrue  == null || iffalse == null){
           return new Coordinate(conditionNumber,column); 
        }   
	    Coordinate result = iffalse.validate();
        if(result!=null){
            return result;
        }
        
        return iftrue.validate();
	}

	public String toString(){
		return "Condition Number "+(conditionNumber+1);
	}
	
	public void addNode(DTNode _node) {
	    if(!(_node instanceof CNode)){
	        throw new RuntimeException("Shouldn't ever attempt to combine nodes of different types");
	    }
	    CNode node = (CNode) _node;
	    iftrue.addNode(node.iftrue);
	    iffalse.addNode(node.iffalse);
	}

    public boolean getStar() {
        return star;
    }

    public void setStar(boolean star) {
        this.star = star;
    }
	
}
