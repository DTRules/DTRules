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

import java.util.ArrayList;
import java.util.Iterator;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.session.DTState;

/**
 * ANodes execute a list of actions.
 * @author ps24876
 *
 */
public class ANode implements DTNode {
    RDecisionTable       dtable;                               // The Decision Table to which this ANode belongs
    ArrayList<IRObject>  action   = new ArrayList<IRObject>(); // The action postfix
    ArrayList<Integer>   anumbers = new ArrayList<Integer>();  // The action numbers (for tracing purposes)
    ArrayList<Integer>   columns  = new ArrayList<Integer>();  // Keep track of the columns that send us here, for tracing
    String               section;                              // What section we are being called from.
    
    
    public DTNode cloneDTNode(){
        ANode newANode = new ANode(dtable);
        newANode.action.addAll(this.action);
        newANode.anumbers.addAll(this.anumbers);
        newANode.columns.addAll(this.columns);
        newANode.section = this.section;
        return newANode;
    }
    
    public int getRow(){ return -1; }
    
    /**
     * Create and return an new ANode instance that holds all of the 
     * actions whose supposed to execute if this column executes.
     * @param col (the zero based column number)
     */
    public static ANode newANode(RDecisionTable dt, int col){
       //int cnt = 0;
       ArrayList<IRObject> list    = new ArrayList<IRObject> ();
       ArrayList<Integer>  numbers = new ArrayList<Integer>();
       for(int i=0; i< dt.actiontable.length; i++){
           if(dt.actiontable[i][col].equalsIgnoreCase("x")){
        	   if(dt.ractions!=null && dt.ractions.length>=i){
                  list.add(dt.ractions[i]);
                  numbers.add(Integer.valueOf(i));
        	   }   
           }
       }
       return new ANode(dt,col+1, list, numbers);
    }   
    /**
     * Combines into this node the given node.  Used to collapse two
     * nodes when both nodes should be executed.
     */
    public void addNode(DTNode _node){
        if(!(_node instanceof ANode)){
            throw new RuntimeException("Shouldn't every call if Node types don't match!");
        }
        ANode node = (ANode)_node;

        for(Integer column : node.columns){
            if(!columns.contains(column)) columns.add(column);
            for(int i=columns.size()-1; i>0 ; i--){
                for(int j=0; j<i; j++){
                    if(columns.get(j)>columns.get(j+1)){
                        Integer hld = columns.get(j);
                        columns.set(j,columns.get(j+1));
                        columns.set(j+1,hld);
                    }
                }
            }
        }
       
        for(int i=0;i<node.anumbers.size(); i++){
            Integer index = node.anumbers.get(i);
            if(!anumbers.contains(index)){
                int v   = index.intValue();
                int pos = 0;
                while(pos< anumbers.size() && anumbers.get(pos).intValue()<v) pos++;
                anumbers.add(pos,index);
                action.add(pos,dtable.ractions[v]);
            }
        }
    }
    
    public int countColumns(){
        return 1;
    }
    
    private ANode(RDecisionTable dt, int column, ArrayList<IRObject> objects, ArrayList<Integer> numbers){
        dtable      = dt;
    	columns.add(new Integer(column));
    	action      = objects;
        anumbers    = numbers;
    }
    
    public ANode(RDecisionTable dt){
        dtable      = dt;
    }
    
    /** Give the list of columns that got us to this ANode.  Unbalanced tables
     *  can give us multiple columns in a single ANode
     * @param columns
     * @return
     */
    public String prtColumns(ArrayList<Integer> columns){
        String s="";
        for(Integer column : columns){
            s += column.toString()+" ";
        }
        return s;
    }
    
	public void execute(DTState state) throws RulesException {
        Iterator<Integer> inum = anumbers.iterator();
        if(state.testState(DTState.TRACE)){
            state.traceTagBegin("column", "n",prtColumns(columns));
        }
        int num = 0;
        try {
            if(state.testState(DTState.TRACE)){
                for(IRObject v : action){
                    num = inum.next().intValue();
                    state.traceTagBegin("action","n",(num+1)+"");
                        state.traceInfo("formal",dtable.getActions()[num]);
                        int d = state.ddepth();                
                        String section = state.getCurrentTableSection();
                        int    numhld  = state.getNumberInSection();
                        state.setCurrentTableSection("Action",num);
                        state.evaluate(v);
                        state.setCurrentTableSection(section, numhld);
                        if(d!=state.ddepth()){
                            throw new RulesException("data stack unbalanced","ANode Execute","Action "+(num+1)+" in table "+dtable.getDtname());
                        }
                    state.traceTagEnd();
                }
            }else{
                for(IRObject v : action){
                    num = inum.next().intValue();
                    String section = state.getCurrentTableSection();
                    int    numhld  = state.getNumberInSection();
                    state.setCurrentTableSection("Action",num);
                    state.evaluate(v);
                    state.setCurrentTableSection(section, numhld);
                }  
            }
        } catch (RulesException e) {
            boolean first = e.isFirstAction(); 
            if(state.testState(DTState.TRACE)){
                if(first){
                    state.traceInfo("Error_Detected",null);
                }
                state.traceTagEnd();
            }
            e.setSection("Action",num+1);
            e.setFormal(dtable.getActions()[num]);
            throw e;
        } catch (Exception e){
            RulesException re = new RulesException(e.getClass().getName(), e.getStackTrace()[0].getClassName(), e.getMessage());
            re.isFirstAction(); /** Just so we note that this is the first action */
            if(state.testState(DTState.TRACE)){
                state.traceInfo("Error_Detected",null);
                state.traceTagEnd();
            }
            re.setSection("Action",num+1);
            re.setFormal(dtable.getActions()[num]);
            throw re;
        }


        if(state.testState(DTState.TRACE)){
            state.traceTagEnd();
        }
	}

	public Coordinate validate() {
		return null;
	}
    
	public String toString(){
		return "Action Node for columns "+(prtColumns(columns));
	}
	/**
     * An Action Node is equal to another DTNode if 1) it has one and
     * only one set of actions it executes regardless of the path taken, and
     * 2) if the actions this node takes are exactly the as the node provided. 
	 */
    public boolean equalsNode(DTState state, DTNode node) {
        ANode other = node.getCommonANode(state);               // Get the common path            
        if(other==null)return false;                            // No common path? Not Equal then!
        if(other.anumbers.size()!=anumbers.size()) return false;// Must be the same length.
        for(int i = 0; i< anumbers.size(); i++){
            if(!other.anumbers.get(i).equals(anumbers.get(i)))return false;  //   Make sure each action is the same action.
        }                                                                    //     If a mismatch is found, Not Equal!!!
        return true;
    }

    public ANode getCommonANode(DTState state) {
        return this;
    }
    
    
}
