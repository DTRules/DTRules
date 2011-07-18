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

import java.util.ArrayList;

import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;

public class DecisionTableNode implements TraceNode {
    
    ArrayList<TraceNode> children = new ArrayList<TraceNode>();
    TraceNode            parent;
    TraceNode            previous;
    
    public RName getAttribute() {
        return null;
    }
    /**
     * Get first child, and return null if no children.
     */
    public TraceNode getFirstChild() {
        if(children.size()==0)return null;
        return children.get(0);
    }
    
    public RInteger getIndex() {
        return null;
    }

    public TraceNode getNext() {
        // TODO Auto-generated method stub
        return null;
    }

    public TraceNode getNextChange() {
        // TODO Auto-generated method stub
        return null;
    }

    public TraceNode getParent() {
        // TODO Auto-generated method stub
        return null;
    }

    public TraceNode getPrevious() {
        // TODO Auto-generated method stub
        return null;
    }

    public TraceNode getPreviousChange() {
        // TODO Auto-generated method stub
        return null;
    }

    public void setNewValues() {
        // TODO Auto-generated method stub

    }

    public void setOldValues() {
        // TODO Auto-generated method stub

    }

}
