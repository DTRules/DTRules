package com.dtrules.testsupport;

import java.io.IOException;
import java.util.HashMap;

import com.dtrules.interpreter.RBoolean;
import com.dtrules.xmlparser.AGenericXMLParser;

public class LoadSettings extends AGenericXMLParser {

	ITestHarness t;
	
	LoadSettings(ITestHarness t){
		this.t = t;
	}

	String getProperty(HashMap<String, String> attribs, String body){
		String sp = attribs.get("systemproperty");
		if(sp!=null){
			sp = System.getProperty(sp);
		}else{
			sp = "";
		}			
		return sp+body;		
	}

	
	public void endTag(String[] tagstk, int tagstkptr, String tag, String body,
			HashMap<String, String> attribs) throws Exception, IOException {
	
		if(tag.equals("trace")){
			t.setTrace(RBoolean.getRBoolean(body).booleanValue());
		}else if(tag.equals("console")){
			t.setConsole(RBoolean.getRBoolean(body).booleanValue());			
		}else if(tag.equals("numbered")){
			t.setNumbered(RBoolean.getRBoolean(body).booleanValue());			
		}else if(tag.equals("verbose")){
			t.setVerbose(RBoolean.getRBoolean(body).booleanValue());			
		}else if(tag.equals("coverage")){
			t.setCoverageReport(RBoolean.getRBoolean(body).booleanValue());			
		}else if(tag.equals("path")){
			t.setPath(getProperty(attribs, body));
		}else if(tag.equals("rulesdirectorypath")){
			t.setRulesDirectoryPath(getProperty(attribs, body));
		}else if(tag.equals("rulesetname")){
			t.setRuleSetName(body);
		}else if(tag.equals("decisiontablename")){
			t.setDecisionTableName(body);
		}else if(tag.equals("rulesdirectoryfile")){
			t.setRulesDirectoryFile(body);
		}
		
	}
}
