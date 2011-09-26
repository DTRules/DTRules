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
import com.dtrules.session.DTState;

public interface DTNode {
   static class Coordinate {
       int             row;
       int             col;
       
    Coordinate(int row, int col){
        this.row = row;
        this.col = col;
    }
       
    public int getCol() {
        return col;
    }
    
    public void setCol(int col) {
        this.col = col;
    }
    public int getRow() {
        return row;
    }
    public void setRow(int row) {
        this.row = row;
    }
   }
  
   public int getRow();
   
   /**
    * Returns a clone of this node, and all nodes below (if it
    * has subnodes).
    * @return
    */
   public DTNode cloneDTNode();
   
   public int countColumns();
   
   void       execute(DTState state) throws RulesException;
   Coordinate validate();
   
   /**
    * For two DTNodes are equal if every path through both nodes
    * execute exactly the same set of Actions.
    * @param node
    * @return
    */
   boolean    equalsNode(DTState state, DTNode node); 
   /**
    * Returns an ANode which represents the execution of every 
    * path through the given DTNode.  If different paths through
    * the DTNode execute different actions, this method returns
    * a null.  Note that ANodes always return themselves (i.e.
    * there is only one execution path through an ANode).
    */
   ANode      getCommonANode(DTState state);
   /**
    * When optimizing a decision table, we have to be able to combine
    * nodes intelligently. We have to maintain the column numbers, for 
    * one thing.
    * @param _node
    */
   public void addNode(DTNode _node);
   
   /**
    * Does this column have a star?
    * @return
    */
   boolean    getStar();
   
   /**
    * Set the fact that this column has a star.
    * @param f
    */
   void       setStar(boolean f);
}
