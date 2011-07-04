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

import java.sql.Timestamp;
import java.util.Date;
import java.util.Calendar;
import java.util.GregorianCalendar;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTime;
import com.dtrules.session.DTState;

/**
 * Boolean Operators
 * @author anand b
  */
public class RDateTimeOps {

		static {
			new Newdate();
			new SetCalendar();
			new Yearof();
			new Getdaysinyear();
			new Getdaysinmonth();
			new Getdayofmonth();
			new Datelt();
			new Dategt();
			new Dateeq();
			new Gettimestamp();
            new Days();
            new DatePlus();
            new DateMinus();
            new FirstOfMonth();
            new FirstOfYear();
            new AddYears();
            new AddMonths();
            new AddDays();
            new EndOfMonth();
            new YearsBetween();
            new DaysBetween();
            new MonthsBetween();
            new TestDateFormat();
		}
		
	    /**
	     * Newdate( String -- Date )
	     * Newdate Operator, returns the Date object for the String value.  Returns
	     * a RNull if the string failed to convert to a date.
	     *
	     */
		static class Newdate extends ROperator {
			Newdate(){super("newdate");}

			public void execute(DTState state) throws RulesException {
			    IRObject obj   = state.datapop();
			    String   date  = obj.stringValue();
				try{
					RTime rdate = RTime.getRDate(state.getSession(), date);
					state.datapush(rdate);
				}catch(RulesException e){
					state.datapush(RNull.getRNull());
				}
			}
		} 

        /**
         * Days( number -- Date )
         * Returns a Date object holding this number of days.
         *
         */
        static class Days extends ROperator {
            Days(){super("days");}

            public void execute(DTState state) throws RulesException {
                int  days = state.datapop().intValue();
                long time = days * 24* 60 * 60 * 1000;
                state.datapush(RTime.getRTime(new Date(time)));
            }
        } 

        /**
         * FirstOfMonth ( date -- date2)
         * Given a date, returns date2 pointing to the first of the month.
         * So given 2/23/07 would return 2/1/07
         */
        static class FirstOfMonth extends ROperator {
            FirstOfMonth(){super("firstofmonth");}

            public void execute(DTState state) throws RulesException {
                Date  date = state.datapop().timeValue();
                state.calendar.setTime(date);
                state.calendar.set(Calendar.DAY_OF_MONTH, 1);
                state.calendar.set(Calendar.HOUR, 0);
                state.calendar.set(Calendar.MINUTE, 0);
                state.calendar.set(Calendar.MILLISECOND, 0);  
                state.datapush(RTime.getRTime(state.calendar.getTime()));
            }
        } 
        /**
         * FirstOfYear ( date -- date2)
         * Given a date, returns date2 pointing to the first of the year.
         * So given 2/23/07 would return 2/1/07
         */
        static class FirstOfYear extends ROperator {
            FirstOfYear(){super("firstofyear");}

            public void execute(DTState state) throws RulesException {
                Date  date = state.datapop().timeValue();
                state.calendar.setTime(date);
                state.calendar.set(Calendar.DAY_OF_MONTH, 1);
                state.calendar.set(Calendar.MONTH, 0);
                state.calendar.set(Calendar.HOUR, 0);
                state.calendar.set(Calendar.MINUTE, 0);
                state.calendar.set(Calendar.MILLISECOND, 0);  
                state.datapush(RTime.getRTime(state.calendar.getTime()));
            }
        } 
        /**
         * EndOfMonth ( date -- date2)
         * Given a date, returns date2 pointing to the first of the month.
         * So given 2/23/07 would return 2/1/07
         */
        static class EndOfMonth extends ROperator {
            EndOfMonth(){super("endofmonth");}

            public void execute(DTState state) throws RulesException {
                Date  date = state.datapop().timeValue();
                state.calendar.setTime(date);
                int maxdays = state.calendar.getActualMaximum(Calendar.DAY_OF_MONTH); 
                state.calendar.set( Calendar.DAY_OF_MONTH, maxdays );
                state.calendar.set(Calendar.HOUR, 0);
                state.calendar.set(Calendar.MINUTE, 0);
                state.calendar.set(Calendar.MILLISECOND, 0);  
                Date result = state.calendar.getTime();
                state.datapush(RTime.getRTime(result));
            }
        } 
        /**
         * AddYears ( date int -- date2)
         * Adds the given number of years to the given date.  Care has to be 
         * taken where leap years are in effect. The month can change.
         * So given 2/23/07  3 addYears would return 2/23/10
         */
        static class AddYears extends ROperator {
            AddYears(){super("addyears");}

            public void execute(DTState state) throws RulesException {
                int   years = state.datapop().intValue();
                Date  date  = state.datapop().timeValue();
                state.calendar.setTime(date);
                state.calendar.add(Calendar.YEAR, years);
                state.datapush(RTime.getRTime(state.calendar.getTime()));
            }
        } 

        /**
         * AddMonths ( date int -- date2)
         * Adds the given number of months to the given date.  Care has to be 
         * taken where the current day (31) may not be present in the new 
         * month (a month with 30 days). 
         * The behavior in this case is defined by the behavior of the Java
         * Calendar.
         * So given 2/23/07  3 addMonths would return 5/23/07
         */
        static class AddMonths extends ROperator {
            AddMonths(){super("addmonths");}

            public void execute(DTState state) throws RulesException {
                int   months = state.datapop().intValue();
                Date  date   = state.datapop().timeValue();
                state.calendar.setTime(date);
                state.calendar.add(Calendar.MONTH, months);
                Date  newdate = state.calendar.getTime();
                state.datapush(RTime.getRTime(newdate));
            }
        } 

        /**
         * AddDays ( date int -- date2)
         * Adds the given number of days to the given date.  You might 
         * move over to the next month!
         */
        static class AddDays extends ROperator {
            AddDays(){super("adddays");}

            public void execute(DTState state) throws RulesException {
                int   days  = state.datapop().intValue();
                Date  date  = state.datapop().timeValue();
                state.calendar.setTime(date);
                state.calendar.add(Calendar.DATE, days);
                state.datapush(RTime.getRTime(state.calendar.getTime()));
            }
        } 

        /**
	     * SetCalendar( String --  )
	     * SetCalendar Operator, 
	     */
        @SuppressWarnings({"unchecked"})
		static class SetCalendar extends ROperator {
			SetCalendar(){super("setCalendar");}

			public void execute(DTState state) throws RulesException {
				try {
					Class clazz = Class.forName(state.datapop().stringValue());
					Object obj = clazz.newInstance();
					if(obj instanceof Calendar){
						state.calendar=(Calendar)obj;
					} else {
						throw new RulesException("Date Time Exception","Set Calendar","Not a Calendar Object");
					}					
				} catch(Exception e){
					throw new RulesException("Date Time Exception","/","Error while creating object: "+e);
				}
			}
		}		

	    /**
	     * Yearof( date -- int )
	     * Yearof Operator, returns the year value for the given date
	     */
		static class Yearof extends ROperator {
			Yearof(){super("yearof");}

			public void execute(DTState state) throws RulesException {
				Calendar calendar = Calendar.getInstance();
				calendar.setTime(state.datapop().timeValue());
				state.datapush(RInteger.getRIntegerValue(calendar.get(Calendar.YEAR)));
			}
		}		

	    /**
	     * Getdaysinyear( date -- long )
	     * Getdaysinyear Operator, returns the number of days in a year from the given date
	     */
		static class Getdaysinyear extends ROperator {
			Getdaysinyear(){super("getdaysinyear");}

			public void execute(DTState state) throws RulesException {
				GregorianCalendar calendar = new GregorianCalendar();
				calendar.setTime(state.datapop().timeValue());
				if(calendar.isLeapYear(calendar.get(Calendar.YEAR))){
					state.datapush(RInteger.getRIntegerValue(366));
				} else {
					state.datapush(RInteger.getRIntegerValue(365));
				}
			}
		}		
	    /**
	     * Getdayofmonth( date -- long )
	     * Returns the day of the month in the given date
	     */
		static class Getdayofmonth extends ROperator {
			Getdayofmonth(){super("getdayofmonth");}

			public void execute(DTState state) throws RulesException {
				GregorianCalendar calendar = new GregorianCalendar();
				calendar.setTime(state.datapop().timeValue());
				int day = calendar.get(Calendar.DAY_OF_MONTH);
				state.datapush(RInteger.getRIntegerValue(day));
			}
		}		
	    /**
	     * GetdaysinMonth( date -- long )
	     * Getdaysinyear Operator, returns the number of days in a year from the given date
	     */
		static class Getdaysinmonth extends ROperator {
			Getdaysinmonth(){super("getdaysinmonth");}

			public void execute(DTState state) throws RulesException {
				GregorianCalendar calendar = new GregorianCalendar();
				calendar.setTime(state.datapop().timeValue());
				int daysinmonth = calendar.getActualMaximum(Calendar.DAY_OF_MONTH);
			    state.datapush(RInteger.getRIntegerValue(daysinmonth));
			}
		}		

	    /**
	     * Datelt( date1 date2 -- boolean )
	     * Datelt Operator, returns true if date1 is less than date2
	     */
		static class Datelt extends ROperator {
			Datelt(){super("d<");}

			public void execute(DTState state) throws RulesException {
				Date date2 = state.datapop().timeValue();
				Date date1 = state.datapop().timeValue();
                boolean test =date1.before(date2);
				state.datapush(RBoolean.getRBoolean(test));
			}
		}				

	    /**
	     * Dategt( date1 date2 -- boolean )
	     * Dategt Operator, returns true if date1 is greater than date2
	     */
		static class Dategt extends ROperator {
			Dategt(){super("d>");}

			public void execute(DTState state) throws RulesException {
				IRObject o2 = state.datapop();
				IRObject o1 = state.datapop();
				Date date1=null, date2=null;
				try{
				  date1 = o1.timeValue();  
				  date2 = o2.timeValue();
				}catch(RulesException e){
				  if(date1==null) {
				      e.addToMessage("The First Parameter is null");
    				  try{
    				      date2 = o2.timeValue();
    				  }catch(RulesException e2){
    				      e.addToMessage("The Second Parameter is also null");    
    				  }
				  }else{
				      e.addToMessage("The Second Parameter is null");
				  }
				  throw e;
				}
                boolean test = date1.after(date2);
				state.datapush(RBoolean.getRBoolean(test));
			}
		}

	    /**
	     * Dateeq (date1 date2 -- boolean )
	     * Dateeq Operator, returns true if date1 is equals to date2
	     */
		static class Dateeq extends ROperator {
			Dateeq(){super("d==");}

			public void execute(DTState state) throws RulesException {
				Date date2 = state.datapop().timeValue();
				Date date1 = state.datapop().timeValue();
				state.datapush(RBoolean.getRBoolean(date1.compareTo(date2)==0));
			}
		}

	    /**
	     * Gettimestamp( date -- String )
	     * Gettimestamp Operator, creates a string timestamp from s date
	     */
		static class Gettimestamp extends ROperator {
			Gettimestamp(){super("gettimestamp");}

			public void execute(DTState state) throws RulesException {
				Date date = state.datapop().timeValue();
				state.datapush(RString.newRString((new Timestamp(date.getTime())).toString()));
			}
		}		
        
        /**
         * d+ ( date1 date2 -- )
         * Add two dates together.  This doesn't make all that much sense unless
         * one or both of the dates is just a count of days.
         */
        static class DatePlus extends ROperator {
            DatePlus() {super("d+"); }
            public void execute(DTState state) throws RulesException {
                long date2 = state.datapop().timeValue().getTime();
                long date1 = state.datapop().timeValue().getTime();
                state.datapush(RTime.getRTime(new Date(date1+date2)));
            }
            
        }
        
        /**
         * d- ( date1 date2 -- )
         * Subtract date2 from date1  This doesn't make all that much sense unless
         * one or both of the dates is just a count of days.
         */
        static class DateMinus extends ROperator {
            DateMinus() {super("d-"); }
            public void execute(DTState state) throws RulesException {
                long date2 = state.datapop().timeValue().getTime();
                long date1 = state.datapop().timeValue().getTime();
                state.datapush(RTime.getRTime(new Date(date1-date2)));
            }
            
        }
        /**
         * ( date1 date2 -- int )
         * Returns the number of years between date1 and date2.  It is always
         * the difference.  the value returned is negative if date1 is before date2.
         */
        static class YearsBetween extends ROperator {
            YearsBetween() {super("yearsbetween"); }
            public void execute(DTState state) throws RulesException {
                Date date2 = state.datapop().timeValue();
                Date date1 = state.datapop().timeValue();
                boolean swapped = false; 
                if(date1.after(date2)){
                    swapped = true;
                    Date hold = date1;
                    date1 = date2;
                    date2 = hold;
                }
                state.calendar.setTime(date1);
                int y1 = state.calendar.get(Calendar.YEAR);
                int m1 = state.calendar.get(Calendar.MONTH);
                int d1 = state.calendar.get(Calendar.DAY_OF_MONTH);
                state.calendar.setTime(date2);
                int y2 = state.calendar.get(Calendar.YEAR);
                int m2 = state.calendar.get(Calendar.MONTH);
                int d2 = state.calendar.get(Calendar.DAY_OF_MONTH);
                int diff = y2-y1;
                if(m2<m1)diff--;
                if(m2==m1 && d2<d1)diff--;
                if(swapped)diff *= -1;
                state.datapush(RInteger.getRIntegerValue(diff));
            }
            
        }
        /**
         * ( date1 date2 -- int )
         * Returns the number of months between date1 and date2.  If date1
         * is after date2, the number returned is negative.
         */
        static class MonthsBetween extends ROperator {
            MonthsBetween() {super("monthsbetween"); }
            public void execute(DTState state) throws RulesException {
                Date date2 = state.datapop().timeValue();
                Date date1 = state.datapop().timeValue();
                boolean swapped = false;
                if(date1.after(date2)){
                    swapped = true;
                    Date hold = date1;
                    date1 = date2;
                    date2 = hold;
                }
                state.calendar.setTime(date1);
                int y1 = state.calendar.get(Calendar.YEAR);
                int m1 = state.calendar.get(Calendar.MONTH);
                int d1 = state.calendar.get(Calendar.DAY_OF_MONTH);
                state.calendar.setTime(date2);
                int y2 = state.calendar.get(Calendar.YEAR);
                int m2 = state.calendar.get(Calendar.MONTH);
                int d2 = state.calendar.get(Calendar.DAY_OF_MONTH);
                int yeardiff = y2-y1;
                if(m2<m1)yeardiff--;
                int monthdiff = m2-m1;
                if(d2<d1-1)monthdiff--;
                if(monthdiff < 0)monthdiff +=12;
                monthdiff += 12*yeardiff;
                if(swapped)monthdiff *= -1;
                state.datapush(RInteger.getRIntegerValue(monthdiff));
             }
          }  
        
          /**
           * (date dateString dateRegex SeparatorRegex --> boolean )
           * Returns true if the date provided strictly matches the dateString, 
           * and the dateRegex.  It does this by breaking out the day and month
           * from the dateString match the day and month returned by the calendar
           * for that date.
           *  
           * @author Paul Snow
           *
           */
          static class TestDateFormat extends ROperator {
        	  TestDateFormat() {super("testdateformat"); }
              public void execute(DTState state) throws RulesException {
                  String sepRegex  = state.datapop().stringValue();
                  String dateRegex = state.datapop().stringValue();
                  String dateStr   = state.datapop().stringValue();
                  Date   date      = state.datapop().timeValue();
                  boolean result = state.getSession().getDateParser().testFormat(
                		  date, dateStr, dateRegex, sepRegex);
                  state.datapush(RBoolean.getRBoolean(result));
              }
          }
 
          /**
           * (date1 date2 --> int )
           * Returns the days between two dates.  This is the difference between
           * the dates, and is negative if date1 is before date2.
           * @author Paul Snow
           *
           */
          static class DaysBetween extends ROperator {
        	  DaysBetween() {super("daysbetween"); }
              public void execute(DTState state) throws RulesException {
                  Date date2 = state.datapop().timeValue();
                  Date date1 = state.datapop().timeValue();
                  boolean swapped = false;
                  if(date1.after(date2)){
                      swapped = true;
                      Date hold = date1;
                      date1 = date2;
                      date2 = hold;
                  }
                  state.calendar.setTime(date1);
                  long from = state.calendar.getTimeInMillis();
                  state.calendar.setTime(date2);
                  long to   = state.calendar.getTimeInMillis();
                  long days = Math.round((to-from)/(1000*60*60*24));
                  if (swapped) days *= -1;
                  state.datapush(RInteger.getRIntegerValue(days));
              }
          }
 
}