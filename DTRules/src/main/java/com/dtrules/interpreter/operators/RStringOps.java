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
  
package com.dtrules.interpreter.operators;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RString;
import com.dtrules.session.DTState;

public class RStringOps {
    static {
        new Stringlength(); new Touppercase();  new Tolowercase();  new Trim();
        new substring();    new RegexMatch();   new Indexof();
    }


    /**
     * ( string -- length ) returns the length of the given string
     * @author paul snow
     *
     */
    static class Stringlength extends ROperator {
        Stringlength(){super("strlength");}

        @Override
        public void execute(DTState state) throws RulesException {
            String   str = state.datapop().stringValue();
            RInteger len = RInteger.getRIntegerValue(str.length());
            state.datapush(len);
        }
    }

    /**
     * ( String -- String ) converts the string to uppercase.
     * @author paul snow
     *
     */
    static class Touppercase extends ROperator {
        Touppercase(){super("touppercase");}

        @Override
        public void execute(DTState state) throws RulesException {
            String   str  = state.datapop().stringValue();
            String   str2 = str.toUpperCase();
            state.datapush(RString.newRString(str2));
        }
    }

    /**
     * ( String -- String ) converts the string to lowercase
     * @author paul snow
     *
     */
    static class Tolowercase extends ROperator {
        Tolowercase(){super("tolowercase");}

        @Override
        public void execute(DTState state) throws RulesException {
            String   str  = state.datapop().stringValue();
            String   str2 = str.toLowerCase();
            state.datapush(RString.newRString(str2));
        }
    }

    /**
     * ( string -- string ) Trims the string, per trim in Java
     * @author paul snow
     *
     */
    static class Trim extends ROperator {
        Trim(){super("trim");}

        @Override
        public void execute(DTState state) throws RulesException {
            String   str  = state.datapop().stringValue();
            String   str2 = str.trim();
            state.datapush(RString.newRString(str2));
         }
    }

    /**
     * (endindex beginindex string -- substring ) Returns the substring
     * @author paul snow
     *
     */
    static class substring extends ROperator {
        substring(){super("substring");}

        @Override
        public void execute(DTState state) throws RulesException {
            String   str  = state.datapop().stringValue();
            int      b    = state.datapop().intValue();
            int      e    = state.datapop().intValue();
            String   str2 = str.substring(b,e);
            state.datapush(RString.newRString(str2));
         }
    }

    /**
     * ( regex String -- boolean ) Returns true if the given string is matched
     * by the given regular expression.
     * 
     * @author Paul Snow
     *
     */
    static class  RegexMatch   extends ROperator {
        RegexMatch(){super("regexmatch");}
        
        @Override
        public void execute(DTState state) throws RulesException {
            String string = state.datapop().stringValue();
            String regex  = state.datapop().stringValue();
            boolean b = string.matches(regex);
            state.datapush( RBoolean.getRBoolean(b));
        }
    }
    
    /**
     * ( String1 String2 -- int ) Returns the index of string1 in string2
     * 
     * @author Paul Snow
     *
     */
    static class  Indexof   extends ROperator {
        Indexof(){super("indexof");}
        
        @Override
        public void execute(DTState state) throws RulesException {
            String string2 = state.datapop().stringValue();
            String string1 = state.datapop().stringValue();
            int index = string2.indexOf(string1);
            state.datapush( RInteger.getRIntegerValue(index));
        }
    }
    
}
