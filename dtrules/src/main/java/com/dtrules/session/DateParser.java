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

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class DateParser implements IDateParser {

    public  Pattern [] patterns ={
        Pattern.compile("\\d{1,2}/\\d{1,2}/\\d{4} \\d\\d:\\d\\d:\\d\\d:\\d\\d\\d\\d"),
        Pattern.compile("\\d{1,2}/\\d{1,2}/\\d{4}"),
        Pattern.compile("[01]?\\d-[0123]?\\d-\\d\\d\\d\\d"),
        Pattern.compile("\\d\\d\\d\\d/[01]?\\d/[0123]?\\d"),
        Pattern.compile("\\d\\d\\d\\d-[01]?\\d-[0123]?\\d"),
    };
    
    public  SimpleDateFormat [] formats ={
            new SimpleDateFormat("MM/dd/yyyy HH:mm:ss:SSSS"),
            new SimpleDateFormat("MM/dd/yyyy"),
            new SimpleDateFormat("MM-dd-yyyy"),
            new SimpleDateFormat("yyyy/MM/dd"),
            new SimpleDateFormat("yyyy-MM-dd"),
        };
    
    SimpleDateFormat dateStringFormat = new SimpleDateFormat("MM/dd/yyyy");
    SimpleDateFormat timeStringFormat = new SimpleDateFormat("MM/dd/yyyy HH:mm:ss:SSSS");
    
    
    /**
     * Attempts to convert the string to a date.  If it fails, it returns a 
     * null.
     */
    public  Date getDate( String s){
    	s = s.trim();
        for(int i=0;i<patterns.length;i++){
            Matcher matcher = patterns[i].matcher(s);
            if(matcher.matches()){
                try {
                    Date d = formats[i].parse(s);
                    return d;
                } catch (ParseException e) { } // Didn't work? just try again
            }
        }
        return null;   
    }
    
    public String getDateString(Date date){
    	return dateStringFormat.format(date);
    }
    
    public String getTimeString(Date date){
    	return timeStringFormat.format(date);
    }
    
    /**
     * Java is much more flexible in what it allows than what data entry necessarily
     * needs to test. So this routine turns off this flexiblity when checking the format
     * of date strings. 
     * @param date
     * @return
     */
    public boolean testFormat(
		String date, String[] formats){
    	for(String format : formats){
    		try{
    			SimpleDateFormat sdf = new SimpleDateFormat(format);
    			sdf.setLenient(false);
    			Date d = sdf.parse(date.trim());
    			if(d != null )return true;
    		}catch(ParseException e) {}
    	}
		return false;
	}
}
