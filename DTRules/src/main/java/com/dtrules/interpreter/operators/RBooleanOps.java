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

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RString;
import com.dtrules.session.DTState;

/**
 * Boolean Operators
 * @author anand b
  */
public class RBooleanOps {

		static {
            ROperator.alias(RBoolean.getRBoolean(true),"true");
            ROperator.alias(RBoolean.getRBoolean(false),"false");
            new Not();
			new And();
			new Or();
			new Greaterthan();
			new Lessthan();
			new Greaterthanequal();
			new Lessthanequal();
			new Equal();
			new FGreaterthan();
			new FLessthan();
			new FGreaterthanequal();
			new FLessthanequal();
			new FEqual();			
			new Isnull();
			new Booleanequal();
			new Booleannotequal();
			new SGreaterthan();
			new SLessthan();
			new SGreaterthanequal();
			new SLessthanequal();
			new SEqual();	
			new SEqualIgnoreCase();
			new SConcat();
			new Strremove();
			new Startswith();
			new Req();
			new InContext();
		}
		/**
		 * InContext ( name --> boolean ) 
		 * Checks the context to see of an Entity with the given name is in the 
		 * current context.
		 * @author Paul Snow
		 *
		 */
		static class InContext extends ROperator {
            InContext(){super("InContext");}

            public void execute(DTState state) throws RulesException {
                RName       entityName  = state.datapop().rNameValue();
                IREntity    entity      = state.findEntity(entityName);
                state.datapush(entity);
            }
        } 

	    /**
	     * not( Boolean -- ~boolean )
	     * Not Operator, returns the negation of the input value
	     *
	     */
		static class Not extends ROperator {
			Not(){super("not");}

			public void execute(DTState state) throws RulesException {
				state.datapush(RBoolean.getRBoolean(!(state.datapop().booleanValue())));
			}
		} 

	    /**
	     * And( Boolean1 Boolean2 -- Boolean3 )
	     * And Operator, returns the && value of two booleans
	     *
	     */
		static class And extends ROperator {
			And(){super("&&"); alias("and");}

			public void execute(DTState state) throws RulesException {
                boolean v2 = state.datapop().booleanValue();
                boolean v1 = state.datapop().booleanValue();
                
				state.datapush(RBoolean.getRBoolean(v1 && v2));
			}
		}		

	    /**
	     * Or( Boolean1 Boolean2 -- Boolean3 )
	     * And Operator, returns the || value of two booleans
	     */
		static class Or extends ROperator {
			Or(){super("||");}

			public void execute(DTState state) throws RulesException {
				state.datapush(RBoolean.getRBoolean(state.datapop().booleanValue() || state.datapop().booleanValue()));
			}
		}
		
	    /**
	     * Greaterthan( Number Number -- Boolean )
	     * Greaterthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class Greaterthan extends ROperator {
			Greaterthan(){super(">");}

			public void execute(DTState state) throws RulesException {
			    IRObject o2 = state.datapop();
			    IRObject o1 = state.datapop();
				long number2 = o2.longValue();
				long number1 = o1.longValue();
				state.datapush(RBoolean.getRBoolean(number1 > number2));
			}
		}		

	    /**
	     * Lessthan( Number Number -- Boolean )
	     * Lessthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class Lessthan extends ROperator {
			Lessthan(){super("<");}

			public void execute(DTState state) throws RulesException {
				long number2 = state.datapop().longValue();
				long number1 = state.datapop().longValue();
				state.datapush(RBoolean.getRBoolean(number1 < number2));
			}
		}		
		
	    /**
	     * Greaterthanequal( Number Number -- Boolean )
	     * Greaterthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class Greaterthanequal extends ROperator {
			Greaterthanequal(){super(">=");}

			public void execute(DTState state) throws RulesException {
				long number2 = state.datapop().longValue();
				long number1 = state.datapop().longValue();
				state.datapush(RBoolean.getRBoolean(number1 >= number2));
			}
		}	

	    /**
	     * Lessthanequal( Number Number -- Boolean )
	     * Lessthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class Lessthanequal extends ROperator {
			Lessthanequal(){super("<=");}

			public void execute(DTState state) throws RulesException {
				long number2 = state.datapop().longValue();
				long number1 = state.datapop().longValue();
				state.datapush(RBoolean.getRBoolean(number1 <= number2));
			}
		}		

	    /**
	     * Equal( Number Number -- Boolean )
	     * Lessthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class Equal extends ROperator {
			Equal(){super("==");}

			public void execute(DTState state) throws RulesException {
				state.datapush(RBoolean.getRBoolean(state.datapop().longValue() == state.datapop().longValue()));
			}
		}		

	    /**
	     * FGreaterthan( Number Number -- Boolean )
	     * FGreaterthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class FGreaterthan extends ROperator {
			FGreaterthan(){super("f>");}

			public void execute(DTState state) throws RulesException {
				double number2 = state.datapop().doubleValue();
				double number1 = state.datapop().doubleValue();
				state.datapush(RBoolean.getRBoolean(number1 > number2));
			}
		}		

	    /**
	     * FLessthan( Number Number -- Boolean )
	     * FLessthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class FLessthan extends ROperator {
			FLessthan(){super("f<");}

			public void execute(DTState state) throws RulesException {
				double number2 = state.datapop().doubleValue();
				double number1 = state.datapop().doubleValue();
				state.datapush(RBoolean.getRBoolean(number1 < number2));
			}
		}		
		
	    /**
	     * FGreaterthanequal( Number Number -- Boolean )
	     * FGreaterthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class FGreaterthanequal extends ROperator {
			FGreaterthanequal(){super("f>=");}

			public void execute(DTState state) throws RulesException {
				double number2 = state.datapop().doubleValue();
				double number1 = state.datapop().doubleValue();
				state.datapush(RBoolean.getRBoolean(number1 >= number2));
			}
		}	

	    /**
	     * FLessthanequal( Number Number -- Boolean )
	     * FLessthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class FLessthanequal extends ROperator {
			FLessthanequal(){super("f<=");}

			public void execute(DTState state) throws RulesException {
				double number2 = state.datapop().doubleValue();
				double number1 = state.datapop().doubleValue();
				state.datapush(RBoolean.getRBoolean(number1 <= number2));
			}
		}		

	    /**
	     * FEqual( Number Number -- Boolean )
	     * FEqual Operator, returns the boolean value for condition with the given parameters
	     */
		static class FEqual extends ROperator {
			FEqual(){super("f==");}

			public void execute(DTState state) throws RulesException {
				state.datapush(RBoolean.getRBoolean(state.datapop().doubleValue() == state.datapop().doubleValue()));
			}
		}		

	    /**
	     * Isnull(object -- Boolean )
	     * Isnull Operator, returns true if the object is null
	     */
		static class Isnull extends ROperator {
			Isnull(){super("isnull");}

			public void execute(DTState state) throws RulesException 
			{
				state.datapush(RBoolean.getRBoolean(state.datapop().type()==IRObject.iNull));
			}
		}

	    /**
	     * Booleanequal(boolean1 boolean2 -- Boolean )
	     * Booleanequal Operator, returns true if both are equal
	     */
		static class Booleanequal extends ROperator {
			Booleanequal(){super("b="); alias("beq");}

			public void execute(DTState state) throws RulesException {
			    IRObject o2 = state.datapop();
			    IRObject o1 = state.datapop();
			    boolean r = false;
			    try{
			        r = o1.booleanValue() == o2.booleanValue();
			    }catch(RulesException e){}   // Ignore any failures, and simply fail.
			    
				state.datapush(RBoolean.getRBoolean(r)  );
			}
		}		

	    /**
	     * Booleannotequal(boolean1 boolean2 -- Boolean )
	     * Booleannotequal Operator, returns true if both are not equal
	     */
		static class Booleannotequal extends ROperator {
			Booleannotequal(){super("b!=");}

			public void execute(DTState state) throws RulesException {
				state.datapush(RBoolean.getRBoolean(state.datapop().booleanValue()!=state.datapop().booleanValue()));
			}
		}

	    /**
	     * SGreaterthan( String String -- Boolean )
	     * SGreaterthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class SGreaterthan extends ROperator {
			SGreaterthan(){super("s>");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RBoolean.getRBoolean(value1.compareTo(value2)>0));
			}
		}		

	    /**
	     * SLessthan( String String -- Boolean )
	     * SLessthan Operator, returns the boolean value for condition with the given parameters
	     */
		static class SLessthan extends ROperator {
			SLessthan(){super("s<");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RBoolean.getRBoolean(value1.compareTo(value2)< 0));
			}
		}		
		
	    /**
	     * SGreaterthanequal( String String -- Boolean )
	     * SGreaterthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class SGreaterthanequal extends ROperator {
			SGreaterthanequal(){super("s>=");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RBoolean.getRBoolean(value1.compareTo(value2)>=0));
			}
		}	

	    /**
	     * SLessthanequal( String String -- Boolean )
	     * SLessthanequal Operator, returns the boolean value for condition with the given parameters
	     */
		static class SLessthanequal extends ROperator {
			SLessthanequal(){super("s<=");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RBoolean.getRBoolean(value1.compareTo(value2)<=0));
			}
		}		

	    /**
	     * SEqual( String String -- Boolean )
	     * SEqual Operator, returns the boolean value for condition with the given parameters
	     */
		static class SEqual extends ROperator {
			SEqual(){super("s=="); alias("streq");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RBoolean.getRBoolean(value1.equals(value2)));
			}
		}		

        /**
         * SEqualIgnoreCase( String String -- Boolean )
         * Same as SEqual Operator, only ignores the case. 
         * Returns the boolean value for condition with the given parameters
         */
        static class SEqualIgnoreCase extends ROperator {
            SEqualIgnoreCase(){super("sic=="); alias("streqignorecase");}

            public void execute(DTState state) throws RulesException {
                String value2 = state.datapop().stringValue();
                String value1 = state.datapop().stringValue();
                state.datapush(RBoolean.getRBoolean(value1.equalsIgnoreCase(value2)));
            }
        }       
		
		
		
	    /**
	     * StrConcat( String String -- String )
	     * StrConcat Operator, add the given two strings and returns a string value
	     */
		static class SConcat extends ROperator {
			SConcat(){super("s+"); alias("strconcat");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RString.newRString(value1+value2));
			}
		}		

		 /**
         * startswith ( string1 string2 index -- boolean )
         * returns true if string1 begins with string2
         */
        static class Startswith extends ROperator {
            Startswith(){super("startswith");}

            public void execute(DTState state) throws RulesException {
                int    index   = state.datapop().intValue();
                String string2 = state.datapop().stringValue();
                String string1 = state.datapop().stringValue();
                state.datapush(RBoolean.getRBoolean(string1.startsWith(string2,index)));
            }
        }
        
        
	    /**
	     * Strremove( String1 String2 -- String3 )
	     * Strremove Operator, removes string2 from string1 and returns string3
	     */
		static class Strremove extends ROperator {
			Strremove(){super("strremove");}

			public void execute(DTState state) throws RulesException {
				String value2 = state.datapop().stringValue();
				String value1 = state.datapop().stringValue();
				state.datapush(RString.newRString(value1.replaceAll(value2, "")));
			}
		}	

	    /**
	     * Req( object1 object2 -- Boolean )
	     * Req Operator, compares the two objects using equals and returns the boolean value
	     */
		static class Req extends ROperator {
			Req(){super("req");}

			public void execute(DTState state) throws RulesException {
				IRObject value2 = state.datapop();
				IRObject value1 = state.datapop();
				state.datapush(RBoolean.getRBoolean(value1.equals(value2)));
			}
		}			
}