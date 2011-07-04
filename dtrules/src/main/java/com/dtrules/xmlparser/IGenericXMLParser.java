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
 * This is a simple interface for handling parsing events
 * from the generic parser.  At this time, only two events
 * are generated, the beginTag and the endTag events.  The
 * endTag() provides the body, if present.
 * Creation date: (9/15/2003 8:35:17 AM)
 * @author: Paul Snow, MTBJ, Inc.
 */
public interface IGenericXMLParser {

	/**
	 * The beginTag method is called when the parser detects the
	 * each begin tag in the XML file being parsed.  When one uses
	 * a SAX parser, the class implementing the interface is responsible
	 * for keeping up with the context of the tags as they are 
	 * encountered and closed.  The GenericXMLParser provides a
	 * stack of all the tags currently open when the begin tag
	 * is encountered.  This is the actual stack maintained by the
	 * GenericXMLParser, which is maintained as an array of tags.
	 * The tagstkptr provides an index into this array. <br><br>
	 * The attributes associated with the tag are provided as a
	 * hashmap of key/Value pairs. <br><br>
	 * 
	 * @param tagstk A stack of tags active at the time the tag was encountered.
	 * @param tagstkptr A pointer into the tag stack to the top of stack.
	 * @param tag The tag encountered.
	 * @param attribs A hash map of attributes as a set of key/Value pairs.
	 * @throws IOException If an error occurs while parsing the XML, this exception will be thrown.
	 * @throws Exception If the tags are not matched, or an unexpected character is encountered, an
	 *                   exception will be thrown.
	 */
	public void beginTag(
		String tagstk[],
		int tagstkptr,
		String tag,
		HashMap<String, String> attribs)
		throws IOException, Exception;

    /**
	 * The endTag method is called when the parser detects the
	 * each end tag in the XML file being parsed.  When one uses
	 * a SAX parser, the class implementing the interface is responsible
	 * for keeping up with the context of the tags as they are 
	 * encountered and closed.  The GenericXMLParser provides a
	 * stack of all the tags currently open when the end tag
	 * is encountered.  The GenericXMLParser also provides the
	 * attributes of the begin tag to the end tag as well.
	 *  <br><br>
	 * 
	 * @param tagstk A stack of tags active at the time the tag was encountered.
	 * @param tagstkptr A pointer into the tag stack to the top of stack.
	 * @param tag The tag encountered.
	 * @param body the body (if any) of the data between the begin and end tags.
	 * @param attribs A hash map of attributes as a set of key/Value pairs.
	 * @throws IOException If an error occurs while parsing the XML, this exception will be thrown.
	 * @throws Exception If the tags are not matched, or an unexpected character is encountered, an
	 *                   exception will be thrown.
	**/ 
	public void endTag(
		java.lang.String[] tagstk,
		int tagstkptr,
		String tag,
		String body,
		HashMap<String,String> attribs)
		throws Exception, IOException;

	/**
	 * When an error is encountered by the parser, the error method
	 * is called on the class implementing the IGenericXMLParser
	 * interface.  This method returns true if parsing should continue.
	 * @param v This is the error string from the GenericXMLParser
	 */
	public boolean error(String v)
		throws Exception; // Returns true if parsing should continue.
}
