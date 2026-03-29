grammar test;


action 	:	statement  {System.out.println($expr.value);};
	
statement 	returns [String value] 
	:	expr		{ $value = $expr.value; }
	|	ID '='  expr  		{ $value = "/"+$ID.text + " " + $expr.value + " def "; } 
	|	forall_stmt		{ $value = $forall_stmt.value; } 
	;
		
expr returns [String value] 
	:	 multExpr {$value = "x ";}  (('+' | '-') multExpr )*
	;
	 
multExpr	:	atom ('*' atom)*
	;
	
forall_stmt returns [String value]
	:	'for' 'all' ID 'in' ID 	{ $value = "for all in "; }
	|	'for' 'all' ID 		{ $value = "for all "; }
	;
	
atom returns [String value]	
	:	INT		{ $value = $INT.text; }
	|	ID		{ $value = $INT.text; }
	|	'(' expr ')'		{ $value = $expr.value; }
	;
	
SEMI	:	';';		
ARTICLES	:	(('a'|'an'|'the') (' ' | '\t' | '\n' | '\r') )  {skip();}; 	
ID	:	('a'..'z'|'A'..'Z')+ ;
INT	:	'0'..'9'+;
WS                	:	(' ' | '\t' | '\n' | '\r')+ {skip();} ;  
