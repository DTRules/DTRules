rem - 
rem - Copyright 2004-2011 DTRules.com, Inc.
rem -
rem - See http://DTRules.com for updates and documentation for the DTRules Rules Engine  
rem -  
rem - Licensed under the Apache License, Version 2.0 (the "License");  
rem - you may not use this file except in compliance with the License.  
rem - You may obtain a copy of the License at  
rem -  
rem -     http://www.apache.org/licenses/LICENSE-2.0  
rem -  
rem - Unless required by applicable law or agreed to in writing, software  
rem - distributed under the License is distributed on an "AS IS" BASIS,  
rem - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
rem - See the License for the specific language governing permissions and  
rem - limitations under the License.  
rem -

set pdir=C:\maximus\RulesEngine\DTRules\src\main\java\com\dtrules
set pdir2=C:\maximus\RulesEngine\DSLCompiler2\lib\
set xmldir=%pdir%\xmlparser
java -classpath %pdir2%\jflex.jar JFlex.Main -d %xmldir% %xmldir%\GenericXMLParser.flex
pause