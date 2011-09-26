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

import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;

public class RDate extends ARObject {
	
	static RType type = RType.newType("date");

    final Date time;
	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}

    
    @Override
    public double doubleValue() throws RulesException {
        return time.getTime();
    }

    @Override
    public long longValue() throws RulesException {
        return  time.getTime();
    }

    @Override
    public RDouble rDoubleValue() throws RulesException {
        return  RDouble.getRDoubleValue(time.getTime());
    }


    @Override
    public RInteger rIntegerValue() throws RulesException {
        return  RInteger.getRIntegerValue(time.getTime());
    }

    @Override
    public RString rStringValue() {
        return RString.newRString(stringValue());
    }

    private RDate(Date t){
        time = t;
    }
   
    
    /**
     * Returns a Null if no valid RDate can be parsed from
     * the string representation
     * @param s
     * @return
     */
    public static RDate getRDate(IRSession session, String s) throws RulesException {
        Date d = session.getDateParser().getDate(s);
        if(d==null){
            throw new RulesException("Bad Date Format","getRDate","Could not parse: '"+s+"' as a Date or Time value");
        }
        return new RDate(d);
    }
    
    public static RDate getRTime(Date t){
        return new RDate(t);
    }

    @Override
    public String toString(){
        SimpleDateFormat f = new SimpleDateFormat("MM/dd/yyyy HH:mm:ss:SSSS");
        return f.format(time);
    }
    
    public String stringValue() {
        return toString();
    }

    @Override
    public String postFix() {
    	return "\""+toString()+"\" cvd";
    }
    
    public int getYear(DTState state) throws RulesException{
        if(time==null){
            throw new RulesException("Undefined", "getYear()", "No valid date available");
        }
        Calendar c = Calendar.getInstance();
        c.setTime(time);
        return c.get(Calendar.YEAR);
    }
    /**
     * Returns self.
     */
    @Override
    public RDate rTimeValue() throws RulesException {
        return this;
    }

    /**
     * Returns the time's date object.
     */
    @Override
    public Date timeValue() throws RulesException {
        return time;
    }

    /**
     * returns 0 if both are equal. -1 if this object is less than the argument. 
     * 1 if this object is greater than the argument
     */
    @Override
    public int compare(IRObject irObject) throws RulesException {
    	return this.time.compareTo(irObject.timeValue());
    }

	@Override
	public boolean equals(IRObject o) throws RulesException {
		return time.equals(o.timeValue());
	}
    
    
}
