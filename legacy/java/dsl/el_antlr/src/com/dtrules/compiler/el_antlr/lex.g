lexer grammar lex;

@header {
package com.dtrules.compiler.el_antlr.output;
}

fragment CR                 :  '\r';
fragment NL                 :  '\n';
fragment TAB                :  '\t';
fragment SPC                :  ' ';
fragment WS                 :  (TAB|CR|NL|SPC);
fragment EOL                :  CR? NL;
fragment DIGIT              :  ('0'..'9');
fragment CHAR               :  ('a'..'z'|'A'..'Z');
fragment INT                :  DIGIT+;
fragment FLT                :  INT       '.' ~DIGIT
                              | ~DIGIT    '.' INT
                              | INT       '.' INT
                              ;
fragment IDT                :  (CHAR|'_')(CHAR|DIGIT|'_')*;
fragment IDENTIFIER         :  (IDT '.')? IDT;
fragment STRINGSGL          :  '\'' ( ESC_SEQ | ~('\\'|'\'') )* '\'';
fragment STRINGDBL          :  '"' ( ESC_SEQ | ~('\\'|'"') )* '"';
fragment ESC_SEQ            :  '\\' ('b'|'t'|'n'|'f'|'r'|'\"'|'\''|'\\') ;
fragment STR                :  STRINGSGL|STRINGDBL; 

fragment A                  :  ('a'|'A');
fragment B                  :  ('b'|'B');
fragment C                  :  ('c'|'C');
fragment D                  :  ('d'|'D');
fragment E                  :  ('e'|'E');
fragment F                  :  ('f'|'F');
fragment G                  :  ('g'|'G');
fragment H                  :  ('h'|'H');
fragment I                  :  ('i'|'I');
fragment J                  :  ('j'|'J');
fragment K                  :  ('k'|'K');
fragment L                  :  ('l'|'L');
fragment M                  :  ('m'|'M');
fragment N                  :  ('n'|'N');
fragment O                  :  ('o'|'O');
fragment P                  :  ('p'|'P');
fragment Q                  :  ('q'|'Q');
fragment R                  :  ('r'|'R');
fragment S                  :  ('s'|'S');
fragment T                  :  ('t'|'T');
fragment U                  :  ('u'|'U');
fragment V                  :  ('v'|'V');
fragment W                  :  ('w'|'W');
fragment X                  :  ('x'|'X');
fragment Y                  :  ('y'|'Y');
fragment Z                  :  ('z'|'Z');

ACTION                      :  A C T I O N;
CONDITION                   :  C O N D I T I O N;
TRUE                        :  T R U E;
FALSE                       :  F A L S E;

INTEGER                     :  INT;
FLOAT                       :  FLT;
DATE                        :  D A T E;
BOOLEAN                     :  B O O L E A N     ;
DOUBLE                      :  D O U B L E     ;
SEMI                        :  ';'    ;
COLON                       :  ':'    ;
COMMA                       :  ','    ;
QUESTIONMARK                :  '?'    ;
PLUS                        :  '+'    ;
MINUS                       :  '-'    ;
DIVIDE                      :  '/'
                               | D I V
                               ;     
TIMES                       :  '*'    ;
LPAREN                      :  '('    ;
RPAREN                      :  ')'    ;
LBRACE                      :  '['    ;
RBRACE                      :  ']'    ;
LCURLY                      :  '{'    ;
RCURLY                      :  '}'    ;
SET                         :  S E T     ;
END                         :  E N D     ;
ADD                         :  A D D     ;
SUBTRACT                    :  S U B T R A C T     ;
REMOVE                      :  R E M O V E     ;
FROM                        :  F R O M     ;
T_ARRAY                     :  A R R A Y     ;
INCLUDE                     :  I N C L U D E     ;
INCLUDES                    :  I N C L U D E S     ;
ATTRIBUTE                   :  A T T R I B U T E     ;
VALUE                       :  V A L U E     ;
T_STRING                    :  S T R I N G     ;
T_NAME                      :  N A M E     ;
LOCAL                       :  L O C A L     ;
SUBSTRING                   :  S U B S T R I N G     ;
INDEX_OF                    :  I N D E X (WS)+ O F     ;
MEMBER                      :  M E M B E R     ;
THIS                        :  T H I S     ;
CONTEXT                     :  C O N T E X T     ;
FORALL                      :  F O R (WS)*A L L     
                                     | F O R (WS)*E A C H     ;
EACH                        :  E A C H     ;
LONG                        :  L O N G     ;
ALL                         :  A L L     ;
PERFORM                     :  P E R F O R M     ;
IN                          :  I N     ;
EQ                          :  '=='    
                               | I S (WS)+ E Q U A L (WS)+T O     
                               | E Q U A L (WS)+T O     
                               ;
NEQ                         :  '!='    
                               | I S (WS)+N O T (WS)+E Q U A L (WS)+T O     
                               | N O T (WS)+E Q U A L (WS)+T O     
                               ;
ASSIGN                      :  '='    ;
TO                          :  T O     ;
IS                          :  I S 
                               | A R E
                               ;
ITS                         :  I T S     ;
GT                          :  '>'     
                               | I S (WS)+G R E A T E R (WS)+T H A N     
                               | G R E A T E R (WS)+T H A N     
                               ;
LT                          :  '<'      
                               | I S (WS)+L E S S (WS)+T H A N     
                               | L E S S (WS)+T H A N     
                               ;
GTE                         :  '>=' 
                               | I S (WS)+G R E A T E R (WS)+T H A N (WS)+O R (WS)+E Q U A L +(WS)+T O 
                               | G R E A T E R (WS)+T H A N (WS)+O R (WS)+E Q U A L +(WS)+T O     
                               ;
    
LTE                         :  '<='
                               | I S (WS)+L E S S (WS)+T H A N (WS)+O R (WS)+E Q U A L +(WS)+T O 
                               | L E S S (WS)+T H A N (WS)+O R (WS)+E Q U A L +(WS)+T O     
                               ;
AND                         :  A N D     
                               | '&&'    
                               ;
OR                          :  O R 
                               | '||'
                               ;
NOT                         :  N O T     ;
NO                          :  N O     ;
IF                          :  I F     ;
THEN                        :  T H E N     ;
ENDIF                       :  E N D I F     ;
ELSE                        :  E L S E     ;
FIRST                       :  F I R S T     ;
OF                          :  O F     ;
ON                          :  O N     ;
USING                       :  U S I N G     ;
COPY                        :  C O P Y     ;
DEEPCOPY                    :  D E E P (WS)+ C O P Y     ;
GET                         :  G E T     ;
SORT                        :  S O R T     ;
BY                          :  B Y     ;
NEW                         :  N E W     ;
EARLIEST                    :  E A R L I E S T     ;
ENTITY                      :  E N T I T Y     ;
DEBUG                       :  D E B U G     ;
CLEAR                       :  C L E A R     ;
CLONE                       :  C L O N E     ;
FOR                         :  F O R     ;
RANDOMIZE                   :  R A N D O M I Z E     ;
WAS                         :  W A S     ;
ONE                         :  O N E     ;
DOES                        :  D O E S     ;
DAYS                        :  D A Y (S )?    ;
ISNULL                      :  I S (WS)+N U L L     ;
ISNOTNULL                   :  I S (WS)+N O T (WS)+N U L L     ;
BETWEEN                     :  B E T W E E N     ;
BEFORE                      :  B E F O R E     ;
AFTER                       :  A F T E R     ;
LENGTH                      :  L E N G T H     ;
THERE                       :  T H E R E     ;
NUMBEROF                    :  N U M B E R (WS)+O F     ;
THENAME                     :  T H E (WS)+ N A M E     ;
RELATIONSHIP_BETWEEN        :  R E L A T I O N S H I P (WS)+B E T W E E N     ;
STARTS_WITH                 :  S T A R T S (WS)+W I T H     ;
ALLOWING                    :  A L L O W I N G     ;
TABLE                       :  T A B L E     ;
HAVE                        :  H A V E     ;
YEARS                       :  Y E A R (S )?    ;
MONTHS                      :  M O N T H (S )?    ;
TOKENIZE                    :  T O K E N I Z E     ;
TOBEREMOVED                 :  T O (WS)+B E (WS)+R E M O V E D     ;
TABLEINFORMATION            :  T A B L E (WS)+I N F O R M A T I O N     ;
WITHIN                      :  W I T H (WS)*I N     ;
PERCENTOF                   :  P E R C E N T (WS)+O F     ;
PLUSORMINUS                 :  P L U S (WS)+O R (WS)+M I N U S     ;
MATCH                       :  M A T C H     ;
MATCHES                     :  M A T C H E S     ;
ONERROR                     :  O N (WS)+E R R O R     ;
ABSOLUTEVALUE               :  A B S O L U T E (WS)+V A L U E     ;
ELSEIFNONEAREFOUND          :  E L S E (WS)+I F (WS)+N O N E (WS)+A R E (WS)+F O U N D     ;
HASA                        :  H A S (WS)+(A |A N )    ;
DESCENDINGORDER             :  D E S C E N D I N G (WS)+O R D E R ?    ;
ASCENDINGORDER              :  A S C E N D I N G (WS)+O R D E R ?    ;
ELSEIF                      :  E L S E (WS)+I F     ;
WHERE                       :  W H E R E 
                               | W H O S E 
                               | W H I C H 
                               | W H I L E     
                            ;
MAP                         :  M A P     ;
MAPPINGKEY                  :  M A P P I N G (WS)+K E Y     ;
THROUGH                     :  T H R O U G H     ;
POSSESSIVE                  :  IDENTIFIER '\'' S     ;
NAME                        :  '$' IDENTIFIER    ;
IDENT                       :  IDENTIFIER     ;
