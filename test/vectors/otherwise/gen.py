#!/usr/bin/env python3
"""Generates the otherwise-column fixtures (#1215). Run once; output is committed. Postfix is hand-written
ONLY because the authoring API cannot yet express '*'; once it can, fixtures should be re-authored through it."""
import os
EDD='''<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="policy" number="100" access="rw">
		<field name="flag" type="integer" subtype="" access="rw" input="" default_value="1" comment="always 1 in these vectors"></field>
	</entity>
	<entity name="result" number="200" access="rw">
		<field name="fired" type="string" subtype="" access="rw" input="" default_value="" comment="each fired column appends its label"></field>
	</entity>
</entity_data_dictionary>
'''
MAP='''<?xml version="1.0" encoding="UTF-8"?>
<mapping><XMLtoEDD><map></map>
<entities><entity name='policy' number='1'></entity><entity name='result' number='1'></entity></entities>
<initialization><initialentity entity='policy' epush='true'></initialentity><initialentity entity='result' epush='true'></initialentity></initialization>
</XMLtoEDD></mapping>
'''
def cond(n,dsl,post,cols): return f'<condition_details><condition_number>{n}</condition_number><condition_dsl>{dsl}</condition_dsl><condition_postfix>{post}</condition_postfix><columns>{cols}</columns></condition_details>\n'
def act(n,label,cols): return f'<action_details><action_number>{n}</action_number><action_dsl>set result.fired = result.fired + "{label}"</action_dsl><action_postfix>result.fired "{label}" strconcat cvs /result.fired xdef</action_postfix><columns>{cols}</columns></action_details>\n'
def table(name,num,typ,c1,c1cols,c2cols,acts):
    return (f'<decision_table el_compiled="true">\n<table_name>{name}</table_name>\n<attribute_fields><Type>{typ}</Type><TABLE_NUMBER>{num}</TABLE_NUMBER></attribute_fields>\n<contexts></contexts><initial_actions></initial_actions>\n<conditions>\n'
     +cond(1,f'policy.flag == {c1}',f'policy.flag {c1} ==',c1cols)+cond(2,'policy.flag &lt; 100','policy.flag 100 &lt;',c2cols)+'</conditions>\n<actions>\n'
     +''.join(act(i+1,l,c) for i,(l,c) in enumerate(acts))+'</actions>\n</decision_table>\n')
def project(d,tables):
    os.makedirs(f'{d}/xml',exist_ok=True); open(f'{d}/xml/v_edd.xml','w').write(EDD); open(f'{d}/xml/v_map.xml','w').write(MAP)
    open(f'{d}/xml/v_dt.xml','w').write('<?xml version="1.0" encoding="UTF-8"?>\n<decision_tables>\n'+''.join(tables)+'</decision_tables>\n')
A3=[('1;','X--'),('2;','-X-'),('O;','--X')]
# cond1: flag == c1 (flag is 1).  cond2: flag < 100 (always true).  col3 is the otherwise column.
project('ok',[
 table('V1_first_none_match',9001,'FIRST',7,'Y-*','-N*',A3),      # col1 N, col2 N           -> O
 table('V2_first_col1_matches',9002,'FIRST',1,'Y-*','-N*',A3),    # col1 Y                   -> 1
 table('V3_all_none_match',9003,'ALL',7,'Y-*','-N*',A3),          #                          -> O
 table('V4_all_two_match',9004,'ALL',1,'Y-*','-Y*',A3),           # col1 Y, col2 Y           -> 1;2;
 table('V5_none_type_none_match',9005,'NONE',7,'Y-*','-N*',A3),   # "all table types"        -> O
 table('V6_x_in_every_column_other',9006,'FIRST',7,'Y-*','-N*',A3+[('A;','XXX')]),  # always-action via X everywhere -> O;A;
 table('V7_x_in_every_column_col1',9007,'FIRST',1,'Y-*','-N*',A3+[('A;','XXX')]),   #                                -> 1;A;
 table('V8_sole_star_column',9008,'FIRST',7,'*','*',[('O;','X')]), # regression guard (Poker-style) -> O
 table('V9_star_first_row_only',9009,'FIRST',7,'Y-*','-N-',A3),   # one '*' marks the column; its other cells are empty ('-' in XML) -> O
 table('V10_star_second_row_only',9010,'FIRST',7,'Y--','-N*',A3), #                                                                  -> O
])
project('err_star_not_last',[table('E1',9101,'FIRST',7,'Y*-','-*N',A3)])
project('err_two_star_columns',[table('E2',9102,'FIRST',7,'Y**','-**',A3)])
project('err_star_mixed_with_Y',[table('E3',9103,'FIRST',7,'Y-*','-NY',A3)])
project('err_star_cell_in_ordinary_column',[table('E4',9104,'FIRST',7,'Y*-','-N-',A3)])
