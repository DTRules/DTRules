package com.dtrules.deploy;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.util.HashMap;
import java.util.Map;

import com.dtrules.interpreter.RName;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.AGenericXMLParser;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * This utility can be called from to strip all unneeded information from your EDD and
 * Decision Table files.  This reduces slightly the weight of deployed decision tables,
 * and removes the "source" information from the tables, making them harder to reverse
 * engineer.
 * 
 * @author Paul Snow
 *
 */
public class StripXML {

    static class StripItDt extends AGenericXMLParser {
        XMLPrinter out;
        String     tags[] = {"COMMENTS","File_Name", "TABLE_NUMBER","xls_file",
                "context_comment",        "context_description",
                "initial_action_comment", "initial_action_requirement", "initial_action_description", 
                "condition_comment",      "condition_requirement",      "condition_description",
                "action_comment",         "action_requirement",         "action_description",
                "policy_description"};
        
        StripItDt(OutputStream out, String tags[]){
            this.out = new XMLPrinter(out);
            if(tags != null){
                this.tags = tags;
            }
        }
        
        @Override
        public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap<String, String> attribs)
                throws IOException, Exception {
            out.opentag(tag,attribs);
        }
        
        @Override
        public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap<String, String> attribs)
                throws Exception, IOException {
            boolean writeIt = true;
            for(String s : tags){
                if (s.equals(tag)){
                    writeIt = false;
                    break;
                }
            }
            if(writeIt && body != null && body.length()>0){
                out.printdata(body);
            }
            out.closetag();
        }
        
    }

    static class StripItEDD extends AGenericXMLParser {
        XMLPrinter out;
        
        String     attributes[] = {"comment"};
        
        StripItEDD(OutputStream out, String attributes[]){
            this.out = new XMLPrinter(out);
            if(attributes != null){
                this.attributes = attributes;
            }
        }
        
        @Override
        public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap<String, String> attribs)
                throws IOException, Exception {
            for(String s : attributes){
                if(attribs.containsKey(s)){
                    attribs.put(s, "");
                }
            }
            out.opentag(tag,attribs);
        }
        
        @Override
        public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap<String, String> attribs)
                throws Exception, IOException {
            if(body != null && body.length()>0){
                out.printdata(body);
            }
            out.closetag();
        }
        
    }

    /**
     * Strips the XML of descriptions and comments, by default.  If you provide an array of tags,
     * it will strip the XML of those tags instead.
     * @param rd            Rules Directory
     * @param tags          List of Tags to strip from the Decision Tables.  Only removes the bodies 
     *                      of XML tags, but leaves the tags.
     * @param attributes    List of attributes to strip from the EDD.  
     * @throws Exception    May throw a number of exceptions.  No error handling done here.
     */
    static public void strip(RulesDirectory rd, String tags[], String attributes []) throws Exception {
        Map<RName, RuleSet> ruleSets = rd.getRulesets();
        
        for(RuleSet rs: ruleSets.values()){
            String edd = rs.getEDD_XMLName();
            String dt  = rs.getDT_XMLName();
            stripDT(rs.getFilepath(),dt, tags);
            stripEDD(rs.getFilepath(),edd, attributes);
        }
        
    }
    
    static private void stripDT(String path, String filename, String tags[]) throws Exception {
        File                f       = new File(path+filename);
        FileInputStream     fi      = new FileInputStream(f);
        String              sn      = filename.replaceAll(".xml", "_strip.xml");
        File                s       = new File(path+sn);
        FileOutputStream    fos     = new FileOutputStream(s);
        StripItEDD             strip   = new StripItEDD(fos, tags);
        
        GenericXMLParser.load(fi, strip);
        
        fi.close();
        fos.close();
    }

    static private void stripEDD(String path, String filename, String attributes[]) throws Exception {
        File                f       = new File(path+filename);
        FileInputStream     fi      = new FileInputStream(f);
        String              sn      = filename.replaceAll(".xml", "_strip.xml");
        File                s       = new File(path+sn);
        FileOutputStream    fos     = new FileOutputStream(s);
        StripItEDD             strip   = new StripItEDD(fos, attributes);
        
        GenericXMLParser.load(fi, strip);
        
        fi.close();
        fos.close();
    }

    
}
