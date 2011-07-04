@echo off
# * Copyright 2004-2008 MTBJ, Inc.  
# *   
# * Licensed under the Apache License, Version 2.0 (the "License");  
# * you may not use this file except in compliance with the License.  
# * You may obtain a copy of the License at  
# *   
# *      http://www.apache.org/licenses/LICENSE-2.0  
# *   
# * Unless required by applicable law or agreed to in writing, software  
# * distributed under the License is distributed on an "AS IS" BASIS,  
# * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
# * See the License for the specific language governing permissions and  
# * limitations under the License.  
# *
project=~/java/DTRules/EL
cupDir="$project/src/main/java/com/dtrules/compiler/el/cup/parser"
libDir="$project/lib"


cd $cupDir
echo $cupDir

java -cp $libDir/java-cup-11a.jar java_cup.Main -parser DTRulesParser -symbols sym -dump_grammar < parser.cup 2> bnf.txt
# echo create Java file
java -cp $libDir/java-cup-11a.jar java_cup.Main -compact_red -nopositions -parser DTRulesParser < parser.cup


cat copyright.txt DTRulesParser.java > xxx.tmp 	
cp  xxx.tmp DTRulesParser.java 					

cat copyright.txt sym.java > xxx.tmp1
cp  xxx.tmp1 sym.java

cat copyright.txt bnf.txt  > xxx.tmp2 				
cp xxx.tmp2 bnf.txt	

rm xxx.tmp  						
rm xxx.tmp1
rm xxx.tmp2

cd $project 
