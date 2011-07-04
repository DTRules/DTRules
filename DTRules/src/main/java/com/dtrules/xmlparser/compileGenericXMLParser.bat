rem Copyright 2004-2009 DTRules.com, Inc.
rem _ 
rem Licensed under the Apache License, Version 2.0 (the "License");  
rem you may not use this file except in compliance with the License.  
rem You may obtain a copy of the License at  
rem _  
rem     http://www.apache.org/licenses/LICENSE-2.0  
rem _   
rem Unless required by applicable law or agreed to in writing, software  
rem distributed under the License is distributed on an "AS IS" BASIS,  
rem WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
rem See the License for the specific language governing permissions and  
rem limitations under the License.  
rem _ 

set pdir=C:\maximus\eb_dev2\RulesEngine\DTRules\src\main\java\com\dtrules
set pdir2=C:\maximus\eb_dev2\RulesEngine\DSLCompiler2\lib\
set xmldir=%pdir%\xmlparser
java -classpath %pdir2%\jflex.jar JFlex.Main -d %xmldir% %xmldir%\GenericXMLParser.flex
pause