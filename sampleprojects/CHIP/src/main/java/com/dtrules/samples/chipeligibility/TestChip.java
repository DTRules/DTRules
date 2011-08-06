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
package com.dtrules.samples.chipeligibility;


import java.util.Date;

import com.dtrules.testsupport.ATestHarness;

public class TestChip extends ATestHarness {
    	
    public static String path    = System.getProperty("user.dir")+"/";
	static  Date start = new Date();
    
    public static void main(String[] args) throws Exception {
    
    	TestChip t = new TestChip();		// Create an instance of the test harness.
    	t.load(path+"/xml/testParms.xml");  // Load the settings for this test.
    	t.runTests();						// Run the tests.
    
    }
}    
	    
