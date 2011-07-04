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

package com.dtrules.session;

import java.util.Date;

public interface IDateParser {
	/**
	 * Convert a string to a date.  On failure to do so, a null is returned.
	 * @param s
	 * @return
	 */
	public abstract Date getDate( String s);
	
	public abstract String getDateString(Date date);
	
	public abstract String getTimeString(Date date);

    /**
     * Test the date string against each of the provided allowed Date Formats
     * provided.  If any match, return true, otherwise return false.
     * @param date
     * @param formats
     * @return
     */
    public boolean testFormat(String dateStr, String[] formats);
}