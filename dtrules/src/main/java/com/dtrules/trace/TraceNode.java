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

package com.dtrules.trace;

import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;

/**
 * A trace node allows two kinds of navigation acorss changes recorded in
 * a DTRules Trace file.   One mode is a tree, and one mode is linear.  In 
 * fact, all changes occur serially within a decision table thread.  Yet all
 * changes occur with in a hierarchy of Decision Table execution.
 * 
 * The parent of all nodes is a root node. The initialization node holds the
 * initial values (includeing data inserted into a ruleset)
 * 
 * The changes are kept as nodes 
 */
public interface TraceNode {
    
    /**
     * These are the hierarchy operators.
     */
    
    /** getParent() -- return the parent node, usually a decisiontable.  But it
     * could be the root node, or the initialization node.
     * @return
     */
    TraceNode getParent();
    /**
     * Get the firstChild()
     * @return
     */
    TraceNode getFirstChild();
    /**
     * Get the next();  
     * Returns a null if no more children;
     * @return
     */
    TraceNode getNext();
    /**
     * Get the perivious node, 
     * @return
     */
    TraceNode getPrevious();
   
    /**
     * These are the linear access methods.  All the changes are organized
     * in a linear list.
     */
    /**
     * Get the next Change Node
     * @return
     */
    TraceNode getNextChange();
    /**
     * Get the previous Change Node
     * @return
     */
    TraceNode getPreviousChange();
    
    
    /**
     * Change Operators
     */
    void setOldValues();
    void setNewValues();
    /**
     * Returns the Name for the Attribute that changed (or a null if 
     * this isn't a change to an Entity).
     */
    RName getAttribute();
    /**
     * Returns the index for the Attribute that changed (or a null if
     * this isn't a change to an Element of an Array).
     */
    RInteger getIndex();
}
