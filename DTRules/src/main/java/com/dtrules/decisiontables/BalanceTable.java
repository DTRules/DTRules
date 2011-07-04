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
package com.dtrules.decisiontables;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.IRSession;

/**
 * Builds a balanced decision table from the given decision table.
 * @author paul snow
 * Mar 5, 2007
 *
 */
public class BalanceTable {

    private int maxRow=0,maxCol=0;
    private int maxARow=0;
    private String ctable[][] = new String[100][10240];
    private String atable[][] = new String[100][10240];
    
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
        ctable = btable.conditiontable;
        atable = btable.actiontable;
        filltable(0,0,btable.decisiontree); 
        btable.setType(RDecisionTable.Type.BALANCED);
        btable.build(s.getState());
        return btable;
    }

    public String getPrintableTable(){
        if(dt.decisiontree==null)return "empty table";
        filltable(0, 0, dt.decisiontree);
        StringBuffer buff = new StringBuffer();
        buff.append("Number of Columns: "+maxCol+"\r\n\r\n");
        String spacer = " ";
        if(maxCol < 25) spacer = "  ";
        
        
        for(int i=0;i<maxRow;i++){
            String row = (i+1)+""; 
            row = "    ".substring(row.length())+row+spacer;
            buff.append(row);
            for(int j=0;j<maxCol; j++){
                String v = ctable[i][j];
                if(  (v == null) || (v.equals(" ") || v.equals(RDecisionTable.DASH))){
                    ctable[i][j]=RDecisionTable.DASH;
                }
                buff.append(ctable[i][j]);
                buff.append(spacer);
            }
            buff.append("\r\n");
        }
        buff.append("\n");
        for(int i=0;i<=maxARow;i++){
            String row = (i+1)+""; 
            row = "    ".substring(row.length())+row+spacer;
            buff.append(row);
            for(int j=0;j<maxCol; j++){
                if(atable[i][j]==null)atable[i][j]=" ";
                buff.append(atable[i][j]);
                buff.append(spacer);
            }
            buff.append("\r\n");
        }
        buff.append("\r\n");
        return buff.toString();
    }
     
    
    private int filltable(int row, int col, DTNode node){ 
        
        if(node.getClass()==CNode.class){
          int ncol;
          CNode cnode = (CNode) node;
          
          if(cnode.conditionNumber!=row){
              ncol = filltable(row+1,col, node);
              for(int i=col;i<ncol;i++){
                  ctable[row][i]=RDecisionTable.DASH;
              }
              return ncol;     
          }
          
          ncol = filltable(row+1,col,cnode.iftrue);
          for(int i=col;i<ncol;i++)ctable[row][i]="y";
          col  = ncol;
          ncol = filltable(row+1,col,cnode.iffalse);
          for(int i=col;i<ncol;i++)ctable[row][i]="n";
          col  = ncol;
       }else{
          ctable[row][col]=RDecisionTable.DASH; 
          ANode anode = (ANode)node;
          for(int i=0;i<anode.anumbers.size();i++){
              int index = anode.anumbers.get(i).intValue();
              atable[index][col]="x";
              if(maxARow<index) maxARow = index;
          }
          col++;
       }
       maxRow = maxRow<row?row:maxRow;
       maxCol = maxCol<col?col:maxCol;
       return col;
    }
    
}
