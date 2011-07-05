grammar test2;

@header {
package com.dtrules.compiler.el_antlr.output
}

action 	:	statement  {System.out.println($statement.value);};
	
statement 	returns [String value] 
	:	expr		{ $value = $expr.value; }
	|	ID '='  expr  		{ $value = "/"+$ID.text + " " + $expr.value + "def "; } 
	|	forall_stmt		{ $value = $forall_stmt.value; } 
	;
		
expr returns [String value] 
	:	 one=multExpr {$value = $one.value;}  (('+' | '-') two=multExpr {$value = $two.value;})*
	;
	 
multExpr returns [String value]
	:	a1=atom {$value = $a1.value; }('*' a2=atom {$value = $a2.value; })*
	;
	
forall_stmt returns [String value]
	:	'for' 'all' id1=ID 'in' id2=ID 	{ $value = "{ } "+$id2.text +" entitypush " +$id1.text+ " entitypop forall "  ; }
	|	'for' 'all' id1=ID 		{ $value = "{ } "+$id1.text +" repeat "; }
	;
	
atom returns [String value]	
	:	INT		{ $value = $INT.text; }
	|	ID		{ $value = $ID.text; }
	|	'(' expr ')'		{ $value = $expr.value; }
	;
	
fragment WSCHAR 
	:	(' ' | '\t' | '\n' | '\r') ;
SEMI	:	';';		
ARTICLES	:	(('a'|'an'|'the')  ) WSCHAR {skip();}; 	
ID	:	('a'..'z'|'A'..'Z')+ ;
INT	:	'0'..'9'+; 
WS                	:	WSCHAR+ {skip();} ;  
