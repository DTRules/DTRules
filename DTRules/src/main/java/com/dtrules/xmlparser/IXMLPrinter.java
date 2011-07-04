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

public interface IXMLPrinter {

    /**
     * Returns the number of tags on the tag stack.
     */
    public int depth();
    
    /**
     * Returns the tag with the given index.  Returns null if out
     * of range.
     */
    public String getTag(int i);
    
    /**
     * Prints a simple open tag with no attributes.
     * @param tag
     */
    public abstract void opentag(String tag);

    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     */
    public abstract void opentag(String tag, String name1, Object value1);

    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     */
    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2);

    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     */
    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3);

    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param name4
     * @param value4
     */
    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5, String name6, Object value6);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5, String name6, Object value6, String name7, Object value7);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5, String name6, Object value6, String name7, Object value7, String name8,
            Object value8);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5, String name6, Object value6, String name7, Object value7, String name8,
            Object value8, String name9, Object value9);

    public abstract void opentag(String tag, String name1, Object value1, String name2,
            Object value2, String name3, Object value3, String name4, Object value4, String name5,
            Object value5, String name6, Object value6, String name7, Object value7, String name8,
            Object value8, String name9, Object value9, String name10,Object value10);
    /**
     * Closes the currently open tag.  Assumes no body text.  Throws a
     * runtime exception if no open tag exists.
     */
    public abstract void closetag();

    /**
     * Print data within a data tag.  The text is encoded.
     * @param text
     */
    public abstract void printdata(Object bodyvalue);

    /**
     * Print data within a given tag.
     */
    public abstract void printdata(String tag, Object bodyvalue);

    /**
     * Print the tag, one attribute, and the body.
     * @param tag
     * @param name1
     * @param value
     * @param body
     */
    public abstract void printdata(String tag, String name1, Object value, Object body);
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            String name4, Object value4, 
            String name5, Object value5, 
            Object bodyvalue);
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            String name4, Object value4, 
            Object bodyvalue);
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            Object bodyvalue);
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            Object bodyvalue);
    /**
     * Closes all open tags, close the file.
     *
     */
    public abstract void close();

    /**
     * Just a helper to print an error during the generation of an XML file
     * @param errorMsg The error message to be printed.  Put into an <error> tag in the XML.
     */
    public abstract void print_error(String errorMsg);

}