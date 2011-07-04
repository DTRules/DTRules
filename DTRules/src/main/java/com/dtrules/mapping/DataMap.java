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

package com.dtrules.mapping;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.Reader;
import java.io.StringReader;
import java.io.StringWriter;
import java.util.ArrayList;
import java.util.HashMap;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser2;
import com.dtrules.xmlparser.IXMLPrinter;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * 
 * The Data Map class holds data for later mapping into the Rules Engine
 * using the same mapping definition that can be used to upload XML to
 * an EDD.  This interface can also be used to write out XML to a file.
 * <br><br>
 * This isn't a complex implementation.  The idea is to collect all the
 * data we would have written to an XML file, then map it in the same
 * way with the same tag structure into the EDD.  And as a bonus, we
 * allow for the writing of the data out as XML for debugging or playback
 * purposes.
 *
 * @author Paul Snow
 *
 */
public class DataMap implements IXMLPrinter{
    Mapping             map;
    OutputStream        out         = null;
    ArrayList<XMLNode>  tagStack    = new ArrayList<XMLNode>();
    XMLNode             rootTag;
    IRSession           session;
    AutoDataMap         autoDataMap = null;
    
    /**
     * Applications using the DataMap facility have a choice to use the older DataMap 
     * interfaces for moving data into the Rules Engine, or the newer AutoDataMap
     * interfaces.  A call to createAutoDataMap creates an AutoDataMap instance if 
     * necessary, and returns the reference to the AutoDataMap instance.
     * @param session
     * @param mapName
     * @return
     * @throws RulesException
     */
    public AutoDataMap createAutoDataMap(String mapName) throws RulesException {
        if(autoDataMap == null){
            autoDataMap = session.getRuleSet().getAutoDataMap(session, mapName); 
        }
        return autoDataMap;
    }
    
    
    public AutoDataMap getAutoDataMap(){
        return autoDataMap;
    }
    
    /**
     * Constructor for the DataMap structure, which will be 
     * enclosed with the given tag.  
     * @param tag
     */
    public DataMap(IRSession session, String tag){
        opentag(tag);
        rootTag = top();
        this.session = session;
    }
    /**
     * Constructor that will write out the datamap as an XML to the 
     * given output stream.  
     * <br><br>
     * Now that we have a method to write out the Datamap to a file, there
     * is no need to write out the XML as the Datamap is being built.
     * 
     * @param map - Provides info on the DO's
     * @param tag
     * @param xmlOutputStream - Writes an XML file out if specified (not null).
     * @deprecated
     */
    public DataMap(IRSession session, Mapping map, String tag, OutputStream xmlOutputStream){
        this(session, tag);
        this.map        = map;
        this.out        = xmlOutputStream;
        this.session    = session;
    }
    
    /**
     * If created with an output stream, XML is generated and written
     * to the output stream.
     * <br><br>
     * Now that we have a method to write out the Datamap to a file, there
     * is no need to write out the XML as the Datamap is being built.
     * 
     * @deprecated
     * @param tag
     * @param xmlOutputStream
     */
    public DataMap(IRSession session, String tag, OutputStream xmlOutputStream){
        this(session, tag);
        this.out = xmlOutputStream;        
    }
    
    /**
     * Pull the XML representation back out from the DataMap
     * If noSpaces == true, then the result will not be pretty
     * printed, saving space.
     * @param noSpaces
     * @return
     */
    public String xmlPull(boolean noSpaces){
        StringWriter sw = new StringWriter();
        XMLPrinter xout = new XMLPrinter(sw);
        xout.setNoSpaces(noSpaces);
        DataMap.print(xout, rootTag);
        return sw.toString();
    }
    
    /**
     * Write out this DataMap as an XML file
     * @param out
     */
    public void print(OutputStream out){
        if(getAutoDataMap()==null){
            XMLPrinter xout= new XMLPrinter(out);
            DataMap.print(xout,rootTag);
        }else{
            getAutoDataMap().printDataLoadXML(out);
        }
    }
    /**
     * Write out a subtree of the DataMap into an XML output stream.  
     * I do not write out tags with no body (body == null) and no 
     * internal structure  (no tag.tags....).  This is because the 
     * load of the XML treats missing attributes with as having not 
     * responded at all.  Such attributes default to the default value 
     * specified in the EDD.
     * <br><br>
     * When we load the text version however, we can't have a tag
     * or we will not behave in the same way.  A null for a string
     * will become a zero length string (for example).
     * @param out
     */
    static public void print(XMLPrinter xout, XMLNode tag){
        if(tag.type() == XMLNode.Type.TAG){ 
                if(tag.getBody() == null && tag.getTags()!=null && tag.getTags().size()>0){
                   if(tag.getTag()!=null){
                       xout.opentag(tag.getTag(),tag.getAttribs());
                   }
                   for(XMLNode t : tag.getTags()){ 
                       print(xout,t);
                   }
                   if(tag.getTag()!=null){
                       xout.closetag();
                   }
                }else{
                   xout.opentag(tag.getTag(),tag.getAttribs()); 
                   xout.printdata(tag.getBody());
                   xout.closetag();
                }

        } else if (tag.type() == XMLNode.Type.COMMENT ){
                xout.comment(tag.getBody().toString());

        } else if (tag.type() == XMLNode.Type.HEADER ){
                xout.header(tag.getBody().toString());

        }
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
        return tagStack.get(i).getTag();
    }
    /**
     * Returns true if a tag (optionally with the given attribute and value,
     * should the attribute be not null) is in the current tag stack. 
     * @param tag
     * @param key_attribute
     * @param value
     * @return
     */
    public boolean isInContext(String tag, String key_attribute, Object value){
        XMLNode t = top();
        while(t!=null && t!=rootTag){
            if( t.getTag().equals(tag)                      && 
                (key_attribute == null ||
                        t.getAttribs().containsKey(key_attribute)   &&
                        (value==null || t.getAttribs().get(key_attribute).equals(value)))){
                return true;
            }
            t = t.getParent();
        }
        return false;
    }
    
    /**
     * If we are supposed to write out the datafile, do so here.  Otherwise this
     * method does any other cleanup necessary in the DataMap. 
     */
    public void close() {
        if(out!=null){
            print(out);
        }
    }

    /**
     * close the last tag.
     */
    public void closetag() {
        int index = tagStack.size()-1;
        if(index>=0){
            tagStack.remove(index);
        }
    }

    /**
     * Create a new tag
     * @param tag
     */
    private void newtag(String tag ){
        XMLNode t = top();
        if(t!=null && top().getBody()!=null){
            throw new RuntimeException("You can't have tags and body text within the same XML tag!");
        }
        XMLNode newtag = new XMLTag(tag,t);
        if(t!=null){
            t.addChild(newtag);
        }
        tagStack.add(newtag);
    }
    /**
     * Create a new header
     * @param header
     */
    private void header(String header){
        XMLNode t = top();
        if(t!=null && top().getBody()!=null){
            throw new RuntimeException("You can't have tags and body text within the same XML tag!");
        }
        XMLNode newheader = new XMLHeader();
        newheader.setBody(header);
        t.addChild(newheader);
    }

    private void comment(String comment){
        XMLNode t = top();
        if(t!=null && top().getBody()!=null){
            throw new RuntimeException("You can't have tags and body text within the same XML tag!");
        }
        XMLNode newcomment = new XMLComment();
        newcomment.setBody(comment);
        t.addChild(newcomment);
    }

    private XMLNode top(){
        if(tagStack.size()==0)return null;
        return tagStack.get(tagStack.size()-1);
    }
    
    private void addValue(String key, Object value){
        top().setAttrib(key, value);
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, 
                String name1, Object value1, 
                String name2, Object value2, 
                String name3, Object value3, 
                String name4, Object value4, 
                String name5, Object value5, 
                String name6, Object value6, 
                String name7, Object value7, 
                String name8, Object value8, 
                String name9, Object value9) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);
        addValue(name6, value6);
        addValue(name7, value7);
        addValue(name8, value8);
        addValue(name9, value9);
    }
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
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
                String name10,Object value10) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);
        addValue(name6, value6);
        addValue(name7, value7);
        addValue(name8, value8);
        addValue(name9, value9);
        addValue(name10,value10);
    }
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, 
                String name1, Object value1, 
                String name2, Object value2, 
                String name3, Object value3, 
                String name4, Object value4, 
                String name5, Object value5, 
                String name6, Object value6, 
                String name7, Object value7, 
                String name8, Object value8) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);
        addValue(name6, value6);
        addValue(name7, value7);
        addValue(name8, value8);
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, 
                String name1, Object value1, 
                String name2, Object value2, 
                String name3, Object value3, 
                String name4, Object value4, 
                String name5, Object value5, 
                String name6, Object value6, 
                String name7, Object value7) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);
        addValue(name6, value6);
        addValue(name7, value7);
    }
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1, String name2, Object value2, String name3, Object value3, String name4, Object value4, String name5, Object value5, String name6, Object value6) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);
        addValue(name6, value6);
        
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1, String name2, Object value2, String name3, Object value3, String name4, Object value4, String name5, Object value5) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        addValue(name5, value5);        
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1, String name2, Object value2, String name3, Object value3, String name4, Object value4) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
        addValue(name4, value4);
        
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1, String name2, Object value2, String name3, Object value3) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        addValue(name3, value3);
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1, String name2, Object value2) {
        newtag(tag);
        addValue(name1, value1);
        addValue(name2, value2);
        
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String, java.lang.String, java.lang.Object)
     */
    public void opentag(String tag, String name1, Object value1) {
        newtag(tag);
        addValue(name1, value1);        
    }
    
    
    /*
     * Open a tag with the given hashmap of values
     */
    @SuppressWarnings("unchecked")
    public void opentag(String tag, HashMap attribs){
        newtag(tag);
        for(Object key : attribs.keySet()){
            addValue((String)key,attribs.get(key));
        }
    }
    
    
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#opentag(java.lang.String)
     */
    public void opentag(String tag) {
        newtag(tag);
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#print_error(java.lang.String)
     */
    public void print_error(String errorMsg) {
        opentag("error","msg",errorMsg);
        closetag();
        
    }

    /* 
     * We write the data to the body of the top tag.  If multiple printdata() calls
     * are made, then everything is converted to a String and added together.
     */
    public void printdata(Object bodyvalue) {
        XMLNode t = top();
        if(t.getBody()==null){
            t.setBody(bodyvalue);
            return;
        }
        t.setBody(t.getBody().toString() + bodyvalue.toString());
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.Object)
     */
    public void printdata(String tag, Object bodyvalue) {
        opentag(tag);
        printdata(bodyvalue);
        closetag();
        
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.String, java.lang.Object, java.lang.Object)
     */
    public void printdata(String tag, String name1, Object value1, Object body) {
        opentag(tag,name1,value1);
        printdata(body);
        closetag();
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.Object)
     */
    public void printdata(String tag, String name1, Object value1,
            String name2, Object value2, Object bodyvalue) {
        opentag(tag,name1,value1, name2, value2);
        printdata(bodyvalue);
        closetag();
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.Object)
     */
    public void printdata(String tag, String name1, Object value1,
            String name2, Object value2, String name3, Object value3,
            Object bodyvalue) {
        opentag(tag,name1,value1, name2, value2, name3, value3);
        printdata(bodyvalue);
        closetag();
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.Object)
     */
    public void printdata(String tag, String name1, Object value1,
            String name2, Object value2, String name3, Object value3,
            String name4, Object value4, Object bodyvalue) {
        opentag(tag,name1,value1, name2, value2, name3, value3, name4, value4);
        printdata(bodyvalue);
        closetag();
    }

    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IXMLPrinter#printdata(java.lang.String, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.String, java.lang.Object, java.lang.Object)
     */
    public void printdata(String tag, String name1, Object value1,
            String name2, Object value2, String name3, Object value3,
            String name4, Object value4, String name5, Object value5,
            Object bodyvalue) {
        opentag(tag,name1,value1, name2, value2, name3, value3, name4, value4, name5, value5);
        printdata(bodyvalue);
        closetag();
        
    }

    /**
     * Return the Root XMLTag of the data structures
     * @return
     */
    public XMLNode getRootTag(){ return rootTag; }    
    /**
     * Read the attributes of a Data Object and populate an EDD with
     * those values.  This assumes that the class for the Data Object has
     * been mapped to an Entity in the Map file for a Rule Set
     * @param obj
     * @param tag
     */
    public void readDO(Object obj, String tag){
        String do_name = obj.getClass().getName();
        ArrayList<DataObjectMap> doMaps = map.dataObjects.get(do_name);
        DataObjectMap doMap = null;
        if(doMaps!=null) for(DataObjectMap DO : doMaps){
            if(DO.tag.equals(tag)){
                doMap = DO;
                break;
            }
        }
        if(doMap==null){
            throw new RuntimeException("Unknown Data Object: "+do_name);
        }    
        try {
            doMap.mapDO(this, obj);
        } catch (RulesException e) {
            throw new RuntimeException("Unknown Data Object: "+do_name);
        }
    }
    /**
     * Looks up the Key from the Data Object and generates the approprate
     * open tag structure.
     * @param obj
     * @param tag
     */
    public void opentag(Object obj, String tag){
        String do_name = obj.getClass().getName();
        ArrayList<DataObjectMap> doMaps = map.dataObjects.get(do_name);
        DataObjectMap doMap = null;
        if(doMaps != null)for(DataObjectMap DO : doMaps){
            if(DO.tag.equals(tag)){
                doMap = DO;
                break;
            }
        }
        if(doMap==null){
            throw new RuntimeException("Attempt to map data into the EDD using an Unknown Data Object: "+do_name);
        }
        doMap.OpenEntityTag(this, obj);
    }
    
    private class XmlLoader implements IGenericXMLParser2 {
        DataMap datamap;
        
        XmlLoader(DataMap datamap){
            this.datamap = datamap;
        }
        public void beginTag(String[] tagstk, int tagstkptr, String tag,
                HashMap<String, String> attribs) throws IOException, Exception {
            datamap.opentag(tag,attribs);
        }

        public void endTag(String[] tagstk, int tagstkptr, String tag,
                String body, HashMap<String, String> attribs) throws Exception,
                IOException {
            if(body!=null && body.length()>0){
                datamap.printdata(body);
            }
            datamap.closetag();
        }
        
        public boolean error(String v) throws Exception {
            return true;
        }
        
        public void comment(String comment) {
            datamap.comment(comment);
            
        }

        public void header(String header) {
            datamap.header(header);
            
        }
        
    }
    
    /**
     * Loads an XML File into a DataMap
     * @param xml
     */
    public void loadXML(InputStream xml) throws RulesException {
        XmlLoader xmlLoader = new XmlLoader(this);
        try{
            GenericXMLParser.load(xml, xmlLoader);
        }catch(Exception e){
            throw new RulesException("Bad XML","loadXML",e.toString());
        }
    }
    
    /**
     * Loads an XML File into a DataMap
     * @param xml
     */
    public void loadXML(Reader xml) throws RulesException {
        XmlLoader xmlLoader = new XmlLoader(this);
        try{
            GenericXMLParser.load(xml, xmlLoader);
        }catch(Exception e){
            throw new RulesException("Bad XML","loadXML",e.toString());
        }
    }
    
    /**
     * Loads an XML String into a DataMap
     * @param xml
     */
    public void loadXML(String xml) throws RulesException {
        loadXML(new StringReader(xml));
    }
}
