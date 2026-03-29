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

import java.util.Date;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.IRSession;

public class SimpleTokenizer {
    private StringBuffer 	buff;
    private int    			index;
    private final IRSession session;
    
    public SimpleTokenizer (IRSession session, String input){
    	this.session = session;
        buff  = new StringBuffer(input);
        index = 0;
    }
    /**
     * Tosses whitespace (anything less than 32).  Returns true if any 
     * whitespace was found, and false otherwise.  The end of string counts
     * as whitespace.
     * @return
     */
    private boolean tossWhite(){
        if(index>=buff.length())return true;
        char c = buff.charAt(index);
        if(c>' ')return false;
        while(hasChar() && buff.charAt(index)<=32)index++;
        return true;
    }
    /**
     * Returns true if any characters remain in the buffer.
     * @return
     */
    private boolean hasChar(){
        return index < buff.length();
    }
    /**
     * Returns the next character in the buffer. Returns a space on end
     * of string.
     * @return
     */
    char getChar(){
        if(index>=buff.length())return ' ';
        char c = buff.charAt(index++);
        return c;
    }
    
    /**
     * Parses out a token at a time, and returns null on the end of input.
     * @return
     */
    public Token nextToken() throws RulesException {
        tossWhite();
        if(!hasChar())return null;
        int start = index;
        char c = getChar();
        switch(c){
            case '"': case '\'': return parseString(c);
            case '[': 
            case ']':
            case '{':
            case '}':
            case '(':
            case ')': return new Token(c);
        }        
        while(getChar()>32);
        return buildToken(buff.substring(start, index));
    }
    /**
     * We assume the first quote has been
     * parsed already.  We parse till we find the delim char.
     * We then build  
     * @param delim
     * @return
     */    
    Token parseString(char delim){
        char c = ' ';
        int start = index;
        while(hasChar()){
            if(getChar()==delim){
                if(start>index-1){
                    return new Token("",Token.Type.STRING);
                }
                return new Token(buff.substring(start,index-1),Token.Type.STRING);
            }
        }    
        throw new RuntimeException("String is missing the closing quote: ("+c+")");
    }
    /**
     * We attempt to convert v to a long.
     * Then we attempt to convert v to a float.
     * Then we try parsing it as a date.
     * Then we build a name.
     * @param v
     * @return
     */
    Token buildToken(String v){
        String vnum = v.replaceAll("[,\\s]","");
        try {
            Long longValue = Long.parseLong(vnum);
            return new Token(longValue);
        } catch (NumberFormatException e) {}
        try {
            Double doubleValue = Double.parseDouble(vnum);
            return new Token(doubleValue);
        } catch (NumberFormatException e) {}
        Date t = session.getDateParser().getDate(v);
        if(t!=null){
           return new Token(RDate.getRTime(t));
        } 
        
        return new Token(RName.getRName(v));
    }
    
}
