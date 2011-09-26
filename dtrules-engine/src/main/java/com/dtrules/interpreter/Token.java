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

package com.dtrules.interpreter;

class Token {
    
    public enum Type {
        INT, 
        REAL, 
        DATE,
        LPAREN, 
        RPAREN, 
        LCURLY, 
        RCURLY, 
        LSQUARE, 
        RSQUARE, 
        NAME, 
        STRING,
    };
    
    Type   type;
    String strValue;
    Long   longValue;
    Double doubleValue;
    RDate  datevalue;
    RName  nameValue;
    
    public Type getType(){
        return type;
    }
    
    Token(char c) {
        strValue = String.valueOf(c);
        switch(c){
            case '[': type = Type.LSQUARE; break;
            case ']': type = Type.RSQUARE; break;
            case '{': type = Type.LCURLY;  break;
            case '}': type = Type.RCURLY;  break;
            case '(': type = Type.LPAREN;  break;
            case ')': type = Type.RPAREN;  break;
            default :
                nameValue = RName.getRName(String.valueOf(c));
                type  = Type.NAME;
        }
    }
    
    Object getValue(){
        if(type == Type.INT)return longValue;
        if(type == Type.REAL)return doubleValue;
        if(type == Type.STRING)return strValue;
        if(type == Type.LSQUARE)return "[";
        if(type == Type.RSQUARE)return "]";
        if(type == Type.LCURLY)return "{";
        if(type == Type.RCURLY)return "}";
        if(type == Type.LPAREN)return "(";
        if(type == Type.RPAREN)return ")";
        if(type == Type.NAME)return "/"+nameValue.stringValue();
        return "?";
    }
    
    Token(RDate t){
        datevalue = t;
        type = Type.DATE;
    }
    
    Token(String s,Type t){
        strValue = s;
        type  = t;
    }
    Token(long v){
        longValue = v;
        type  = Type.INT;
    }
    
    Token(Double v){
        doubleValue = v;
        type  = Type.REAL;
    }
    
    Token(RName n){
        nameValue = n;
        type  = Type.NAME;
    }
}
