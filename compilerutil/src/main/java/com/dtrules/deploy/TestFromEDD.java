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

public class TestFromEDD {


    static class ParseEDD extends AGenericXMLParser {
        XMLPrinter out;
                
        ParseEDD(OutputStream out){
            this.out = new XMLPrinter("test",out);
        }
        
        @Override
        public void beginTag(
        		String[] tagstk, 
        		int tagstkptr, 
        		String tag, 
        		HashMap<String, String> attribs)
                throws Exception {
        	String name = attribs.get("name");
        	String type = attribs.get("type");
        	if(name == null || name.length()==0){
        		if(tag !=null && tag.length()>0) out.opentag(tag, attribs);
        	}else{
        		if(type != null && type.length()>0){
        			out.opentag(name, "type", type);
        		}else{
        			out.opentag(name);
        		}
        	}
        }
        
        @Override
        public void endTag(
        		String[] tagstk, 
        		int tagstkptr, 
        		String tag, 
        		String body, 
        		HashMap<String, 
        		String> attribs)
                throws Exception, IOException {
            String b = attribs.get("default_value");
        	
        	if(b != null && b.length()>0){
                out.printdata(b);
            }
        	
        	if(tag!=null && tag.length()>0) out.closetag();
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
    static public void buildTest(RulesDirectory rd ) throws Exception {
        Map<RName, RuleSet> ruleSets = rd.getRulesets();
        
        for(RuleSet rs: ruleSets.values()){
            String edd = rs.getEDD_XMLName();
            parseEDD(rs.getFilepath(),edd);
        }
        
    }
    
    static private void parseEDD(String path, String filename) throws Exception {
        File                f       = new File(path+filename);
        FileInputStream     fi      = new FileInputStream(f);
        String              sn      = filename.replaceAll(".xml", "_test.xml");
        File                s       = new File(path+sn);
        FileOutputStream    fos     = new FileOutputStream(s);
        ParseEDD            strip   = new ParseEDD(fos);
        
        GenericXMLParser.load(fi, strip);
        
        fi.close();
        fos.close();
    }

    
}
