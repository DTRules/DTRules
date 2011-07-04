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

package com.dtrules.automapping;

/**
 * This enum defines the Java Types that we can handle.  If it
 * returns INVALID, then this is not a type we know how to 
 * handle, and (if possible) it should be ignored.  
 *
 * @author Paul Snow
 *
 */
public enum MapType {
    NULL {
        public boolean isPrimitive(){return true;};
        public String getName(){return "null"; }
    },
    INT {
        public boolean isPrimitive(){return true;};
        public String getName(){return "int"; }
    },
    LONG{
        public boolean isPrimitive(){return true;};
        public String getName(){return "long"; }
    },
    SHORT{
        public boolean isPrimitive(){return true;};
        public String getName(){return "short"; }
    },
    DOUBLE{
        public boolean isPrimitive(){return true;};
        public String getName(){return "double"; }
    },
    STRING{
        public boolean isPrimitive(){return true;};
        public String getName(){return "string"; }
    },
    DATE{
        public boolean isPrimitive(){return true;};
        public String getName(){return "date"; }
    },
    BOOLEAN{
        public boolean isPrimitive(){return true;};
        public String getName(){return "boolean"; }
    },
    LIST{
        public boolean isPrimitive(){return false;};
        public String getName(){return "list"; }
    },
    MAP{
        public boolean isPrimitive(){return false;};
        public String getName(){return "map"; }
    },
    OBJECT{
        public boolean isPrimitive(){return false;};
        public String getName(){return "object"; }
    };
    
    public abstract boolean isPrimitive();
    public abstract String  getName();
    
    public static MapType get(String s){
        if(s == null ){
            return NULL;
        }
    	if(s.indexOf(".")>=0){
    		s = s.substring(s.lastIndexOf(".")+1);
    	}
        s = s.trim();
        if(s.length()==0){
            return NULL;
        }else if(s.equalsIgnoreCase("integer")){
            return INT;
        }else if(s.equalsIgnoreCase("int")){
            return INT;
        }else if (s.equalsIgnoreCase("long")){
            return LONG;
        }else if (s.equalsIgnoreCase("short")){
            return SHORT;
        }else if (s.equalsIgnoreCase("double")){
            return DOUBLE;
        }else if (s.equalsIgnoreCase("string")){
            return STRING;
        }else if (s.equalsIgnoreCase("date")){
            return DATE;
        }else if (s.equalsIgnoreCase("time")){
            return DATE;
        }else if (s.equalsIgnoreCase("integer")){
            return INT;
        }else if (s.equalsIgnoreCase("boolean")){
            return BOOLEAN;
        }else if (s.equalsIgnoreCase("list")){
            return LIST;
        }else if (s.equalsIgnoreCase("array")){
            return LIST;
        }else if (s.equalsIgnoreCase("HashMap")){
            return MAP;
        }else if (s.equalsIgnoreCase("Map")){
            return MAP;
        }
        
        return OBJECT;
    }
}
