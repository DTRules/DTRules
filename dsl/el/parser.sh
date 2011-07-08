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
project=.
cupDir="$project/src/main/java/com/dtrules/compiler/el/cup/parser"
libDir="$project/lib"

echo $cupDir

java -cp $libDir/java-cup-10k.jar java_cup.Main -parser DTRulesParser -symbols sym -dump_grammar < $cupDir/parser.cup 2> $cupDir/bnf.txt

echo create Java file
java -cp $libDir/java-cup-10k.jar java_cup.Main -compact_red -nopositions -parser DTRulesParser < $cupDir/parser.cup
mv DTRulesParser.java $cupDir
mv sym.java $cupDir

cat $cupDir/copyright.txt $cupDir/DTRulesParser.java > $cupDir/xxx.tmp 	
cp  $cupDir/xxx.tmp $cupDir/DTRulesParser.java 					

cat $cupDir/copyright.txt $cupDir/sym.java > $cupDir/xxx.tmp1
cp  $cupDir/xxx.tmp1 $cupDir/sym.java

cat $cupDir/copyright.txt $cupDir/bnf.txt  > $cupDir/xxx.tmp2 				
cp $cupDir/xxx.tmp2 $cupDir/bnf.txt	

rm $cupDir/xxx.tmp  						
rm $cupDir/xxx.tmp1
rm $cupDir/xxx.tmp2

cd $project 
