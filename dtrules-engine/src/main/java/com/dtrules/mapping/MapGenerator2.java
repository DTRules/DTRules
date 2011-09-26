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

import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.lang.reflect.InvocationTargetException;
import java.util.ArrayList;
import java.util.HashMap;

import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * This Generator works with the new Format for EDD's which defines Entities and fields 
 * within those entities
 */
@SuppressWarnings({"unchecked"})
public class MapGenerator2 implements IGenericXMLParser, IMapGenerator  {
    String[]                    tagstk;
    int                         tagstkptr;
    String                      tag;
    HashMap<String,String>      attribs;
    String                      body;

    XMLPrinter                  out     = null;
    String                      mapping = null;
    
    /** 
     * These are the list of entities for which at least one
     * Attribute will be set by this mapping file.
     */
    ArrayList <String> entities     = new ArrayList<String>();
    ArrayList <String> unreferenced = new ArrayList<String>();
    /**
     * Returns true if the particular mapping here is one of the
     * input sources for this attribute.
     */
    boolean inputMatch(String inputs){
        /** Deal quickly with the trival case **/
        if(inputs.trim().length()==0) return false;
        String [] inputlist = inputs.split("[\\s]");
        for(String input : inputlist){
            if(input.equalsIgnoreCase(mapping))return true;
        }
        return false;
    }    
    String entity_name;
    String entity_access;
    String entity_comment;

    public void begin_entity () {
        entity_name    = (String) attribs.get("name");
        entity_access  = (String) attribs.get("access");
        entity_comment = (String) attribs.get("comment");
        out.comment(entity_name);
    }

    
    public void end_field(){
        String attribute = (String) attribs.get("name");
        String type      = (String) attribs.get("type");
        String subtype   = (String) attribs.get("subtype");
        String input     = (String) attribs.get("input");
        String entity    =  entity_name;
        
        if(inputMatch(input) && !type.equalsIgnoreCase("array")){
            out.printdata("setattribute",
                    "tag"        ,attribute ,
                    "RAttribute" ,attribute,
                    "enclosure"  ,entity,
                    "type"       ,type,
                    "subtype"    ,subtype,
                    null);
            if(!entities.contains(entity)){
                entities.add(entity);
            }else{
                if(unreferenced.contains(entity)){
                    unreferenced.remove(entity);
                }
            }
        }else{
            if(!entities.contains(entity)){
                entities.add(entity);
                if(!unreferenced.contains(entity)){
                    unreferenced.add(entity);
                }
            }    
        }
    }

    public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap attribs) throws IOException, Exception {
        this.tagstk    = tagstk;
        this.tagstkptr = tagstkptr;
        this.tag       = tag.toLowerCase();
        this.attribs   = attribs;
        this.body      = null;

        try {
            this.getClass().getMethod("begin_"+tag, (Class[])null).invoke(this,(Object [])null);
        } catch (IllegalArgumentException e) {
            e.printStackTrace();
        } catch (SecurityException e) {
            e.printStackTrace();
        } catch (IllegalAccessException e) {
            e.printStackTrace();
        } catch (InvocationTargetException e) {
            System.out.println("Error thrown in processing a Begin Tag");
            System.out.println(e.getCause().getMessage());
        } catch (NoSuchMethodException e) {
            //Ignore tags we don't know.
        }
    }

    
    /* (non-Javadoc)
     * @see com.dtrules.xmlparser.IGenericXMLParser#endTag(java.lang.String[], int, java.lang.String, java.lang.String, java.util.HashMap)
     */
    public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap attribs) throws Exception, IOException {
        this.tagstk    = tagstk;
        this.tagstkptr = tagstkptr;
        this.tag       = tag;
        this.attribs   = attribs;
        this.body      = body;
        
        try {
            this.getClass().getMethod("end_"+tag, (Class[])null).invoke(this,(Object [])null);
        } catch (IllegalArgumentException e) {
            e.printStackTrace();
        } catch (SecurityException e) {
            e.printStackTrace();
        } catch (IllegalAccessException e) {
            e.printStackTrace();
        } catch (InvocationTargetException e) {
            System.out.println("Error thrown in processing a End Tag");
            System.out.println(e.getCause().getMessage());
        } catch (NoSuchMethodException e) {
            //System.out.println("Unknown: end_"+tag);
        }
    }
        
    /* Something required of the XML parser.
     * @see com.dtrules.xmlparser.IGenericXMLParser#error(java.lang.String)
     */
    public boolean error(String v) throws Exception {
        return true;
    }
    
    public void generateMapping(String mapping, String inputfile, String outputfile) throws Exception {
        FileInputStream input = new FileInputStream(inputfile);
        XMLPrinter      out   = new XMLPrinter(new FileOutputStream(outputfile));
        generateMapping(mapping,input,out);
    }
    /**
     * Given an EDD XML, makes a good stab at generating a Mapping file for a given
     * mapping source.  The mapping source is specified as an input in the input column
     * in the EDD.
     * @param mapping
     * @param input
     * @param out
     * @throws Exception
     */
    public void generateMapping(String mapping, InputStream input, XMLPrinter out)throws Exception {
        this.out     = out;
        this.mapping = mapping;
        out.opentag("mapping");
        out.opentag("XMLtoEDD");

        out.opentag("map");
        out.comment("                 ");
        out.comment(" Map Attributes  ");
        out.comment("                 ");

            GenericXMLParser.load(input,this);
            out.comment("                 ");
            out.comment(" Create Entities ");
            out.comment("                 ");
            for(String entity : entities){
                out.printdata("createentity",
                        "entity", entity,
                        "tag"   , entity,
                        "id"    , "id",
                        null);
            }
        out.closetag();
        
        out.comment("                 ");
        out.comment(" Entity List     ");
        out.comment("                 ");
        out.opentag("entities");
            for(String entity : entities) {
                out.printdata("entity","name",entity,"number","*",null);
            }
        out.closetag();

        out.comment("                 ");
        out.comment(" Initialization  ");
        out.comment("                 ");

        out.opentag("initialization");
        out.printdata("initialentity","entity",entities.get(0),"epush","true",null); 
            for(String entity : unreferenced){
                out.printdata("initialentity","entity",entity,"epush","true",null); 
            }
        out.closetag();
        
        out.close();
    }

    /**
     * @return the entities
     */
    public ArrayList<String> getEntities() {
        return entities;
    }
}
