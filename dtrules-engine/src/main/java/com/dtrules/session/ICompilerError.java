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

/**
 * ICompilerError defines a linked list of errors generated when compiling
 * a decision table.  The error types are:
 * <br><br>
 * 1 -- Condition <br>
 * 2 -- Action <br>
 * 3 -- Table <br>
 * 4 -- InitialAction <br>
 * 5 -- Context <br>
 * <br><br>
 * @author paul snow
 * Feb 20, 2007
 *
 */
public interface ICompilerError {
    
    enum Type { CONDITION, ACTION, TABLE, INITIALACTION, CONTEXT};
    
    /**
    * Returns the Error type, which will be a 1 if the error was in the
    * compilation of a Condition, a 2 if the error was in the compliation
    * of an Action, or a 3 if it was in building the Table.
    * @return ErrorTypeNumber
    */
   Type             getErrorType();
   /**
    * Returns the text provided the compiler which generated the error. If
    * the error type is a 3, this function returns null.
    * @return Source code of error
    */
   String           getSource();
   /**
    * Returns an error message explaining the error.
    * @return message explaining the error
    */
   String           getMessage();
   /**
    * Returns the index (1 based) of the condition that triggered the error.
    * Returns a value > 0 if compling a condition caused the error, and a 
    * zero otherwise.  Error types 2 and 3 will always return zero.
    * @return index of condition
    */
   int              getConditionIndex();
   /**
    * Returns the index (1 based) of the action that triggered the error.  
    * Returns a value > 0 if compling an action caused the error, and a zero
    * otherwise.  Error types 1 and 3 will always return zero.
    * @return index of action
    */
   int              getActionIndex();
   /**
    * Really, we should toss the getConditionIndex and getActionIndex in
    * favor of a getIndex which returns a 1 or greater for valid indexes,
    * and a zero otherwise.  Use the errorType to decide if this is an
    * index into InitialActions, Conditions, or Actions.
    */
   int              getIndex();
   /**
    * If the error was due to an unbalanced decision table, this funciton 
    * returns the row in the condition table where the error was detected. 
    * @return row index of error
    */
   int              getRow();
   /**
    * If the error was due to an unbalanced decision table, this funciton 
    * returns the column in the condition table where the error was detected. 
    * @return column index of error
    */
   int              getCol();
   
}
