set path=%path%;jflex-1.4.3\bin;
set JFLEX_HOME=.\
set JAVA_HOME=c:\Program Files\Java\jdk1.6.0_18
set CLPATH=%JFLEX_HOME%\libJFlex.jar
echo .
set CLPATH
set JFLEX_HOME
echo .
java -Xmx128m -jar %jflex_home%\lib\JFlex.jar -v -d "src\main\java\com\dtrules\compiler\el\flex\scanner" "src\main\java\com\dtrules\compiler\el\flex\scanner\scanner.flex\
pause