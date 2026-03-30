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
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.IRSession;

/**
 * Builds a balanced decision table from the given decision table.
 * @author paul snow
 * Mar 5, 2007
 *
 */
public class BalanceTable {

    private int         maxRow=0;
    private int         maxCol=0;
    private int         maxARow=0;
    Map<Long, String>   ctable;
    Map<Long, String>   atable;
    List<List<Integer>> b_columns;

    void putCT(long row, long col, String v) {
        ctable.put((row<<32)+col, v);
    }

    String getCT(long row, long col){
        String v = ctable.get((row<<32)+col);
        if(v==null) return " ";
        return v;
    }
    
    void putAT(long row, long col, String v) {
        atable.put((row<<32)+col, v);
    }

    String getAT(long row, long col){
        String v = atable.get((row<<32)+col);
        if(v==null) return " ";
        return v;
    }
    
    final   RDecisionTable  dt;
    public BalanceTable(RDecisionTable dt) throws RulesException{
        this.dt = dt;
    } 
    
    RDecisionTable balancedTable (IRSession s) throws RulesException{
        if(!dt.isCompiled())dt.build(s.getState());
        if(!dt.isCompiled()){
            throw new RulesException("DecisionTableError","balancedTable()","Malformed Decision Table");
        }
        RDecisionTable btable = (RDecisionTable) dt.clone(s);
        if(dt.decisiontree == null) return btable;
        
        ctable = new HashMap<Long, String>();
        atable = new HashMap<Long, String>();
        b_columns = new ArrayList<List<Integer>>();
        
        filltable(0,0,dt.decisiontree); 
        btable.conditiontable = new String[dt.getConditiontable().length][maxCol];
        btable.actiontable    = new String[dt.getActiontable().length]   [maxCol];
        for(int col = 0; col < maxCol; col++){
            for(int row = 0; row < btable.getConditiontable().length; row++){
                String v = getCT(row,col);
                btable.conditiontable[row][col]=v;
            }
            for(int row=0; row < btable.getActiontable().length; row++){
                String v = getAT(row,col);
                btable.actiontable[row][col]=v;
            }
        }
        
        btable.policystatements = new String[maxCol+1];
        for(int i=1; i< maxCol+1; i++){
            btable.policystatements[i] ="";
            if(b_columns.get(i)!=null) for(int j : b_columns.get(i)){
                if(j < dt.policystatements.length){
                    btable.policystatements[i] += dt.getPolicystatements()[j] +";  ";
                }
            }
        }
        
        btable.setType(RDecisionTable.Type.BALANCED);
        btable.build(s.getState());
        return btable;
    }

    public String getPrintableTable() throws RulesException {
        try {
            if (dt.decisiontree == null)
                return "empty table";
            
            ctable    = new HashMap<Long, String>();
            atable    = new HashMap<Long, String>();
            b_columns = new ArrayList<List<Integer>>();
            
            
            filltable(0, 0, dt.decisiontree);
            StringBuffer buff = new StringBuffer();
            buff.append("Number of Columns: " + maxCol + "\n");
            buff.append("Type: "+dt.getType().name()+"\n");
            String spacer = " ";
            if (maxCol < 25)
                spacer = "  ";

            for (int i = 0; i < maxRow; i++) {
                String row = (i + 1) + "";
                row = "    ".substring(row.length()) + row + spacer;
                buff.append(row);
                for (int j = 0; j < maxCol; j++) {
                    String v = getCT(i, j);
                    if ((v == null) || (v.equals(" ") || v.equals(RDecisionTable.DASH))) {
                        putCT(i, j, RDecisionTable.DASH);
                    }
                    buff.append(getCT(i, j));
                    buff.append(spacer);
                }
                buff.append("\r\n");
            }
            buff.append("\n");
            for (int i = 0; i <= maxARow; i++) {
                String row = (i + 1) + "";
                row = "    ".substring(row.length()) + row + spacer;
                buff.append(row);
                for (int j = 0; j < maxCol; j++) {
                    buff.append(getAT(i, j));
                    buff.append(spacer);
                }
                buff.append("\r\n");
            }
            buff.append("\r\n");
            return buff.toString();
        } catch (ArrayIndexOutOfBoundsException e) {
            throw new RulesException("Invalid", dt.getName().stringValue(),
                    "Decision Table is too complex (generates too many columns");
        }
    }     
    
    /**
     * This routine takes the given Decision Table, and runs down its Decision Tree,
     * and builds up the balanced Decision Tree in the sparse arrays atable and ctable.
     * It also collects the column information from the action nodes and stores them
     * per each column in the b_columns list.
     * 
     * From this the calling routine (either printing or filling out a balanced Decision
     * Table object) can process to get the balanced information
     *
     * @param row
     * @param col
     * @param node
     * @return
     */
    private int filltable(int row, int col, DTNode node){ 
        
        if(node.getClass()==CNode.class){           // The Node is a Condition Node
          int ncol;
          CNode cnode = (CNode) node;
          
          if(cnode.conditionNumber!=row){
              ncol = filltable(row+1,col, node);
              for(int i=col;i<ncol;i++){
                  putCT(row,i,RDecisionTable.DASH);
              }
              return ncol;     
          }
          
          ncol = filltable(row+1,col,cnode.iftrue);
          for(int i=col;i<ncol;i++)putCT(row,i,"y");
          col  = ncol;
          ncol = filltable(row+1,col,cnode.iffalse);
          for(int i=col;i<ncol;i++)putCT(row,i,"n");
          col  = ncol;
       }else{                                       // The Node is an action node
          putCT(row,col,RDecisionTable.DASH); 
          ANode anode = (ANode)node;
          for(int i=0;i<anode.anumbers.size();i++){
              int index = anode.anumbers.get(i).intValue();
              putAT(index,col,"x");
              if(maxARow<index) maxARow = index;
          }
          while(col+1 >= b_columns.size()){           // Make sure there is room for the
              b_columns.add(null);                  //   column information
          }
          b_columns.add(col+1, anode.getColumns());
          col++;
       }
       maxRow = maxRow<row?row:maxRow;
       maxCol = maxCol<col?col:maxCol;
       return col;
    }
    
}
