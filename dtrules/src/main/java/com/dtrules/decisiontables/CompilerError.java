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

import com.dtrules.session.IDecisionTableError;


public class CompilerError implements IDecisionTableError {
    final private Type   errorType;        //NOPMD Supid PMD wants a set method for a final field... 
    final private String message;          //NOPMD
    final private String source;           //NOPMD Only valid for type ACTION or CONDITION
       
    
    /**
     * Returns the source string which caused the error.
     * @return source 
     */
    public String getSource() {
        return source;
    }
    final private int    index;            //NOPMD Only valid for type INITIALACTION, 
                                           //  ACTION, andCONDITON
    final private int    row;              //NOPMD Only valid for type TABLE
    final private int    col;              //NOPMD Only valid for type TABLE
    
    /**
     * Constructor for INITALACTION, ACTION or CONDITION errors.
     * @param type
     * @param message
     * @param source
     * @param index provided as a zero based number; reported 1 based.
     */
    public CompilerError(
            final Type   type, 
            final String message,
            final String source,
            final int    index){
        errorType    = type;
        this.message = message;
        this.source  = source;
        row          = 0;
        col          = 0;
        this.index = index + 1;
    }
    /**
     * Constructor for TABLE errors.
     * @param type
     * @param message
     * @param row provided as a zero based number; reported 1 based.
     * @param col provided as a zero based number; reported 1 based.
     */
    public CompilerError(
            final Type type,
            final String message,
            final int row,
            final int col){
        errorType      = type;
        this.message   = message;
        this.source    = "";
        this.row       = row+1;
        this.col       = col+1;
        this.index = 0;
    }
    
    
    public int getActionIndex() {
        if(errorType == Type.ACTION){
            return index;
        }
        return 0;
    }

    public int getCol() {
        return col;
    }

    public int getConditionIndex() {
        if(errorType == Type.CONDITION){
            return index;
        }
        return 0;
    }

    public int getIndex() {
        return index;
    }
    
    public Type getErrorType() {
        return errorType;
    }

    public String getMessage() {
        return message;
    }

    public int getRow() {
        return row;
    }
   
}
