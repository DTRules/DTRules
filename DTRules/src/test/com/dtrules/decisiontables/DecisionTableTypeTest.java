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

import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Iterator;

import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.mapping.LoadMapping;
import com.dtrules.mapping.Mapping;
import com.dtrules.session.DTState;
import com.dtrules.session.ICompilerError;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.GenericXMLParser;

public class DecisionTableTypeTest {

    int maxRow=0,maxCol=0;
    String ctable [][] = new String[128][128];
    String atable [][] = new String[128][128];
    RDecisionTable dt;
    
    public DecisionTableTypeTest(IRSession session, RuleSet rs) throws RulesException{
        
        dt = new RDecisionTable(session, rs, "test");

        String ctable[][] = { 
                { "y", "n", "n", "y", "d", "a" },
                { "n", "n", "y", " ", " ", " " },
                { "y", "y", "y", "n", " ", " " },
                { "y", "y", " ", " ", " ", " " },
                { " ", " ", "y", " ", " ", " " },
                { "y", " ", " ", " ", " ", " " },
                 
        };
        dt.conditiontable = ctable;

        String atable[][] = { 
                { "x", " ", " ", " ", " ", " " },
                { " ", "x", "x", " ", " ", " " },
                { " ", "x", "x", " ", " ", " " },
                { " ", " ", " ", "x", "x", " " },
                { " ", " ", " ", " ", "x", " " },
                { " ", " ", " ", "x", " ", "x" }, 
        };
        dt.maxcol = ctable[0].length;
        dt.actiontable = atable;
        dt.rconditions = new IRObject[ctable.length];
        dt.ractions = new IRObject[atable.length];

        String cps[] = new String[ctable.length];
        for (int i = 0; i < cps.length; i++)
            cps[i] = "{ 'Condition " + i + "' }";

        String aps[] = new String[atable.length];
        for (int i = 0; i < aps.length; i++)
            aps[i] = "{ 'Action    " + i + "' }";

        dt.conditions = cps;
        dt.conditionsPostfix = cps;
        dt.actions = aps;
        dt.actionsPostfix = aps;
        dt.setType(RDecisionTable.Type.FIRST);
    } 
    
    public static void main(String[] args) {
        String path = "C:\\eclipse\\workspace\\EB_POC\\CA HCO Plan\\xml\\";
        String file = "DTRules.xml";
        if(args.length>0){
            file = args[0];
        }
        
        try {
            RulesDirectory rd       = new RulesDirectory(path,file);
            RuleSet        rs       = rd.getRuleSet(RName.getRName("ebdemo"));
            IRSession      session  = rs.newSession();
            DTState        state    = session.getState();
            DecisionTableTypeTest test;
            
            test = new DecisionTableTypeTest(session,rs);
            test.dt.setType(RDecisionTable.Type.FIRST);
            test.dt.build();
            test.printtable ();
            
            test = new DecisionTableTypeTest(session,rs);
            test.dt.setType(RDecisionTable.Type.ALL);
            test.dt.build();
            test.printtable ();
            
            
        } catch (Exception e) {
            e.printStackTrace(System.out);
        } 
        
    } 
        
    void printtable(){
       maxRow = maxCol = 0;
       filltable(0,0,dt.decisiontree);
      
       System.out.print("                   ");
       for(int i=0;i<maxCol;i++){
           System.out.print(i+((i<10)?"  ":" "));
       }
       System.out.println();
       for(int i=0;i<maxRow;i++){
           System.out.print(dt.conditions[i]+"  ");
           for(int j=0; j<maxCol;j++){
               System.out.print(ctable[i][j]==null?"-  ":ctable[i][j]+"  ");
           }
           System.out.println();
       }
       System.out.println("--------------------------------------");
       for(int i=0;i<dt.actions.length;i++){
           System.out.print(dt.actions[i]+"  ");
           for(int j=0;j<maxCol;j++){
               System.out.print(atable[i][j]==null?"   ":"X  ");
           }
           System.out.println();
       }
       Iterator<ICompilerError> errors = dt.getErrorList().iterator();
       while(errors.hasNext()){
           ICompilerError error = errors.next();
           System.out.println(error.getMessage()+
                   " on "+"Row " + error.getRow()+" Column "+error.getCol());
                  
           
       }
    } 
    
    
    private int filltable(int row, int col, DTNode node){ 
       if(node.getClass()==CNode.class){
          int ncol;
          CNode cnode = (CNode) node;
          
          if(cnode.conditionNumber!=row){
              ncol = filltable(row+1,col, node);
              for(int i=col;i<ncol;i++)ctable[row][i]="-";
              return ncol;     
          }
          
          ncol = filltable(row+1,col,cnode.iftrue);
          for(int i=col;i<ncol;i++)ctable[row][i]="y";
          col  = ncol;
          ncol = filltable(row+1,col,cnode.iffalse);
          for(int i=col;i<ncol;i++)ctable[row][i]="n";
          col  = ncol;
       }else{
          ctable[row][col]="-"; 
          ANode anode = (ANode)node;
          for(int i=0;i<anode.anumbers.size();i++){
              int index = anode.anumbers.get(i).intValue();
              atable[index][col]="x";
          }
          col++;
       }
       maxRow = maxRow<row?row:maxRow;
       maxCol = maxCol<col?col:maxCol;
       return col;
    }
    
}
