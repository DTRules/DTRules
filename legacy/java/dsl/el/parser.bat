@echo off
rem * Copyright 2004-2008 MTBJ, Inc.  
rem *   
rem * Licensed under the Apache License, Version 2.0 (the "License");  
rem * you may not use this file except in compliance with the License.  
rem * You may obtain a copy of the License at  
rem *   
rem *      http://www.apache.org/licenses/LICENSE-2.0  
rem *   
rem * Unless required by applicable law or agreed to in writing, software  
rem * distributed under the License is distributed on an "AS IS" BASIS,  
rem * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
rem * See the License for the specific language governing permissions and  
rem * limitations under the License.  
rem *
set projectDirectory=%cd%
set cupDir=%projectDirectory%\src\main\java\com\dtrules\compiler\el\cup\parser
set libDir=%projectDirectory%\lib

rem echo projectDirectory
rem dir %projectDirectory%
rem echo libDir
rem dir %libDir%

cd %cupDir%
rem echo cupDir
rem dir %cupDir%
rem echo create BNF

java -cp %libDir%\java-cup-11a.jar java_cup.Main -parser DTRulesParser -symbols Symbols -dump_grammar < parser.cup 2> bnf.txt
rem echo create Java file
java -cp %libDir%\java-cup-11a.jar java_cup.Main -compact_red -nopositions -parser DTRulesParser < parser.cup


copy /y copyright.txt+DTRulesParser.java xxx.tmp 	> nul
copy /y xxx.tmp DTRulesParser.java 					> nul

copy /y copyright.txt+sym.java xxx.tmp 				> nul
copy /y xxx.tmp sym.java 							> nul

copy /y copyright.txt+bnf.txt  xxx.tmp 				> nul
copy /y xxx.tmp bnf.txt								> nul

del xxx.tmp > nul 						


cd %projectDirectory%
pause