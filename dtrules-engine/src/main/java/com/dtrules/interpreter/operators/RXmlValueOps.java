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
  
package com.dtrules.interpreter.operators;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RXmlValue;
import com.dtrules.mapping.XMLNode;
import com.dtrules.mapping.XMLTag;
import com.dtrules.session.DTState;

public class RXmlValueOps {
	    static {
	    	new SetXmlAttribute();
	    	new GetXmlAttribute();
	    	new NewXmlAttribute();
	    }
	 
	    /**
	     * ( Name --> xmlValue)
	     * Creates a new XmlValue of the given name.
	     */
	    public static class NewXmlAttribute extends ROperator {
	        NewXmlAttribute(){super("newxmlattribute"); }
	        @Override
            public void execute(DTState state) throws RulesException {
                RName     name      = state.datapop().rNameValue();
                XMLTag    xmlNode   = new XMLTag(name.stringValue(),null);
                RXmlValue xmlValue  = new RXmlValue(state,xmlNode);
                
                state.datapush(xmlValue);
            }
	    }
	    /**
	     * SetXmlAttribute ( XmlValue Attribute Value --> )
	     * Overwrites the attribute in the XML node.  If the object provided
	     * doesn't actually have an XmlValue, this becomes a no op.
	     * @author Paul Snow
	     *
	     */
		public static class SetXmlAttribute extends ROperator {
			SetXmlAttribute(){super("setxmlattribute");}

			@Override
            public void execute(DTState state) throws RulesException {
				IRObject  value     = state.datapop();
				IRObject  attribute = state.datapop();
				XMLNode   xmlNode   = state.datapop().xmlTagValue();
				if(xmlNode != null){
				    state.traceInfo("SetXmlAttribute","tag",xmlNode.getTag(),"attribute",attribute.stringValue(),"value",value.stringValue(), null);
				    xmlNode.getAttribs().put(attribute.stringValue(), value.stringValue());
				}
			}
		}
		/**
         * GetXmlAttribute ( XmlValue Attribute --> Value )
         * Get the value of the given attribute from this XmlValue.
         * If the attribute is not defined, or an Entity is found
         * with no XMLValue, a null is returned.
         * @author Paul Snow
         *
         */
        public static class GetXmlAttribute extends ROperator {
            GetXmlAttribute(){super("getxmlattribute");}

            @Override
            public void execute(DTState state) throws RulesException {
                String    attribute = state.datapop().stringValue();
                XMLNode   xmlNode    = state.datapop().xmlTagValue();
                if(xmlNode == null) {
                    state.datapush(RNull.getRNull());
                    state.traceInfo("GetXmlAttribute","tag","nullTag","attribute",attribute,"value","null",null);
                    return;
                }
                String    value     = (String) xmlNode.getAttribs().get(attribute);
                if(value != null ){
                   state.datapush(RString.newRString(value)); 
                   state.traceInfo("GetXmlAttribute","tag",xmlNode.getTag(),"attribute",attribute,"value",value,null);
                }else{
                   state.datapush(RNull.getRNull()); 
                   state.traceInfo("GetXmlAttribute","tag",xmlNode.getTag(),"attribute",attribute,"null","true",null);
                }
            }
        }
		
        
}