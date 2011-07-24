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

import java.io.OutputStream;
import java.io.PrintWriter;
import java.io.Writer;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;

import com.dtrules.interpreter.RDate;

/**
 * A simple class for the support of printing XML files.  Start and end tags
 * are tracked and printed.  An enclosing tag is printed.  Strings are encoded
 * automatically.  Tags are specified by their name, i.e. "Document" and not 
 * using any XML syntax, i.e. "<Document>".
 * 
 * @author paul snow
 * Jul 9, 2007
 *
 */
public class XMLPrinter implements IXMLPrinter {
    ArrayList<String>  tagStack    = new ArrayList<String>();
    final PrintWriter  pw;
    boolean            newline     = true;
    boolean            intextbody  = false;
    boolean            intagbody   = false;
    boolean            noSpaces    = false;
    String             indentStr   = "\t";
   
    StringBuffer       buffer      = new StringBuffer();
    boolean            buffering   = false;
   
    /**
     * Start buffering XML output.  Later you can decide to delete
     * or write the buffered XML output.  A pointer into the buffer
     * is returned to allow you to nest calls to buffer output.
     * This index must be saved by the caller, and provided to either
     * the deleteBuffer() or writeBuffer() call.
     * 
     * @return buffer index
     */
    public int bufferOutput(){
        buffering =  true;
        return buffer.length();
    }
    /**
     * Delete the buffered output up to the point where the buffering
     * begins.  This is defined by the pointer returned by the bufferOutput
     * call.  Pointers that are greater than the current buffer size are
     * ignored (this can happen if you mess up your nesting of 
     * start buffer/end buffer calls) are ignored.
     * @param ptr
     */
    public void deleteBuffer(int ptr){
        if(ptr <= buffer.length()){
            buffer.setLength(ptr);
            buffering = ptr>0;
        }
    }
    /**
     * Write the buffer associated with the given buffer index.  If this
     * index is zero, then this will turn off buffering.
     * @param ptr
     */
    public void writeBuffer(int ptr){
        if(ptr==0){
            pw.write(buffer.toString());
            pw.flush();
            buffering =  false;
        }
    }
    
    private void prt(String s){
        if(buffering){
            buffer.append(s);
        }else{
            pw.write(s);
        }
        pw.flush();
    }
    
    public void setSpaceCnt(int cnt){
        if(cnt>=0){
            indentStr = "";
            for(int i=0;i<cnt;i++)indentStr += " ";
        }else{
            indentStr = "\t";
        }
    }
    
    public void setNoSpaces(boolean v){
        noSpaces = v;
    }
    
    /**
     * Returns the number of tags on the tag stack.
     */
    public int depth() {
        return tagStack.size();
    }
    
    /**
     * Returns the tag with the given index.  Returns null if out
     * of range.
     */
    public String getTag(int i){
        if(i<0 || i>=tagStack.size())return null;
        return tagStack.get(i);
    }
    
    /**
     * This function puts the output on a newline, but not if we
     * are already on a newline.
     */
    private void newline(){
        if(noSpaces) return;
        if(!newline) prt("\r\n");
        newline = true;
    }
    /**
     * A helper function just to print text, and sets the state of the
     * newline function.  Note that we are never going to print a newline
     * on a non-XML syntax boundry.
     * @param text
     */
    private void print(String text){
        prt(text);
        newline = false;
    }
    /**
     * Just prints data inside a tag.  No effect on the newline character.
     * @param text
     */
    private void printData(String text){
        prt(text);
    }
    
    /**
     * Prints out spaces to indent
     */
    private void indent(){ indent(0);}
    private void indent(int mod){
        if(noSpaces)return;
        int indent = tagStack.size()+mod;
        for(int i=0;i<indent;i++)print(indentStr);
    }
    /**
     * Prints a comment 
     */
    public void comment(String comment){
        if(intextbody)throw new RuntimeException("Can't open a tag within a data body");
        newline();
        intagbody = false;
        intextbody = false;
        indent();
        print("<!--"); print(comment); print("-->");
    }
    /**
     * Print the header
     * @param comment
     */
    public void header(String header){
        if(intextbody)throw new RuntimeException("Can't open a tag within a data body");
        newline();
        intagbody = false;
        intextbody = false;
        indent();
        print("<?"); print(header); print("?>");
    }
    /**
     * Prints a simple open tag with no attributes.
     * @param tag
     */
    private void halfopentag(String tag){
        if(intextbody)throw new RuntimeException("Can't open a tag within a data body");
        newline();
        indent();
        print("<"); 
        print(tag);
        tagStack.add(tag);
        intagbody = false;  // Just don't know at this point how this tag will be used.
        intextbody = false;
    }
    
    /**
     * Prints a simple open tag with no attributes.
     * @param tag
     */
    public void opentag(String tag){
        halfopentag(tag);
        print(">");
    }
    /**
     * Prints an attribute.  The value is encoded.
     */
    private void printAttribute(String name, Object value){
        if(value==null)value="";
        name = name.replaceAll(" ", "_");
        print(" ");
        print(name);
        print("='");
        print(GenericXMLParser.encode(value.toString()));
        print("'");
    }
    
    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     */
    public void opentag(String tag, String name1, Object value1){
        halfopentag(tag);
        printAttribute(name1, value1);
        print(">");
    }
    /**
     * Open a tag with a given set of attributes
     */
    public void opentag(String tag, @SuppressWarnings("rawtypes") Map attribs){
        halfopentag(tag);
        if(attribs != null) for(Object key : attribs.keySet()){
            Object o = attribs.get(key);
            if(o!=null){
               printAttribute(key.toString(), o);
            }else{
               printAttribute(key.toString(),"");
            }
        }
        print(">");
    }

    /**
     * Open a tag with a given set of attributes; this is a Hash of strings to strings,
     * where as opentag(String tag, HashMap<String,Object> attribs) takes a Hash of 
     * strings to objects.  
     */
    public void opentagStringMap(String tag, HashMap<String,String> attribs){
        halfopentag(tag);
        if(attribs != null) for(String key : attribs.keySet()){
            Object o = attribs.get(key);
            if(o!=null){
               printAttribute(key, o);
            }else{
               printAttribute(key,"");
            }
        }
        print(">");
    }

    
    /**
     * Open a tag with one named attribute
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     */
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2
            ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        print(">");
    }
    
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
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3
            ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        print(">");
    }

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
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4
            ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        print(">");
    }

    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4,
            String name5, Object value5
                    ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        printAttribute(name5, value5);
        print(">");
    }
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4,
            String name5, Object value5,
            String name6, Object value6
                    ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        printAttribute(name5, value5);
        printAttribute(name6, value6);
        print(">");
    }
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4,
            String name5, Object value5,
            String name6, Object value6,
            String name7, Object value7
                    ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        printAttribute(name5, value5);
        printAttribute(name6, value6);
        printAttribute(name7, value7);
        print(">");
    }
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4,
            String name5, Object value5,
            String name6, Object value6,
            String name7, Object value7,
            String name8, Object value8
                    ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        printAttribute(name5, value5);
        printAttribute(name6, value6);
        printAttribute(name7, value7);
        printAttribute(name8, value8);
        print(">");
    }
    public void opentag(String tag, 
            String name1, Object value1,
            String name2, Object value2,
            String name3, Object value3,
            String name4, Object value4,
            String name5, Object value5,
            String name6, Object value6,
            String name7, Object value7,
            String name8, Object value8,
            String name9, Object value9
                    ){
        halfopentag(tag);
        printAttribute(name1, value1);
        printAttribute(name2, value2);
        printAttribute(name3, value3);
        printAttribute(name4, value4);
        printAttribute(name5, value5);
        printAttribute(name6, value6);
        printAttribute(name7, value7);
        printAttribute(name8, value8);
        printAttribute(name9, value9);
        print(">");
    }
    
        public void opentag(String tag, 
                String name1, Object value1,
                String name2, Object value2,
                String name3, Object value3,
                String name4, Object value4,
                String name5, Object value5,
                String name6, Object value6,
                String name7, Object value7,
                String name8, Object value8,
                String name9, Object value9,
                String name10,Object value10
                        ){
            halfopentag(tag);
            printAttribute(name1, value1);
            printAttribute(name2, value2);
            printAttribute(name3, value3);
            printAttribute(name4, value4);
            printAttribute(name5, value5);
            printAttribute(name6, value6);
            printAttribute(name7, value7);
            printAttribute(name8, value8);
            printAttribute(name9, value9);
            printAttribute(name10,value10);
            print(">");
    }

    
    /**
     * Closes the currently open tag.  Assumes no body text.  Throws a
     * runtime exception if no open tag exists.
     */ 
    public void closetag(){
        int lastIndex = tagStack.size()-1;
        if(intagbody){
            newline();
            indent(-1);
        }
        if(tagStack.size()<=0){
            throw new RuntimeException("No Enclosing Tag to close");
        }
        print("</");
        print(tagStack.get(lastIndex));
        print(">");
        newline();
        tagStack.remove(lastIndex);
        intextbody = false;
        intagbody = true;
    }
    /**
     * Print data within a data tag.  The text is encoded.
     * @param text
     */
    public void printdata(Object bodyvalue){
        if(intagbody){
            throw new RuntimeException("You can't mix data and tags");
        }
        if(bodyvalue != null){
            if(bodyvalue instanceof Date) {
                bodyvalue = RDate.getRTime((Date)bodyvalue);
            }
            String v = GenericXMLParser.encode(bodyvalue.toString());
            printData(v);
        }    
        intextbody = true;
    }
    
    /**
     * Print data within a given tag.
     */
    public void printdata(String tag, Object bodyvalue){
        opentag(tag);
        printdata(bodyvalue);
        closetag();
    }
    /**
     * Open a tag with a given set of attributes
     */
    public void printdata(String tag, @SuppressWarnings("rawtypes") Map attribs, Object bodyvalue){
        halfopentag(tag);
        for(Object key : attribs.keySet()){
            String v = GenericXMLParser.encode(attribs.get(key).toString());
            printAttribute(key.toString(), v);
        }
        print(">");
        printdata(bodyvalue);
        closetag();
    }
    /**
     * Print the tag, attributes, and the body.
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param body
     */
    public void printdata(String tag, String name1, Object value1,String name2, Object value2, Object bodyvalue){
        opentag(tag,name1,value1,name2,value2);
        printdata(bodyvalue);
        closetag();
    }
    /**
     * Print the tag, attributes, and the body.
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param body
     */
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            Object bodyvalue){
        opentag(tag,name1,value1,name2,value2,name3,value3);
        printdata(bodyvalue);
        closetag();
    }
    
    /**
     * Print the tag, attributes, and the body.
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param name4
     * @param value4
     * @param body
     */
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            String name4, Object value4, 
            Object bodyvalue){
        opentag(tag,
                name1,value1,
                name2,value2,
                name3,value3,
                name4,value4
                );
        printdata(bodyvalue);
        closetag();
    }
    
    /**
     * Print the tag, attributes, and the body.
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param name4
     * @param value4
     * @param name5
     * @param value5
     * @param body
     */
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            String name4, Object value4, 
            String name5, Object value5, 
            Object bodyvalue){
        opentag(tag,
                name1,value1,
                name2,value2,
                name3,value3,
                name4,value4,
                name5,value5
                );
        printdata(bodyvalue);
        closetag();
    }

    /**
     * Print the tag, attributes, and the body.
     * @param tag
     * @param name1
     * @param value1
     * @param name2
     * @param value2
     * @param name3
     * @param value3
     * @param name4
     * @param value4
     * @param name5
     * @param value5
     * @param body
     */
    public void printdata(String tag, 
            String name1, Object value1,
            String name2, Object value2, 
            String name3, Object value3, 
            String name4, Object value4, 
            String name5, Object value5, 
            String name6, Object value6, 
            Object bodyvalue){
        opentag(tag,
                name1,value1,
                name2,value2,
                name3,value3,
                name4,value4,
                name5,value5,
                name6,value6
                );
        printdata(bodyvalue);
        closetag();
    }

    /**
     * Print the tag, attributes, no
     * @param tag
     * @param name1
     * @param value
     * @param body
     */
    public void printdata(String tag, String name1, Object value, Object bodyvalue){
        opentag(tag,name1,value);
        printdata(bodyvalue);
        closetag();
    }
    
    public XMLPrinter(Writer out ){
        pw = new PrintWriter(out,true);
    }
    
    public XMLPrinter(OutputStream stream ){
        pw = new PrintWriter(stream,true);
    }
    /**
     * Opens an output stream, and puts out the surrounding root tag.
     * @param tag  The surrounding tag.  No XML syntax should be specified.
     * @param stream
     */
    public XMLPrinter(String tag, OutputStream stream){
        pw = new PrintWriter(stream,true);
        opentag(tag);
    }
    
    /**
     * Closes all open tags, close the file.
     *
     */
    public void close(){
        for(int i = tagStack.size()-1; i>=0;i--){
            closetag();
        }
    }
    
    /**
     * Just a helper to print an error during the generation of an XML file
     * @param errorMsg The error message to be printed.  Put into an <error> tag in the XML.
     */
    public void print_error(String errorMsg){
        if(intextbody)closetag();
        opentag("error");
        print(GenericXMLParser.encode(errorMsg));
        closetag();
    }
    
}
