/*  
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
 */
package com.dtrules.compiler.el.cup.parser;

import java_cup.runtime.Symbol;

/**
 * @author ps24876
 *
 * We return this symbol only if the identifier is local.
 */
public class RSymbol extends Symbol {
    public boolean local      = false;
    public String  leftvalue;
    public String  rightvalue;
    /**
     * @param id
     * @param l
     * @param r
     * @param o
     */
    public RSymbol(boolean local, Symbol s) {
        super(s.sym);
        this.local       = local;
        this.left        = s.left;
        this.parse_state = s.parse_state;
        this.right       = s.right;
        this.sym         = s.sym;
        this.value       = s.value;
        if(s.value != null) this.rightvalue  = s.value.toString();
    }
}
