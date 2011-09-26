/** 
 * Copyright 2004-2009 DTRules.com, Inc.
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
package com.dtrules.compiler.el.flex.scanner;
import java.util.*;
import java.io.*;
import java_cup.runtime.*;
import com.dtrules.compiler.el.cup.parser.sym;
import com.dtrules.infrastructure.RulesException;
@SuppressWarnings({"unchecked","unused"})
%%
%public
%class DTRulesscanner
%yylexthrow RulesException
%line
%char
%cup
%ignorecase

%{
  
    Symbol build(int code) {
      return new Symbol(code,yytext());
    }
    
	/** Accessor for yyline **/
	public int linenumber () { return yyline; }
	
	/** Accessor for yychar **/
    public int charnumber () { return yychar; }
    
%}
  
EOL        = \r|\n|\r\n
Whitespace = {EOL}|[ \t\f]
ws         = {Whitespace}
Char       = [a-z]|[A-Z]
digit      = [0-9]
integer    = {digit}+
float      = ({digit}+"."{digit}*)|({digit}*"."{digit}+)
Ident      = ({Char}|"_")({Char}|{digit}|"_")*
Identifier = ({Ident}".")?{Ident}
stringdbl  = ("\""[^\"]*"\"")
stringsgl  = ("'"[^']*"'")
string     = {stringdbl}|{stringsgl}

%%

<YYINITIAL> {


  "action"            {return build(sym.ACTION); }
  "condition"         {return build(sym.CONDITION); }
  "policystatement"   {return build(sym.POLICYSTATEMENT); }

  "true"              {return build(sym.RBOOLEAN); }
  "false"             {return build(sym.RBOOLEAN); }
  "default"           {return build(sym.RBOOLEAN); }
  "otherwise"         {return build(sym.RBOOLEAN); }
  "always"            {return build(sym.RBOOLEAN); } 

  "perform"{ws}+"when"{ws}+"called"
				      {return build(sym.RBOOLEAN); }
   
  {integer}           {return build(sym.INTEGER); }
  {float}             {return build(sym.FLOAT); }
  {string}            {return build(sym.STRING); }
  {Whitespace}        {}

  "policy"{ws}+"statements"
                      {return build(sym.POLICYSTATEMENTS);}

  "date"           {return build(sym.DATE);   }
  "time"           {return build(sym.DATE);   }
  "boolean"        {return build(sym.BOOLEAN);}
  "double"         {return build(sym.DOUBLE); }
  ";"              {return build(sym.SEMI);   }
  ":"              {return build(sym.COLON);  }
  ","              {return build(sym.COMMA);  }
  "?"              {return build(sym.QUESTIONMARK); }
  "+"              {return build(sym.PLUS);   }
  "-"              {return build(sym.MINUS);  }
  "/"              {return build(sym.DIVIDE); }
  "div"            {return build(sym.DIVIDE); }
  "*"              {return build(sym.TIMES);  }
  "("              {return build(sym.LPAREN); }
  ")"              {return build(sym.RPAREN); }
  "["              {return build(sym.LBRACE); }
  "]"              {return build(sym.RBRACE); }
  "{"              {return build(sym.LCURLY); }
  "}"              {return build(sym.RCURLY); }
  "set"            {return build(sym.SET);    }
  "end"            {return build(sym.END);    }
  "add"            {return build(sym.ADD);    }
  "subtract"       {return build(sym.SUBTRACT); }
  "remove"         {return build(sym.REMOVE); }
  "from"           {return build(sym.FROM);   }
  "array"          {return build(sym.ARRAY);  }
  "include"        {return build(sym.INCLUDE); }
  "includes"       {return build(sym.INCLUDES); }
  "attribute"      {return build(sym.ATTRIBUTE); }
  "value"          {return build(sym.VALUE);  }
  "string"         {return build(sym.STRING); }
  "name"           {return build(sym.NAME);   }
  "local"          {return build(sym.LOCAL);  }
  "substring"      {return build(sym.SUBSTRING); }
  "index"{ws}+"of" {return build(sym.INDEX_OF); }
  "member"("s")?   {return build(sym.MEMBER); }
  "this"           {return build(sym.THIS);   }
  "context"        {return build(sym.CONTEXT); }
  "for"{ws}*"all"  {return build(sym.FORALL); }
  "for"{ws}*"each" {return build(sym.FOREACH); }
  "each"           {return build(sym.EACH);   }
  "int"            {return build(sym.LONG);   }
  "long"           {return build(sym.LONG);   }
  "all"            {return build(sym.ALL);    }
  "perform"        {return build(sym.PERFORM); }
  "in"             {return build(sym.IN); }
  "=="             {return build(sym.EQ); }
  "!="             {return build(sym.NEQ); }
  "="              {return build(sym.ASSIGN); }
  "to"             {return build(sym.TO); }
  "is"             {return build(sym.IS); }
  "are"            {return build(sym.IS); }
  "its"            {return build(sym.ITS); }
  ">"|"&gt"        {return build(sym.GT); }
  "<"|"&lt"        {return build(sym.LT); }
  ">="|"&gt="      {return build(sym.GTE); }
  "<="|"&lt="      {return build(sym.LTE); }
  
  "is"{ws}+"equal"{ws}+("to"{ws})?+"ignore"{ws}+"case"                      {return build(sym.EQ_IGNORE_CASE);}  
  "is"{ws}+"not"{ws}+"equal"{ws}+("to"{ws})?+{ws}+"ignore"{ws}+"case"       {return build(sym.NEQ_IGNORE_CASE);}  
  "is"{ws}+"equal"{ws}+("to"{ws})?                                          {return build(sym.EQ);}
  "equal"{ws}+("to"{ws})?                                                   {return build(sym.EQ);}
  "is"{ws}+"not"{ws}+"equal"{ws}+("to"{ws})?                                {return build(sym.NEQ);}
  "not"{ws}+"equal"{ws}+("to"{ws})?                                         {return build(sym.NEQ);}
  "is"{ws}+"greater"{ws}+"than"                                  {return build(sym.GT);}
  "greater"{ws}+"than"                                           {return build(sym.GT);}
  "is"{ws}+"greater"{ws}+"than"{ws}+"or"{ws}+"equal"+{ws}+"to"   {return build(sym.GTE);}
  "greater"{ws}+"than"{ws}+"or"{ws}+"equal"+{ws}+"to"            {return build(sym.GTE);}
  "is"{ws}+"less"{ws}+"than"                                     {return build(sym.LT);}
  "less"{ws}+"than"                                              {return build(sym.LT);}
  "is"{ws}+"less"{ws}+"than"{ws}+"or"{ws}+"equal"+{ws}+"to"      {return build(sym.LTE);}
  "less"{ws}+"than"{ws}+"or"{ws}+"equal"+{ws}+"to"               {return build(sym.LTE);}
  
  "and"            {return build(sym.AND); }
  "&&"             {return build(sym.AND); }
  "or"             {return build(sym.OR); }
  "||"             {return build(sym.OR); }
  "not"            {return build(sym.NOT); }
  "no"             {return build(sym.NO); }
  "if"             {return build(sym.IF); }
  "then"           {return build(sym.THEN); } 
  "endff"          {return build(sym.ENDFF); } 
  "endif"          {return build(sym.ENDIF); } 
  "else"           {return build(sym.ELSE); } 
  "first"          {return build(sym.FIRST); } 
  "of"             {return build(sym.OF); } 
  "on"             {return build(sym.ON); } 
  "using"          {return build(sym.USING); }
  "copy"           {return build(sym.COPY); }
  "deep"{ws}+"copy" {return build(sym.DEEPCOPY); }
  "get"            {return build(sym.GET); }
  "sort"           {return build(sym.SORT); }
  "by"             {return build(sym.BY); }
  "new"            {return build(sym.NEW); }
  "earliest"       {return build(sym.EARLIEST); }
  "entity"         {return build(sym.ENTITY); }
  "debug"          {return build(sym.DEBUG); }
  "clear"          {return build(sym.CLEAR); }
  "clone"          {return build(sym.CLONE); }
  "for"            {return build(sym.FOR); }
  "randomize"      {return build(sym.RANDOMIZE); }
  "was"            {return build(sym.WAS); }
  "one"            {return build(sym.ONE); }
  "does"           {return build(sym.DOES); }
  "day"("s")?      {return build(sym.DAYS); }
  "is"{ws}+"null"  {return build(sym.ISNULL); }
  "is"{ws}+"not"{ws}+"null"
                   {return build(sym.ISNOTNULL); }
  "change"         {return build(sym.CHANGE); }
  "upper"{ws}+"case"
                   {return build(sym.UPPER_CASE); }
  "lower"{ws}+"case"
                   {return build(sym.LOWER_CASE); }
  "between"        {return build(sym.BETWEEN); }
  "before"         {return build(sym.BEFORE); }
  "after"          {return build(sym.AFTER); }
  "length"         {return build(sym.LENGTH); }
  "there"          {return build(sym.THERE); }
  "number"{ws}+"of" {return build(sym.NUMBEROF); }
  "the"{ws}+"name" {return build(sym.THENAME); }
  "relationship"{ws}+"between"   
                   {return build(sym.RELATIONSHIP_BETWEEN); }
  "starts"{ws}+"with"
                   {return build(sym.STARTS_WITH); }
  "allowing"       {return build(sym.ALLOWING); } 
  "table"          {return build(sym.TABLE); }
  "have"           {return build(sym.HAVE); }
  "year"("s")?     {return build(sym.YEARS); }
  "month"("s")?    {return build(sym.MONTHS); }
  "tokenize"       {return build(sym.TOKENIZE); }
  "to"{ws}+"be"{ws}+"removed"  
                   {return build(sym.TOBEREMOVED); }
  "table"{ws}+"information"
		           {return build(sym.TABLEINFORMATION); }
  "with"{ws}*"in"  {return build(sym.WITHIN); }
  "percent"{ws}+"of" 
                   {return build(sym.PERCENTOF); }
  "plus"{ws}+"or"{ws}+"minus"
                   {return build(sym.PLUSORMINUS); }
  "match"          {return build(sym.MATCH); }
  "matches"        {return build(sym.MATCHES); }

  "on"{ws}+"error"                  {return build(sym.ONERROR); }
  "absolute"{ws}+"value"            {return build(sym.ABSOLUTEVALUE); }
  "else"{ws}+"if"{ws}+"none"{ws}+"are"{ws}+"found"
                                    {return build(sym.ELSEIFNONEAREFOUND); }
  "has"{ws}+("a"|"an")              {return build(sym.HASA); }
  "descending"{ws}+"order"?         {return build(sym.DESCENDINGORDER); }
  "ascending"{ws}+"order"?          {return build(sym.ASCENDINGORDER); }
  "else"{ws}+"if"                   {return build(sym.ELSEIF);  }
  "where"|"whose"|"which"|"while"   {return build(sym.WHERE); }     
  "map"                             {return build(sym.MAP); }
  "mapping"{ws}+"key"               {return build(sym.MAPPINGKEY); }
  "through"                         {return build(sym.THROUGH); }                                        
                   
  "//"[^\r\n]*     { }             //   //        comments
  "/*"([^/]|("/"[^*]))*"*/" { }    //   /* ... */ comments

  "a"|"an"|"the" { } /* Just ignore articles so you can put them anywhere you like */  
  {Identifier}"'s"                  { return build(sym.POSSESSIVE); }
  "$"{Identifier}                   { return build(sym.NAME); }
  {Identifier} 				{ return build(sym.IDENT); }

} 

  


     