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
import java.util.*;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.util.regex.Pattern;
import java.io.IOException;
@SuppressWarnings({"unused"})
%%
%public
%class GenericXMLParser
%yylexthrow  Exception
%unicode
%line
%column
%int

%{
	// Start of User Funcitons
	
    private String 			sourcename 		= "unknown name";
    String            		tagstk []       = new String[1000];
    int               		tagstkptr       = 0;
    int               		statestk []     = new int[1000];
    int               		statestkptr     = 0;
    ArrayList<HashMap<String,String>> attribstk = new ArrayList<HashMap<String,String>>();
    HashMap<String,String>  attribs         = new HashMap<String,String>(2,1.0f);
    boolean           		printflg        = true;
    IGenericXMLParser2 		parser          = null;
    
    String            		body            = "";
    String            		currenttag      = "";
    String 		     		Source;

	public GenericXMLParser(String filename) throws FileNotFoundException {
		this(new FileInputStream(filename));
		sourcename = filename;
	}

    void pushTag(String tag){
       tagstk[tagstkptr++] = tag;
       currenttag = tag;
       attribstk.add(attribs);    // We save and restore the attribs hashmap
       attribs = new HashMap<String,String>(2,1.0f);
    }
    
    String popTag(String endtag) {
       attribs = (HashMap<String,String>) attribstk.remove(attribstk.size()-1);	
       if(tagstkptr<=0){
          System.err.print("Stack Underflow\n");
       }
       String tag = tagstk[--tagstkptr];
       if(!tag.equals(endtag)){
          System.err.print("Begin and End tags do not match:\n"+
                           "Begin tag: "+tag+"\n"+
                           "End tag:   "+endtag+"\n");
       }

       if(tagstkptr<=0){
          currenttag = "";
       }else{
          currenttag = tagstk[tagstkptr-1];
       }

       return tag;
    }

	public void setSource(String source){
	   this.sourcename=source;
	}

   class Shell implements IGenericXMLParser2 {
        IGenericXMLParser p;
        Shell(IGenericXMLParser p){this.p = p;}
        public void comment(String comment) {}
        public void header(String header) {}
        public void beginTag(String[] tagstk, int tagstkptr, String tag,
                HashMap<String, String> attribs) throws IOException, Exception {
            p.beginTag(tagstk, tagstkptr, tag, attribs);
        }
        public void endTag(String[] tagstk, int tagstkptr, String tag,
                String body, HashMap<String, String> attribs) throws Exception,
                IOException {
            p.endTag(tagstk, tagstkptr, tag, body, attribs);
        }
        public boolean error(String v) throws Exception {
            return p.error(v);
        }
        
    }
    public void setParser (IGenericXMLParser p_parser){
        parser = new Shell(p_parser);
    }

    public void setParser (IGenericXMLParser2 p_parser){
        parser = p_parser;
    }

    String getcomment(){
        String s = yytext();
        s = yytext().substring(4,s.length()-3);
        return s;
    }
    
    String getheader(){
        String s = yytext();
        s = yytext().substring(2, s.length()-2);
        return s;
    }
    void pushstate(int state) { 
       statestk[statestkptr++]=yystate();
       yybegin(state); 
    }

    int  popstate (){ 
       int newstate = statestk[--statestkptr];
       statestk[statestkptr]=0;
       yybegin(newstate);
       return newstate;  
    }
    
    void error(String v)throws Exception{
       if(!parser.error(v)){
         throw new Exception("Unmatched characters, parsing cannot continue at" + where());
       }  
    }

    static private Pattern xamp = Pattern.compile("&amp;"); 
    static private Pattern xlt = Pattern.compile("&lt;"); 
    static private Pattern xgt = Pattern.compile("&gt;"); 
    static private Pattern xsqu = Pattern.compile("&apos;");
    static private Pattern xdqu = Pattern.compile("&quot;"); 

     static public String unencode (String s){
         if(s.indexOf("&")>=0){
             s= xlt.matcher(s).replaceAll("<");
             s= xgt.matcher(s).replaceAll(">");
             s= xsqu.matcher(s).replaceAll("'");
             s= xdqu.matcher(s).replaceAll("\"");
             s= xamp.matcher(s).replaceAll("&");
         }
        return s;
      } 
     static private Pattern tst = Pattern.compile("[&<>'\"]+");
     static private Pattern amp = Pattern.compile("&"); 
     static private Pattern lt = Pattern.compile("<"); 
     static private Pattern gt = Pattern.compile(">"); 
     static private Pattern squ = Pattern.compile("'");
     static private Pattern dqu = Pattern.compile("\""); 
     
     static public String encode (String s) {
         if(tst.matcher(s).find()){
             s= amp.matcher(s).replaceAll("&amp;");
             s= lt.matcher(s).replaceAll("&lt;");
             s= gt.matcher(s).replaceAll("&gt;");
             s= squ.matcher(s).replaceAll("&apos;");
             s= dqu.matcher(s).replaceAll("&quot;");
         }
        return s;
      }


  static public StringBuffer encode (StringBuffer sb) {

	return new StringBuffer(encode(sb.toString()));	

  }    
    

  public String where() {
  	return sourcename + ":" + (yyline+1) + "." + (yycolumn+1) ;
  }

  public int getYYLine() { return yyline+1;}
  public int getYYColumn() { return yycolumn+1;}
  
   /**
     * Loads an XML file with the given Generic Parser.
     * <br><br>
     * @param file  An inputStream providing the XML
     * @param gp    A parser implementing the IGenericXMLParser interface.
     */
    static public void load(java.io.InputStream xmlStream, IGenericXMLParser2 gp) throws Exception{
        GenericXMLParser parser = new GenericXMLParser(xmlStream);
        parser.setParser(gp);
        while(true){
            if(GenericXMLParser.YYEOF == parser.yylex()) break;
        }
     } 
     
    /**
	 * Loads an XML file with the given Generic Parser.
	 * <br><br>
	 * @param file  An inputStream providing the XML
	 * @param gp    A parser implementing the IGenericXMLParser interface.
	 */
    static public void load(java.io.InputStream xmlStream, IGenericXMLParser gp) throws Exception{
	    GenericXMLParser parser = new GenericXMLParser(xmlStream);
        parser.setParser(gp);
		while(true){
			if(GenericXMLParser.YYEOF == parser.yylex()) break;
		}
	 } 
	
	/**
      * Loads an XML file with the given Generic Parser.
      * @param xmlStream
      * @param gp
      * @throws Exception
      */
     static public void load(java.io.Reader xmlStream, IGenericXMLParser2 gp) throws Exception{
        GenericXMLParser parser = new GenericXMLParser(xmlStream);
        parser.setParser(gp);
        while(true){
            if(GenericXMLParser.YYEOF == parser.yylex()) break;
        }
     } 
	 
	 /**
	  * Loads an XML file with the given Generic Parser.
	  * @param xmlStream
	  * @param gp
	  * @throws Exception
	  */
     static public void load(java.io.Reader xmlStream, IGenericXMLParser gp) throws Exception{
		GenericXMLParser parser = new GenericXMLParser(xmlStream);
		parser.setParser(gp);
		while(true){
			if(GenericXMLParser.YYEOF == parser.yylex()) break;
		}
	 } 

    /**
     * Loads an XML file with the given Generic Parser, extended from AGenericXMLParser.  This
     * parser allows the "calling" of other AGenericXMLParser's to process tags.  The process
     * all of the given tag (and sub tags).  Then processing returns to the original parser.
     * <br><br>
     * @param file  An inputStream providing the XML
     * @param gp    A parser implementing the IGenericXMLParser interface.
     */
    static public void load(java.io.InputStream xmlStream, AGenericXMLParser gp) throws Exception{
        GenericXMLParser      parser = new GenericXMLParser(xmlStream);
        GenericXMLParserStack stack  = new GenericXMLParserStack(parser);
        gp.genericXMLParserStack = stack;
        stack.parseTagWith(gp);
        parser.setParser(stack);
        while(true){
            if(GenericXMLParser.YYEOF == parser.yylex()) break;
        }
     } 
  
//End of User Functions  
    
%}
   
Char       = [a-z]|[A-Z]|"_"
Digit      = [0-9]
namestart  = {Char}
namechar   = {namestart} | "-" | "." | {Digit}
Identifier = {namestart}{namechar}*(":"{namestart}{namechar}*)?
EOL        = \r|\n|\r\n
ws         = {EOL}|[ \t\f]
string1    = "'"[^']*"'"
string2    = "\""[^\"]*"\""
string     = {string1}|{string2}
body       = ({ws}|[^<>])*
any        = Char|ws|Digit|">"|"<"|"&"|.
comment    = "<!--"([^-]|([-][^-])|[-][-][^>])*"-->"
header     = "<?"([^?]|[?][^>])*"?>"

%xstate Attributes
%xstate Tag
%xstate GetEndTag
%xstate GetNestedEndTag
%xstate EndTag
%xstate NestedTag
%%

<YYINITIAL> {

  "<"                   {pushstate(Tag); }
  {header}              {parser.header(getheader());}
  {ws}                  { }
  {comment}             {parser.comment(getcomment());}
  {any}                 { error(yytext()); }
  
}  

<Attributes> {
  {Identifier}{ws}*"="{ws}*{string} {
     String text  = yytext();
     String key   = text.substring (0,text.indexOf('=')).trim();
     String value = text.substring (text.indexOf('=')+1).trim();
	 char delimiter = value.charAt(0);
	 value = value.substring(1,value.lastIndexOf(delimiter));
	 
     attribs.put(key,unencode(value));
  }

  "/"?">" {
     yypushback(1);
     if(yytext().indexOf("/")>=0) yypushback(1);
     popstate();
  }
  
  {ws} { }
  {any}          { error(yytext()); }
}

<Tag> {
  {Identifier} {
     pushTag(yytext());
     body="";
     pushstate(Attributes);
  }

  ">" {
     parser.beginTag(tagstk,tagstkptr,currenttag,attribs);
     popstate();
     pushstate(EndTag);
   }

  "/>" {
     parser.beginTag(tagstk,tagstkptr,currenttag,attribs);
     parser.endTag(tagstk,tagstkptr,currenttag,"",attribs);

     popTag(currenttag);
     popstate();
     return 1;
   }
  
  {comment}    {parser.comment(getcomment());}
  {ws} { }
  {any}          { error(yytext()); }
  
}

<GetEndTag> {
  {Identifier} {
     String endTag = yytext();
     parser.endTag(tagstk,tagstkptr,currenttag,unencode(body),attribs);
     popTag(endTag);
  }
  
  {ws}* { }

  ">"  {
    popstate();
  }
}  

<GetNestedEndTag> {
  {Identifier} {
     String endTag = yytext();
     parser.endTag(tagstk,tagstkptr,currenttag,"",attribs);
     popTag(endTag);
  }
  
  {ws}* { }

  ">"  {
     popstate();
     return 1;
  }
}  



<EndTag> {
  
  {body} {
     String text = yytext();
     body += text;
  }

  "</" {
     popstate();
     pushstate(GetEndTag);
  }

  "<"       {
     popstate();
     pushstate(NestedTag);
     pushstate(Tag); 
  }

  {comment}    {parser.comment(getcomment());}

  {any}     { body += yytext(); error(yytext()); }
}

<NestedTag> {
  "<"       {pushstate(Tag); }

  "</" {
     popstate();
     pushstate(GetNestedEndTag);
  }

  {ws}             {}
  {comment}        {parser.comment(getcomment());}
  {any}            {error(yytext()); }

}

