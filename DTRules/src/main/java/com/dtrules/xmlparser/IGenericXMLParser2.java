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

/**
 * This is a simple interface for handling parsing events
 * from the generic parser.  At this time, only two events
 * are generated, the beginTag and the endTag events.  The
 * endTag() provides the body, if present.
 * Creation date: (9/15/2003 8:35:17 AM)
 * @author: Paul Snow, MTBJ, Inc.
 */
public interface IGenericXMLParser2 extends IGenericXMLParser {
       /**
        * Called with the contents of a comment line.
        * @param comment
        */
       void comment(String comment);
       
       /**
        * Called with the contexts of a header line
        * @param header
        */
       void header(String header);
}