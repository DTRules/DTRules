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
 

import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.XMLNode;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;

public abstract class ARObject implements IRObject {

    /**
     * No point in implementing this here.  Every Object that has
     * an array representation needs to implement it themselves.
     * 
     * @see com.dtrules.interpreter.IRObject#rArrayValue()
     */
    public RArray rArrayValue() throws RulesException {
       throw new RulesException("Conversion Error","ARObject","No Array Value value exists for "+this.stringValue());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#rBooleanValue()
     */
    public RBoolean rBooleanValue() throws RulesException {
        return RBoolean.getRBoolean(booleanValue());
    }
    /**
     * @see com.dtrules.interpreter.IRObject#rDoubleValue()
     */
    public RDouble rDoubleValue() throws RulesException {
        return RDouble.getRDoubleValue(doubleValue());
    }
    
    /**
     * @see com.dtrules.interpreter.IRObject#rTimeValue()
     */
    public RDate rTimeValue() throws RulesException {
        return RDate.getRTime(timeValue());
    }

    public RDate rTimeValue (IRSession session) throws RulesException {
    	return RDate.getRTime(timeValue());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#timeValue()
     */
    public Date timeValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Time value exists for: "+this.type());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#execute(DTState)
     */
	public void execute(DTState state) throws RulesException {
		state.datapush(this);
	}

    /**
     * @see com.dtrules.interpreter.IRObject#arrayExecute(DTState)
     */
	public void arrayExecute(DTState state) throws RulesException {
		state.datapush(this);
	}

    /**
     * @see com.dtrules.interpreter.IRObject#getExecutable()
     */
    public IRObject getExecutable(){
    	return this;
    }
    
    /**
     * @see com.dtrules.interpreter.IRObject#getNonExecutable()
     */
    public IRObject getNonExecutable() {
    	return this;
    }

    /**
     * @see com.dtrules.interpreter.IRObject#equals(IRObject)
     */
	public boolean equals(IRObject o) throws RulesException {
		return o==this;
	}

    /**
     * @see com.dtrules.interpreter.IRObject#isExecutable()
     */
	public boolean isExecutable() {
		return false;
	}

    /**
     * @see com.dtrules.interpreter.IRObject#postFix()
     */
	public String postFix() {
		return toString();
	}
	
    /**
     * @see com.dtrules.interpreter.IRObject#rStringValue()
     */
    public RString rStringValue() {
        return RString.newRString(stringValue());
    }
    
    /**
     * @see com.dtrules.interpreter.IRObject#rclone()
     */
	public IRObject rclone() {
		return (IRObject) this;
	}

	/** Conversion Methods.  Default is to throw a RulesException **/
	
    /**
     * @see com.dtrules.interpreter.IRObject#intValue()
     */
    public int intValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Integer value exists for "+this.type());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#arrayValue()
     */
	public List<IRObject> arrayValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Array value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#booleanValue()
     */
	public boolean booleanValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Boolean value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#doubleValue()
     */
	public double doubleValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No double value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#rEntityValue()
     */
	public IREntity rEntityValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Entity value exists for "+this.type());
	}
    /**
     * @see com.dtrules.interpreter.IRObject#hashMapValue()
     */
	public HashMap<IRObject,IRObject> hashMapValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No HashMap value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#longValue()
     */
	public long longValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Long value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#rNameValue()
     */
	public RName rNameValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Name value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#rIntegerValue()
     */
	public RInteger rIntegerValue() throws RulesException {
        throw new RulesException("Undefined","Conversion Error","No Integer value exists for "+this.type());
	}

    /**
     * @see com.dtrules.interpreter.IRObject#compare(IRObject)
     */
	public int compare(IRObject irObject) throws RulesException {
        throw new RulesException("Undefined","No Supported",this.type()+" Objects do not support Compare");
	}

    /**
     * By default, objects clone themselves by simply returning themselves.
     * This is because the clone of a number or boolean etc. is itself.
     *
     * @see com.dtrules.interpreter.IRObject#clone(IRSession)
     */
    public IRObject clone(IRSession s) throws RulesException {
        return this;
    }

    /**
     * @see com.dtrules.interpreter.IRObject#rTableValue()
     */
    public RTable rTableValue() throws RulesException {
        throw new RulesException("Undefined","Not Supported","No Table value exists for "+this.type());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#tableValue()
     */
    public Map<IRObject,IRObject> tableValue() throws RulesException {
        throw new RulesException("Undefined","Not Supported","No Table value exists for "+this.type());
    }

    /**
     * @see com.dtrules.interpreter.IRObject#rXmlValue()
     */
    public IRObject rXmlValue() throws RulesException {
        return RNull.getRNull();
    }
    
    /**
     * @see com.dtrules.interpreter.IRObject#xmlTagValue()
     */
    public XMLNode xmlTagValue() throws RulesException {
        return null;
    }    
    
}
