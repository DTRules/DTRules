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


package com.dtrules.testsupport;


import java.io.ByteArrayOutputStream;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.HashMap;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;
import com.dtrules.xmlparser.XMLTree;
import com.dtrules.xmlparser.XMLTree.Node;


/**
 * Compares the differences between two versions of a Rule Set.  Generally
 * we compare a Rule Set in development against the Rule set currently 
 * deployed. 
 *
 */
public class ChangeReport {
    
    RulesConfig rules1 = null;
    RulesConfig rules2 = null;
   
    private String checkStructure [] = {
           "initial_actions",
           "contexts",
           "conditions",
           "actions", 
           "policy_statements"
    };
   
    private String checkValue[] = { 
           "type", 
           "context_postfix",
           "initial_action_postfix", 
           "condition_postfix",
           "condition_column",
           "action_column",
           "policy_statement"
    };    
    
    public class RulesConfig {
        protected String       ruleSetName   = "";
        protected String       path          = "";       // The path to the configuration file
        protected String       dtRulesConfig = "";       // The configuration file name.
        protected String       xml           = "";       // Subpath to the xml directory
        protected String       description   = "";
        protected String       dtsName       = "";
        protected String       eddName       = "";
        protected String       mappingName   = "";
        protected InputStream  dts           = null;
        protected InputStream  edd           = null;
        protected InputStream  mapping       = null;
        protected Node         dtsRoot       = null;
        protected Node         eddRoot       = null;
        protected Node         mappingRoot   = null;
        
        
        private Node loadTree(InputStream s){
            try {
                Node root = XMLTree.BuildTree(s,false,false);
                return root;
            } catch (Exception e) {
                return null;
            }
            
        }
        
        /**
         * @return the dtsRoot
         */
        public Node getDtsRoot() {
            if(dtsRoot == null){
               dtsRoot = loadTree(dts); 
            }
            return dtsRoot;
        }
        /**
         * @param dtsRoot the dtsRoot to set
         */
        public void setDtsRoot(Node dtsRoot) {
            this.dtsRoot = dtsRoot;
        }
        /**
         * @return the eddRoot
         */
        public Node getEddRoot() {
            if(eddRoot == null){
                eddRoot = loadTree(edd);
            }
            return eddRoot;
        }
        /**
         * @param eddRoot the eddRoot to set
         */
        public void setEddRoot(Node eddRoot) {
            this.eddRoot = eddRoot;
        }
        /**
         * @return the mappingRoot
         */
        public Node getMappingRoot() {
            if(mappingRoot == null){
                mappingRoot = loadTree(mapping);
            }
            return mappingRoot;
        }
        /**
         * @param mappingRoot the mappingRoot to set
         */
        public void setMappingRoot(Node mappingRoot) {
            this.mappingRoot = mappingRoot;
        }
        RulesConfig(String ruleSetName, String description, String path, String dtRulesConfig) throws Exception {
            setRuleSetName(ruleSetName);
            setDescription(description);
            setPath(path);
            setDtRulesConfig(dtRulesConfig);
            
            InputStream s = new FileInputStream(this.path+this.dtRulesConfig);
            GenericXMLParser.load(s,new parseConfig(ruleSetName, this));
        }
        /**
         * The Rule Set we are going to compare.
         * @param ruleSetName
         */
        public void setRuleSetName(String ruleSetName){
            this.ruleSetName=ruleSetName.trim();
        }
        
        /**
         * Set the path.  If the path changes, we attempt to update the Inputstreams.  
         * Maybe not necessary, but it avoids restricting the order of tags in the
         * Rules Engine configuration file.
         * 
         * @param path
         */
        public void setPath(String path){
            this.path = path.trim();
            if(!this.path.endsWith("/") && !this.path.endsWith("\\")){
                this.path += "/";
            }
           
            setDtsName(dtsName);
            setEddName(eddName);
            setMappingName(mappingName);
        }
        
        public void setDtRulesConfig(String dtRulesConfig){
            this.dtRulesConfig = dtRulesConfig.trim();
        }
        
        public void setDescription(String description){
            this.description = description;
        }
        /**
         * Set the xml subpath.  If the path changes, we attempt to update the Inputstreams.  
         * Maybe not necessary, but it avoids restricting the order of tags in the
         * Rules Engine configuration file.
         * 
         * @param xmlSubPath
         */
        public void setXml(String xmlSubPath){
            this.xml = xmlSubPath.trim();
            if(!this.xml.endsWith("/") && !this.xml.endsWith("\\")){
                this.xml += "/";
            }
            if(this.xml.startsWith("/")|| this.xml.startsWith("\\")){
                this.xml = this.xml.substring(1);
            }
            
            setDtsName(dtsName);
            setEddName(eddName);
            setMappingName(mappingName);
        }
        
        /**
         * Returns a null if the file could not be openned
         * @param file
         * @return
         */
        private InputStream open(String file){
            try{
                InputStream s = new FileInputStream(path+xml+file);
                return s;
            }catch(FileNotFoundException e){
                return null;
            }
        }
        
        public void setDtsName(String name){
            dtsName = name.trim();
            if(dtsName.length()>0){
                dts = open(dtsName);
            }
        }
        
        public void setEddName(String name){
            eddName = name.trim();
            if(eddName.length()>0){
                edd = open(eddName);
            }
        }

        public void setMappingName(String name){
            mappingName = name.trim();
            if(mappingName.length()>0){
                mapping = open(mappingName);
            }
        }

        protected class parseConfig implements IGenericXMLParser {
            RulesConfig config;
            String      ruleSetName;
            boolean     found = false;      // Have we found the rule set ?
            
            parseConfig(String ruleSetName, RulesConfig config){
                this.ruleSetName    = ruleSetName;
                this.config         = config;
            }
            
            public void beginTag(String[] tagstk, int tagstkptr, String tag,
               HashMap<String, String> attribs) throws IOException, Exception {
                if(tag.equals("RuleSet") && attribs.get("name").equalsIgnoreCase(ruleSetName)){
                    found = true;
                }
            }

            public void endTag(String[] tagstk, int tagstkptr, String tag,
               String body, HashMap<String, String> attribs) throws Exception,
               IOException {
                if(found){                                          // Means we are in our
                    if(tag.equals("RuleSetFilePath")){              //    rule set.
                        config.setXml(body);        
                    }else if(tag.equals("Decisiontables")){
                        String file = attribs.get("name");
                        config.setDtsName(file);
                    }else if(tag.equals("Entities")
                            || tag.equals("EDD")){
                        String file = attribs.get("name");
                        config.setEddName(file);
                    }else if(tag.equals("Map")){
                        String file = attribs.get("name");
                        config.setMappingName(file);
                    }else if (tag.equals("RuleSet")){               // Leaving our rule set?
                        found = false;                              //   kill the found!
                    }
                }
            }

            public boolean error(String v) throws Exception {
               return true;
            }
            
        }

    }
    
    
   
   /**
    * Provides a change report between to Rules Engine Configurations.  The 
    * first is the configuration that has presumably changed.  The second is 
    * the reference configuration, in general. 
    * 
    * @param path1
    * @param dtRulesConfig1
    * @param description1
    * @param path2
    * @param dtRulesConfig2
    * @param description2
    */
   public ChangeReport (
           String ruleSetName,
           String path1, 
           String dtRulesConfig1, 
           String description1,
           String path2, 
           String dtRulesConfig2,
           String description2){
   
       try{
           rules1 = new RulesConfig(ruleSetName, description1,path1,dtRulesConfig1);
           rules2 = new RulesConfig(ruleSetName, description2,path2,dtRulesConfig2);
       }catch(Exception e){
           throw new RuntimeException(e.toString());
       }
   }
    
   public RulesConfig getRules1() {
       return rules1;
   }
    
   public RulesConfig getRules2() {
       return rules2;
   }
   /**
    * Resets file pointers to all the decision tables, edds, and mapping
    * files.  This way, users can use this object to do other, unanticipated
    * checks of rule sets and between rulesets.
    */
   public void reset(){
       rules1.setPath(rules1.path);
       rules2.setPath(rules2.path);
   }
   
   private static boolean contains(String [] list, String s){
       for(String v : list){
           if(v.equals(s))return true;
       }
       return false;
   }
    
   public static ArrayList<Node> findtables(Node root){
       ArrayList<Node> decisiontables = new ArrayList<Node>();
       findnodes("decision_table",root,decisiontables);
       return decisiontables;
   }
   
   /**
    * Find all the nodes in a tree with a particular tag 
    * @param tag        The tag to search for
    * @param node       The node in the tree to search
    * @param list       List of nodes with tag names that match.  If null, the
    *                   list is allocated for you.
    */
   public static ArrayList<Node> findnodes(String tag, Node node, ArrayList<Node> list){
       if(list == null){
           list = new ArrayList<Node>();
       }
       if(node.getName().equals(tag)){
           list.add(node);
       }else{
           for(Node n : node.getTags()){
               findnodes(tag,n,list);
           }
       }
       return list;
   }
   
   public static ArrayList<Node> findentities(Node root){
       ArrayList<Node> entities = new ArrayList<Node>();
       findnodes("entity", root,entities);
       return entities;
   }
     
   /**
    * It is interesting to find what nodes have been added, and which have been deleted.
    * This routine does that for us easily. We look for the nodes that have a matching 
    * set of attributes in both lists.  If a node in the first list isn't in the second list (i.e.
    * if we are matching on an attribute "name", and a node in list one has a name X, and
    * no node in list2 has a name X, then we remove it from list one).
    * 
    * We remove nodes that cannot be compared.  It leaves us with only nodes in our lists
    * common to both sources.
    * @attrib
    * @param t1
    * @param t2
    * @param message
    * 
    */
   public static ArrayList<Node> findMissingNodes(String attribs[], ArrayList<Node> dts1, ArrayList<Node> dts2){
       ArrayList<Node> missing = new ArrayList<Node>();
       for(int i = dts1.size()-1; i>=0; i--){
           Node t1 = dts1.get(i);
           Node match = null;
           for(Node t2: dts2){
               
               boolean allmatch = true;
               for(String attrib : attribs){
                   if(!t1.getAttributes().get(attrib).equals(t2.getAttributes().get(attrib))){
                       allmatch = false;
                       break;
                   }
               }
               if(allmatch){
                   match = t2;
                   break;
               }
           }
           if(match == null){
               missing.add(t1);         // Add the missing node to the list of missing nodes.
               dts1.remove(i);
           }else{
               dts2.remove(match);      // Now this insures dts2 is in the same order
               dts2.add(0,match);       // as dts1!  We do assume no duplicate tables...
           }
       }
       return missing;
   }
   
   /**
    * Helper function to simplify the following function.
    * @param t
    * @param attrib
    * @return
    */
   private static String getName(Node t, String attrib){
       String tname;
       if(attrib == null){
           tname = t.findTag("table_name").getBody();
       }else{
           tname = t.getAttributes().get(attrib); 
       }
       return tname;
   }

   /**
    * It is interesting to find what nodes have been added, and which have been deleted.
    * This routine does that for us easily. We look for the nodes that have a matching 
    * attribute in both lists.  If a node in the first list isn't in the second list (i.e.
    * if we are matching on an attribute "name", and a node in list one has a name X, and
    * no node in list2 has a name X, then we remove it from list one).
    * 
    * We remove nodes that cannot be compared.  It leaves us with only nodes in our lists
    * common to both sources.
    * @attrib
    * @param t1
    * @param t2
    * @param message
    * 
    */
   public static ArrayList<Node> findMissingNodes(String attrib, String tag, ArrayList<Node> dts1, ArrayList<Node> dts2){
       ArrayList<Node> missing = new ArrayList<Node>();
       for(int i = dts1.size()-1; i>=0; i--){
           Node t1 = dts1.get(i);
           String t1name = getName(t1,attrib);
           Node match = null;
           for(Node t2: dts2){
               String t2name = getName(t2,attrib);
               
               if(t1name.equals(t2name)){
                   match = t2;
                   break;
               }
           }
           if(match == null){
               missing.add(t1);         // Add the missing node to the list of missing nodes.
               dts1.remove(i);
           }else{
               dts2.remove(match);      // Now this insures dts2 is in the same order
               dts2.add(0,match);       // as dts1!  We do assume we might have duplicates ... remove all matches
               for(int j = dts2.size()-1; j >0; j--){
                   String t2name = getName(dts2.get(j),attrib);
                   if(t1name.equals(t2name)){
                       dts2.remove(j);
                   }
               }
           }
       }
       return missing;
   }
   /**
    * Bubble sort with quick out and a fence ... Very fast on previously sorted data
    * and pretty fast on nearly sorted data.
    * @param array
    */
   public static void sort(boolean ascending, Object [] array){
       int fence = array.length-1;
       boolean sorted = false;
       for(int i=0; i < fence && !sorted ; i++){
           for(int j = 0; j < fence-i; j++){
               if( array[j].toString().compareTo(array[j+1].toString())>0 ^ !ascending){
                   sorted = false;
                   Object hld = array[j];
                   array[j]   = array[j+1];
                   array[j+1] = hld;
               }
           }
       }
   }
   
   
   String getAttributes(Node n){
       Object attribs[] = n.getAttributes().keySet().toArray();
       sort(true,attribs);
       StringBuffer sb = new StringBuffer();
       for(Object s : attribs){
           sb.append(s); 
           sb.append(" = "); 
           sb.append(n.getAttributes().get(s)+" ");
       }
       return sb.toString();
   }
   
   public void compare(XMLPrinter report) throws Exception {
       report.opentag("changeReport");
           report.opentag("header");
               report.printdata("File1",rules1.description);
               report.printdata("File2",rules2.description);
           report.closetag();
           compareDecisionTables(report);
           compareEDD(report);
           compareMapping(report);
       report.close();
   }
   
   public void compare(OutputStream reportStream) throws Exception {
       XMLPrinter report = new XMLPrinter(reportStream);
       compare(report);
   }
   
   public String compare() throws Exception {
       ByteArrayOutputStream buff = new ByteArrayOutputStream();
       compare(buff);
       return buff.toString();
   }
   
   public String compareDecisionTables() throws Exception{
       ByteArrayOutputStream buff = new ByteArrayOutputStream();
       compareDecisionTables(buff);
       return buff.toString();
   }
   
   public void compareDecisionTables(OutputStream reportStream) throws Exception {
       XMLPrinter report = new XMLPrinter(reportStream);
       compareDecisionTables(report);
   }
   
   public void compareDecisionTables(XMLPrinter report) throws Exception {
       report.opentag("decisionTables");       

       if(rules1.dts == null ){ 
           report.printdata("error","decision tables for "+rules1.description+" were not found");
       }
       if(rules2.dts == null ){
           report.printdata("error","decision tables for "+rules2.description+" were not found");
       }
       if(rules1.dts == null || rules2.dts == null){
           report.closetag();
           return;
       }

       Node root1 = rules1.getDtsRoot();
       Node root2 = rules2.getDtsRoot();


       ArrayList<Node> decisionTables1 = findtables(root1);
       ArrayList<Node> decisionTables2 = findtables(root2);

       ArrayList<Node> missing;
       missing = findMissingNodes((String) null, "table_name", decisionTables1,decisionTables2);
       {for(Node m : missing){ report.printdata("NewTable",m.findTag("table_name").getBody());}}
       missing = findMissingNodes((String) null, "table_name", decisionTables2,decisionTables1);
       {for(Node m : missing){ report.printdata("DeletedTable",m.findTag("table_name").getBody());}}

       ArrayList<Integer> differences = new ArrayList<Integer>();

       report.opentag("modified_tables");
       for(int i = 0; i < decisionTables1.size();  i++){
           String name1 = decisionTables1.get(i).findTag("table_name").getBody();
           if(  !decisionTables1.get(i).absoluteMatch(decisionTables2.get(i),false) ){
               differences.add(i);
               report.printdata("table_name",name1);
           }
       }
       report.closetag();

       int executionChanges = 0;
       report.opentag("tables_with_changes_in_execution");
       for(int i : differences){
           if(executionChanged(decisionTables1.get(i),decisionTables2.get(i))){
               String name1 = decisionTables1.get(i).findTag("table_name").getBody();
               report.printdata("table_name",name1);
               executionChanges ++;
           }
       }
       report.closetag();
       
       if(executionChanges>0){
           report.printdata("execution",executionChanges + " Decision Table(s) have had their execution modified");
       }else{
           report.printdata("execution","Changes have not been made that effect execution");
       }
       
       report.closetag();
   }
      
   /**
    * Returns true if the execution behavior of a decision table has changed
    * @param n1
    * @param n2
    * @return
    */
   boolean executionChanged(Node n1, Node n2){
       if(contains(checkStructure,n1.getName())){
           if(n1.getTags().size() != n2.getTags().size()){
               return true;
           }
       }else if ( contains(checkValue,n1.getName())){
           if(n1.compareToNode(n2,false) != Node.MATCH.match) {
               return true;
           }
       }
       for(int i =0; i < n1.getTags().size() || i < n2.getTags().size(); i++){
           if(i >= n1.getTags().size()){
               if(contains(checkStructure,n2.getTags().get(i).getName()) 
                  || contains(checkValue,n2.getTags().get(i).getName())){
                   return true;
               }
           }
           if(i >= n2.getTags().size()){
               if(contains(checkStructure,n1.getTags().get(i).getName()) 
                  || contains(checkValue,n1.getTags().get(i).getName())){
                   return true;
               }
           }
           if(i >= n1.getTags().size() && i >= n2.getTags().size()){
	           if(executionChanged(n1.getTags().get(i),n2.getTags().get(i))){
	               return true;
	           }
           }
       }
       return false;
   }
  
   public String compareEDD() throws Exception{
       ByteArrayOutputStream buff = new ByteArrayOutputStream();
       compareEDD(buff);
       return buff.toString();
   }
   
   
   public void compareEDD(OutputStream reportStream) throws Exception {
       XMLPrinter report = new XMLPrinter(reportStream);
       compareEDD(report);
   }
   
   public void compareEDD(XMLPrinter report) throws Exception {
       
       report.opentag("edd");
       
       if(rules1.edd == null || rules2.edd == null){
           if(rules1.edd == null ){ 
               report.printdata("error","the EDD for "+rules1.description+" was not found");
           }
           if(rules2.edd == null ){
               report.printdata("error","the EDD for "+rules2.description+" was not found");
           }       
           report.closetag();
           return;
       }
       
       Node root1 = rules1.getEddRoot();
       Node root2 = rules2.getEddRoot();
       
       ArrayList<Node> entities1 = findentities(root1);
       ArrayList<Node> entities2 = findentities(root2);
       
       if(entities1.size()==0){
           report.printdata("empty","the EDD for "+rules1.description+" has no entitiees");
       }
       if(entities2.size()==0){
           report.printdata("empty","the EDD for "+rules2.description+" has no entitiees");
       }
       {
           ArrayList<Node> missing;
           missing = ChangeReport.findMissingNodes("name",(String)null,entities1,entities2);
           {for(Node m : missing){ report.printdata("NewEntity",m.getAttributes().get("name"));}}
           missing = ChangeReport.findMissingNodes("name",(String)null,entities2,entities1);
           {for(Node m : missing){ report.printdata("DeletedEntity",m.getAttributes().get("name"));}}
           for(int i=0;i<entities1.size();i++){
               ArrayList<Node> fields1 = findnodes("field",entities1.get(i),null);
               ArrayList<Node> fields2 = findnodes("field",entities2.get(i),null);
               missing = ChangeReport.findMissingNodes("name",(String)null,fields1,fields2);
               {for(Node m : missing){ 
                   report.printdata("NewAttribute",
                           "entity1", entities1.get(i).getAttributes().get("name"), 
                           m.getAttributes().get("name"));}}
               missing = ChangeReport.findMissingNodes("name",(String)null,entities2,entities1);
               {for(Node m : missing){ 
                   report.printdata("DeletedAttribute",
                           "entity2",entities2.get(i).getAttributes().get("name"),
                           m.getAttributes().get("name"));}}
               for(int j=0; j<fields1.size();j++){
                   if(!fields1.get(j).absoluteMatch(fields2.get(j),false)){
                       report.printdata("AttributeChanged",
                               "entity",    entities1.get(i).getAttributes().get("name"),
                               "attribute", fields1.get(j).getAttributes().get("name"),
                               null);
                   }
               }
           }
       } 
       report.closetag();
   }
   
   
   public String compareMapping() throws Exception{
       ByteArrayOutputStream buff = new ByteArrayOutputStream();
       compareMapping(buff);
       return buff.toString();
   }
   
   
   public void compareMapping(OutputStream reportStream) throws Exception {
       XMLPrinter report = new XMLPrinter(reportStream);
       compareMapping(report);
   }
   
   
   public boolean compareMapping(XMLPrinter report) throws Exception {
       boolean differences = false;
       
       int buffIndex = report.bufferOutput();
       report.opentag("mapping");
       
       if(rules1.mapping == null ){ 
           report.printdata("error","the Mapping File for "+rules1.description+" was not found");
       }
       if(rules2.mapping == null ){
           report.printdata("error","the Mapping File for "+rules2.description+" was not found");
       }
       if(rules1.mapping == null || rules2.mapping == null){
           report.closetag();
           return differences;
       }
   
       Node root1 = rules1.getMappingRoot();
       Node root2 = rules2.getMappingRoot();
       {        
           ArrayList<Node> attributes1 = findnodes("setattribute",root1,null);
           ArrayList<Node> attributes2 = findnodes("setattribute",root2,null);
           
           if(attributes1.size()==0){
               report.printdata("empty","no attributes mapped in map file for "+rules1.description);
           }
           if(attributes2.size()==0){
               report.printdata("empty","no attributes mapped in map file for "+rules2.description);
           }
           
           XMLTree.sortByAttribute(true,attributes1,"enclosure");
           XMLTree.sortByAttribute(true,attributes1,"RAttribute");
           XMLTree.sortByAttribute(true,attributes1,"tag");
           XMLTree.sortByAttribute(true,attributes2,"enclosure");
           XMLTree.sortByAttribute(true,attributes2,"RAttribute");
           XMLTree.sortByAttribute(true,attributes2,"tag");
           
           String attribs2match[] = {"tag","RAttribute","enclosure"};
           ArrayList<Node> dups;
           dups = XMLTree.removeDuplicates(attributes1, attribs2match);
           if(dups.size()>0)differences = true;
           {for(Node m : dups){ report.printdata("DuplicateNode",m.getAttributes().get("tag in source")+" for "+
                   m.getAttributes().get("enclosure")+"."+m.getAttributes().get("RAttribute"));}}
           dups = XMLTree.removeDuplicates(attributes2, attribs2match);
           if(dups.size()>0)differences = true;
           {for(Node m : dups){ report.printdata("DuplicateNode",m.getAttributes().get("tag in reference ")+" for "+
                   m.getAttributes().get("enclosure")+"."+m.getAttributes().get("RAttribute"));}}
                        
           {
               ArrayList<Node> missing;
               missing = ChangeReport.findMissingNodes(attribs2match,attributes1,attributes2);
               if(missing.size()>0)differences = true;
               {for(Node m : missing){ report.printdata("NewMappings",m.getAttributes().get("tag")+" now maps to "+
                       m.getAttributes().get("enclosure")+"."+m.getAttributes().get("RAttribute"));}}
               missing = ChangeReport.findMissingNodes(attribs2match,attributes2,attributes1);
               if(missing.size()>0)differences = true;
               {for(Node m : missing){ report.printdata("DeletedMappings",m.getAttributes().get("tag")+" no longer maps to "+
                       m.getAttributes().get("enclosure")+"."+m.getAttributes().get("RAttribute"));}}
           }
           
           ArrayList<Integer> diffs = new ArrayList<Integer>(); 
           
           if(attributes1.size() > 0 ){
               report.opentag("ModifiedMappings");
               for(int i = 0; i < attributes1.size() && i < attributes2.size();  i++){
                   HashMap <String,String> attribs = attributes1.get(i).getAttributes();
                   if(  !attributes1.get(i).absoluteMatch(attributes2.get(i),false) ){
                       String name1 = attribs.get("tag")+" maps "+attribs.get("enclosure")+"."+attribs.get("RAttribute")+" (now type "+attribs.get("type")+")";
                       diffs.add(i);
                       differences = true;
                       report.printdata("MapTypeChanged",name1);
                   }
               }
               report.closetag();   
           } 
       }
       
       ArrayList<Node> datamaps1 = findnodes("do2entitymap",root1,null);
       ArrayList<Node> datamaps2 = findnodes("do2entitymap",root2,null);
       
       {
           ArrayList<Node> missing;
           missing = ChangeReport.findMissingNodes("class",(String)null,datamaps1,datamaps2);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("NewDataObjectMappings",m.getAttributes().get("class"));}}
           missing = ChangeReport.findMissingNodes("class",(String)null,datamaps2,datamaps1);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("DeletedDataObjectMappings",m.getAttributes().get("class"));}}
       }
       
       if(datamaps1.size() > 0 ){
           report.opentag("ModifiedDataMaps");
           for(int i = 0; i < datamaps1.size();  i++){
               if(  !datamaps1.get(i).absoluteMatch(datamaps2.get(i),false) ){
                   String name1 = datamaps1.get(i).getAttributes().get("class");
                   report.printdata("dataObject",name1);
                   differences = true;
               }
           }
           report.closetag();   
       }
       
       ArrayList<Node> entitydefs1 = findnodes("entity",root1,null);
       ArrayList<Node> entitydefs2 = findnodes("entity",root2,null);
       
       {
           ArrayList<Node> missing;
           missing = ChangeReport.findMissingNodes("name",(String)null,entitydefs1,entitydefs2);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("NewEntityDeclarations",m.getAttributes().get("class"));}}
           missing = ChangeReport.findMissingNodes("name",(String)null,entitydefs2,entitydefs1);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("DeletedEntityDeclarations",m.getAttributes().get("class"));}}
       }
       
       if(entitydefs1.size() > 0 ){
           report.opentag("ModifiedEntityDeclarations");
           for(int i = 0; i < datamaps1.size();  i++){
               if(  !entitydefs1.get(i).absoluteMatch(entitydefs2.get(i),false) ){
                   String name1 = entitydefs1.get(i).getAttributes().get("name");
                   report.printdata("dataObject",name1);
                   differences = true;
               }
           }
           report.closetag();   
       }
       
       ArrayList<Node> initialentities1 = findnodes("initialentity",root1,null);
       ArrayList<Node> initialentities2 = findnodes("initialentity",root2,null);
       
       {
           ArrayList<Node> missing;
           missing = ChangeReport.findMissingNodes("entity",(String)null,initialentities1,initialentities2);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("NewInitialEntities",m.getAttributes().get("entity"));}}
           missing = ChangeReport.findMissingNodes("entity",(String)null,initialentities2,initialentities1);
           if(missing.size()>0)differences = true;
           {for(Node m : missing){ report.printdata("DeletedInitialEntities",m.getAttributes().get("entity"));}}
       }       
       
       report.closetag();
       
       if(differences){
           report.writeBuffer(buffIndex);
       }else{
           report.deleteBuffer(buffIndex);
           report.opentag("mapping");
           report.closetag();
       }
       return differences;
   }
}
