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

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RDouble;
import com.dtrules.interpreter.RInteger;
import com.dtrules.session.DTState;
/**
 * Defines math operators.
 * @author paul snow
 *
 */
public class RMath {
    static {
        new Add();      new Mul();        new Sub();        new Div();
        new FAdd();     new FMul();       new FSub();       new FDiv();
        new Abs();      new Negate();
        new FAbs();     new FNegate();
        new Roundto();
    }
    
    /**
     * ( number #places boundary -- number2 )
     * The number of decimal places would be zero to round to the nearest 
     * integer, 1 to the nearest 10th, -1 to the nearest 10. The boundary 
     * is the value added to the lower fractional amount to trigger the 
     * round.  In other words, <br><br>
     * 
     * 1.3 0 .7 roundto<br><br>
     * 
     * would result in 1, while<br><br>
     * 
     * 1.7 0 .7 roundto<br>
     * 
     * would reslut in 2.<br><br><br>
     * 
     * Note:  if the boundary is zero, than any fractional amount will round up.  
     * If the boundary is 1, the number is simply truncated.
     * 
     * Limitation:  We always towards zero.  Lots of other rounding ideas are
     * possible.  We need to come back here and rework and add to our options
     * if the need for more complexity comes our way.
     * 
     * @author paul snow
     *
     */
    static class Roundto extends ROperator {
        Roundto(){super("roundto"); }
        
        double round(double number,double boundary){
            double v = (int)number;                 // Get the integer porition of number
            if (boundary>=1) return v;              // If boundary is 1 or greater we are done
            double r = Math.abs(number - v);        // Get the fractional portion of number
            if (boundary<=0) return r>0 ? v++ : v;  // If boundary is 0 or less, inc on any fraction
            if(r>=boundary)return v++;              // Otherwise test the boundary.  Inc if fraction
            return v;                               //    is greater or equal to the boundary.
        }
        public void execute(DTState state)throws RulesException {
            double boundary = state.datapop().doubleValue();
            int    places   = state.datapop().intValue();
            double number   = state.datapop().doubleValue();
            if(places >0){                          // We put the boundary across zero. shift left if
                number *= 10*places;                //    places is positive (okay, its a decimal shift)
                number = round(number,boundary);    // Do the round thing.
                number /= 10*places;                // Fix it back when done.
            }else{
                number /= -10*places;               // We decimal shift right if places is negative
                number = round(number,boundary);    // Do the round thing
                number *= -10*places;               // Fix it back.
            }
            
        }
    }
    
    
    /**
     * Negate a double
     * @author paul snow
     *
     */
    static class FNegate extends ROperator {
        FNegate(){super("fnegate"); }
        
        public void execute(DTState state)throws RulesException {
            state.datapush(
              RDouble.getRDoubleValue(
                 -state.datapop().doubleValue()
              )
            );
        }
    }
    
    
    /**
     * Absolute value of a double.
     * @author paul snow
     *
     */
    static class FAbs extends ROperator {
        FAbs(){super("fabs"); }
        
        public void execute(DTState state)throws RulesException {
            state.datapush(
              RDouble.getRDoubleValue(
                 Math.abs(state.datapop().doubleValue())
              )
            );
        }
    }
    /**
     * Absolute value of an integer
     * @author paul snow
     *
     */
    static class Abs extends ROperator {
        Abs(){super("abs"); }
        
        public void execute(DTState state)throws RulesException {
            state.datapush(
              RInteger.getRIntegerValue(
                 Math.abs(state.datapop().intValue())
              )
            );
        }
    }
    
    /**
     * Negate an integer
     * @author paul snow
     *
     */
    static class Negate extends ROperator {
        Negate(){super("negate"); }
        
        public void execute(DTState state)throws RulesException {
            state.datapush(
              RInteger.getRIntegerValue(
                 -state.datapop().intValue()
              )
            );
        }
    }    

    /**
     * Add Operator, adds two integers
     * @author Paul Snow
     *
     */
	static class Add extends ROperator {
		Add(){
			super("+"); alias("ladd");
		}

		public void execute(DTState state) throws RulesException {
			state.datapush(RInteger.getRIntegerValue(state.datapop().longValue()+state.datapop().longValue()));
		}
	}
	
	
	
	
	/**
	 * Sub Operator, subracts two integers
	 * @author Paul Snow
	 *
	 */
	static class Sub extends ROperator {
		Sub(){super(RDecisionTable.DASH); alias("lsub");}

		public void execute(DTState state) throws RulesException {
			long b = state.datapop().longValue();
			long a = state.datapop().longValue();
			long result = a-b;
			state.datapush(RInteger.getRIntegerValue(result));
		}
	}

	/**
	 * Mul Operator, multiply two integers
	 * @author Paul Snow
	 *
	 */
	static class Mul extends ROperator {
		Mul(){super("*"); alias("lmul");}

		public void execute(DTState state) throws RulesException {
			state.datapush(RInteger.getRIntegerValue(state.datapop().longValue()*state.datapop().longValue()));
		}
	}

	/**
	 * Divide Operator, divides one integer by another
	 * @author Paul Snow
	 *
	 */
	static class Div extends ROperator {
		Div(){super("/"); alias("div"); alias("ldiv");}

		public void execute(DTState state) throws RulesException {
			long result;
			long a=0; long b=0;
			try {
				b = state.datapop().longValue();
				a = state.datapop().longValue();
				result = a/b;
			} catch (ArithmeticException e) {
				throw new RulesException("Math Exception","/","Error in Divide: "+a+"/"+b+"\n"+e);
			}
			state.datapush(RInteger.getRIntegerValue(result));
		}
	}
	
    /**
     * FAdd (f+) Operator, adds two doubles
     * @author Paul Snow
     *
     */
    static class FAdd extends ROperator {
        FAdd(){
            super("f+");alias("fadd");
        }

        public void execute(DTState state) throws RulesException {
            IRObject b = state.datapop();
            IRObject a = state.datapop();
            state.datapush(RDouble.getRDoubleValue(a.doubleValue()+b.doubleValue()));
        }
    }
    
    
    
    
    /**
     * FSub (f-) Operator, subracts two doubles
     * @author Paul Snow
     *
     */
    static class FSub extends ROperator {
        FSub(){super("f-");alias("fsub");}

        public void execute(DTState state) throws RulesException {
            double b = state.datapop().doubleValue();
            double a = state.datapop().doubleValue();
            double result = a-b;
            state.datapush(RDouble.getRDoubleValue(result));
        }
    }

    /**
     * FMul Operator, multiply two doubles
     * @author Paul Snow
     *
     */
    static class FMul extends ROperator {
        FMul(){super("f*");alias("fmul");}

        public void execute(DTState state) throws RulesException {
            IRObject d2 = state.datapop();
            IRObject d1 = state.datapop();
            state.datapush(RDouble.getRDoubleValue(d1.doubleValue()*d2.doubleValue()));
        }
    }

    /**
     * FDiv Operator, divides one double by another double.
     * @author Paul Snow
     *
     */
    static class FDiv extends ROperator {
        FDiv(){super("fdiv"); alias("f/");}

        public void execute(DTState state) throws RulesException {
            double result;
            double a=0; double b=0;
            try {
                b = state.datapop().doubleValue();
                a = state.datapop().doubleValue();
                result = a/b;
            } catch (ArithmeticException e) {
                throw new RulesException("Math Exception","f/","Error in Divide: "+a+"/"+b+"\n"+e);
            }
            state.datapush(RDouble.getRDoubleValue(result));
        }
    }
}
