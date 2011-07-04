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
 
  
package com.dtrules.interpreter.operators;

import java.util.Iterator;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

public class OperatorTest {

    static String tests[] = {
 
        " 1.2 2.3 f+               ", "3.5",
        
        " 1.2 2.3 +                ", "3",
        " 2.5 6   f*               ", "15.0",
        " 2.5 6   *                ", "12",
        " mark 1 2 3 4 arraytomark ", "[ 1  2  3  4  ]",
        " 1 2 drop                 ", "1",
        " 1 2 pop                  ", "1",
        " true                     ", "true",
        " false                    ", "false",
        " 1 2 3 pstack print  pop  ", "1",
        " 'TypeCheck' 'error test' error", "",
        " 'TypeCheck'              ", "TypeCheck",
        " \"TypeCheck\"            ", "TypeCheck",        
        " newarray                 ", "[ ]",        
        " newarray dup 2 addto     ", "[ 2  ]",
        " [ 2  ] dup 1 3 addat ",  "[ 2  3  ]",
        " [ 2 3 4 3 ] dup 3 remove ", "[ 2  4  ]",
        " [ 1 2 3 4 ] dup 3 removeat ", "[ 1  2  3  ]",
        " [ 3 4 5 6 ] dup 0 getat  ", "3",
        " [ 3 2 6 6 ] dup length   ", "4",
        " [ 5 7 8 1 ] dup 8 memberof ", "true",
        " [ 5 7 8 1 ] dup 9 memberof ", "false",
        " [ 1  2  3  4 ] copyelements ", "[ 1  2  3  4  ]",
        " [ 5 7 8 1 ] dup 9 add_no_dups ", "[ 5  7  8  1  9  ]",
        " [ 5 7 8 1 ] dup 8 add_no_dups ", "[ 5  7  8  1  ]",
        " [ 5 7 8 1 ] dup clear ", "[ ]",
        " [ 1 2 ] [ 3 4 ] merge ", "[ 1  2  3  4  ]",
        
        " false not                    ", "true",
        " false false &&               ", "false",
        " false true ||                ", "true",
        " 5 3 >                        ", "true",
        " 5 3 <                        ", "false",
        " 3 3 >=                       ", "true",
        " 3 8 <=                       ", "true",
        " 1 1 ==                       ", "true",
        " 5.1 3.2 >                    ", "true",
        " 5.5 3.4 <                    ", "false",
        " 3.1 3.0 >=                   ", "true",
        " 3.9 4.0 <=                   ", "true",
        " 1.5 1.5 ==                   ", "true",
        " true true b=                 ", "true",
        " true true b!=                ", "false",
        " 'austin' 'austin' s==        ", "true",
        " 'madison' 'austin' s>        ", "true",        
        " 'dallas' 'austin' s<         ", "false",
        " 3 3 >=                       ", "true",
        " 3 8 <=                       ", "true",
        " 1 1 ==                       ", "true", 
        " 'austin ' 'downtown' s+      ", "austin downtown",
        " 'austin downtown ' 'downtown' strremove  ", "austin",    
        " 'abc' 'abc' req               ", "true",
        " 5 5 req                       ", "true",
        
        " '2007-12-12' newdate          ", "2007-12-12",
        " '2007-12-12' newdate yearof         ", "2007",
        " '2007-12-12' newdate getdaysinyear         ", "365",
        " '2008-12-12' newdate getdaysinyear         ", "366",
        " '2008-12-12' newdate '2007-12-12' newdate d< ", "false",
        " '2008-12-12' newdate '2007-12-12' newdate d> ", "true",
        " '2008-12-12' newdate '2008-12-11' newdate d== ", "false",
        " '2008-12-12' newdate '2008-12-12' newdate d== ", "true",
        " '2008-12-12' newdate gettimestamp", "2008-12-12 00:00:00.0",
        
    };
    
    static int testcnt         = 0;
    static int testcntfailed   = 0;
        
    public static void main(String args[]){ 
        
        String          file    = "c:\\eclipse\\workspace\\DTRules\\com.dtrules.testfiles\\DTRules.xml";
        IRSession       session;
        DTState         state;
        RulesDirectory  rd;
        RuleSet         rs;
        
        try {
            rd      = new RulesDirectory("",file);
            rs      = rd.getRuleSet(RName.getRName("test",true));
            session = rs.newSession();
            state   = session.getState();
        } catch (RulesException e1) {
            System.out.println("Failed to initialize the Rules Engine");
            return;
        }
        
        for(int i=0;i<tests.length;i+=2){
            try{
               session.execute(tests[i]);
               String result = state.datapop().stringValue();
               if(tests[i+1].equals(result.trim())){
                   state.debug("test: << "+tests[i]+" >> expected: "+tests[i+1]+" --passed\n");
               }else{
                   state.debug("test: << "+tests[i]+" >> expected: "+tests[i+1]+" result:"+result+" --FAILED\n");
               }
               state.debug("\n");
            }catch(Exception e){
               state.error(" Exception Thrown:\n");
               state.error("test: "+tests[i]+"expected: "+tests[i+1]+" result: Exception thrown --FAILED\n");
               state.error(e+"\n");
            }
        }    
      }
}
