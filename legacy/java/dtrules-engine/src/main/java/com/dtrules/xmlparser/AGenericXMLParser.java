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
package com.dtrules.xmlparser;

import java.io.IOException;
import java.util.HashMap;

/**
 * This Abstract class cuts down on the clutter when implementing a little
 * XML parser.  It defaults to the parser doing absolutely nothing. This way
 * your small parser can simply catch just the XML structures it needs to
 * parse for what you need from an XML source.
 * 
 * The Abstract Class implements the second version (IGenericXMLParser2) 
 * interface.  If you make your parser inherit from this class, updates to 
 * the IGenericXMLParser2 interface will not likely hurt your parser.
 *  
 * @author Paul Snow
 *
 */
public abstract class AGenericXMLParser implements IGenericXMLParser2 {
    
    GenericXMLParserStack genericXMLParserStack;
    
    public void parseTagWith(IGenericXMLParser parser) throws Exception{
        genericXMLParserStack.parseTagWith(parser);
    }
    
    /**
     * By default, ignore comments
     */
    @Override
    public void comment(String comment) {}
    /**
     * By default, ignore the header
     */
    @Override
    public void header(String header) {}
    /**
     * By default ignore the beginning Tags; 
     */
    @Override
    public void beginTag(
            String[] tagstk, 
            int tagstkptr, 
            String tag,
            HashMap<String, String> attribs) throws IOException, Exception {}

    /**
     * By default ignore ending tags.
     */
    @Override
    public void endTag(
            String[] tagstk, 
            int tagstkptr, 
            String tag, 
            String body,
            HashMap<String, String> attribs) throws Exception, IOException {}

    /**
     * By default, keep plugging even if we encounter an XML format error.
     */
    @Override
    public boolean error(String v) throws Exception {
        return true;
    }
    
        
}
