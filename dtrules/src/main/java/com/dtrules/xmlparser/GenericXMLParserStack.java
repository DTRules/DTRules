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
import java.util.Stack;

/** 
 * The GenericXMLParserStack allows the creation of parsers
 * for particular tags that are separate and distinct from
 * the general parser as a whole.  When a Parser is pushed,
 * it takes over the processing of the current tag and all
 * sub tags until the end tag is reached.  Then control is 
 * returned to the previous parser.
 * 
 * @author Paul snow
 *
 */
public class GenericXMLParserStack extends AGenericXMLParser{

    final GenericXMLParser   flexParser;
    Stack<IGenericXMLParser> parsers     = new Stack<IGenericXMLParser>();
    Stack<Integer>           tagstkptrs  = new Stack<Integer>();
    
    /**
     *  The Stack processing of this parser requires it to look
     *  at the state of the flexParser. 
     */
    public GenericXMLParserStack(GenericXMLParser flexParser) {
        this.flexParser = flexParser;
    }
    
    public void parseTagWith(IGenericXMLParser parser) throws Exception {
        parsers.push(parser);
        tagstkptrs.push(flexParser.tagstkptr);
        parser.beginTag(
                flexParser.tagstk, 
                flexParser.tagstkptr, 
                flexParser.currenttag, 
                flexParser.attribs);
    }
    
    public void beginTag(String[] tagstk, int tagstkptr, String tag,
            HashMap<String, String> attribs) throws IOException, Exception {
        
        IGenericXMLParser  parser = parsers.peek();
        parser.beginTag(tagstk, tagstkptr, tag, attribs);
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.AGenericXMLParser#comment(java.lang.String)
     */
    @Override
    public void comment(String comment) {
        IGenericXMLParser  parser = parsers.peek();
        if(parser instanceof IGenericXMLParser2){
            ((IGenericXMLParser2) parser).comment(comment);
        }
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.AGenericXMLParser#endTag(java.lang.String[], int, java.lang.String, java.lang.String, java.util.HashMap)
     */
    @Override
    public void endTag(String[] tagstk, int tagstkptr, String tag, String body,
            HashMap<String, String> attribs) throws Exception, IOException {
        IGenericXMLParser parser;
        if(tagstkptr == tagstkptrs.peek()){                         // We allow BOTH parsers to process
            parser = parsers.pop();                                 // the "entry" tag for the called
            tagstkptrs.pop();                                       // parser.  So call the called endTag...
            parser.endTag(tagstk, tagstkptr, tag, body, attribs);   
        }
        parser = parsers.peek();                                    // And always call the end tag for the
        parser.endTag(tagstk, tagstkptr, tag, body, attribs);       //   active parser.
    }

    @Override
    public boolean error(String v) throws Exception {
        return parsers.peek().error(v);
    }

    @Override
    public void header(String header) {
        IGenericXMLParser  parser = parsers.peek();
        if(parser instanceof IGenericXMLParser2){
            ((IGenericXMLParser2) parser).header(header);
        }
    }

}
