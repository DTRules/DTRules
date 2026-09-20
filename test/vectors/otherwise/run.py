#!/usr/bin/env python3
"""Otherwise-column vectors for #1215. Written BEFORE the fix; not the implementer's to edit.
    '*' does not mean "don't care". It is allowed only in the LAST column, and only when that column has no
    Y/N entries. It means OTHERWISE: it executes only if no other column executes, in ALL table types.
    There is no "always" column; to always execute an action, put an X in every column.
usage: run.py [engine|authoring|docs|all]   (builds ./cmd/dtrules from the repo this file lives in)"""
import json,os,re,shutil,subprocess,sys,tempfile
here=os.path.dirname(os.path.abspath(__file__)); repo=os.path.abspath(here+'/../../..'); which=(sys.argv[1:] or ['all'])[0]
tmp=tempfile.mkdtemp(); B=tmp+'/dtrules'
b=subprocess.run(['go','build','-o',B,'./cmd/dtrules'],cwd=repo,capture_output=True,text=True)
if b.returncode: print('FAIL build:',b.stderr[-400:]); sys.exit(1)
fails=0
def report(ok,name,detail=''):
    global fails; print(('ok   ' if ok else 'FAIL ')+name+('' if ok else '  -- '+detail)); fails+=not ok
def run(args,cwd): r=subprocess.run([B]+args,cwd=cwd,capture_output=True,text=True); return r.returncode,(r.stdout+r.stderr)
def fired(proj,entry):
    rc,out=run(['run','.','--entry',entry,'--result-entity','result'],proj); m=re.search(r'^\s*fired:\s*(\S*)\s*$',out,re.M)
    return rc,(m.group(1) if m else ''),out
ENGINE={'V1_first_none_match':'O;','V2_first_col1_matches':'1;','V3_all_none_match':'O;','V4_all_two_match':'1;2;','V5_none_type_none_match':'O;',
 'V6_x_in_every_column_other':'O;A;','V7_x_in_every_column_col1':'1;A;','V8_sole_star_column':'O;','V9_star_first_row_only':'O;','V10_star_second_row_only':'O;'}
if which in('engine','all'):
    for t,want in ENGINE.items():
        rc,got,out=fired(here+'/ok',t); report(rc==0 and got==want,'engine '+t,f'want fired={want!r} got {got!r} rc={rc} {out.strip()[-160:] if rc else ""}')
    for p,entry in (('err_star_not_last','E1'),('err_two_star_columns','E2'),('err_star_mixed_with_Y','E3'),('err_star_cell_in_ordinary_column','E4')):
        rc,got,out=fired(f'{here}/{p}',entry); named=re.search(entry,out) and re.search(r'otherwise|\*',out) and re.search(r'last|only',out,re.I)
        report(rc!=0 and bool(named),'engine '+p,f'want a load error naming table {entry} and the rule (mentions * / otherwise, and last/only); rc={rc} out={out.strip()[-200:]!r}')
    g=subprocess.run('grep -rn "alwaysColumn\\|starColumn" pkg/ --include=*.go | grep -v _test',shell=True,cwd=repo,capture_output=True,text=True).stdout
    report(g.strip()=='' ,'engine no dead always/star fields in pkg/',g.strip()[:200])
def T(cols1,cols2,acts):  # authoring JSON for a 2-condition table
    return {'name':'A_put','file':'a_dt.xml','number':9500,'policy':'FIRST','contexts':[],
      'conditions':[{'number':1,'dsl':'policy.flag == 7','columns':cols1},{'number':2,'dsl':'policy.flag < 100','columns':cols2}],
      'actions':[{'number':i+1,'dsl':f'set result.fired = result.fired + "{l}"','columns':c} for i,(l,c) in enumerate(acts)]}
if which in('authoring','all'):
    def fresh(): d=tempfile.mkdtemp(); shutil.copytree(here+'/ok',d+'/p'); return d+'/p'
    def put(proj,tbl): r=subprocess.run([B,'table','put','A_put','--file','a_dt.xml','--range','9500-9599','--reason','vector','--project','.'],input=json.dumps(tbl),cwd=proj,capture_output=True,text=True); return r.returncode,r.stdout+r.stderr
    acts=[('1;',{'1':True,'2':False,'3':False}),('2;',{'1':False,'2':True,'3':False}),('O;',{'1':False,'2':False,'3':True})]
    p=fresh(); rc,out=put(p,T({'1':'Y','2':'-','3':'*'},{'1':'-','2':'N','3':'-'},acts)); report(rc==0,'authoring put accepts an otherwise last column',out.strip()[-200:])
    if rc==0:
        rc2,got,o2=fired(p,'A_put'); report(got=='O;','authoring put -> run fires otherwise',f'got {got!r} {o2.strip()[-120:]}')
        rc3,o3=run(['table','get','A_put','--project','.'],p); report(rc3==0 and '"*"' in o3,'authoring get returns "*"',o3.strip()[-160:])
        rc4,o4=run(['verify','.'],p); report(rc4==0,'authoring verify passes (Excel paired, * round-trips)',o4.strip()[-200:])
        rc5,o5=run(['build','.'],p); rc6,got6,_=fired(p,'A_put'); report(rc5==0 and got6=='O;','authoring Excel -> build keeps the otherwise column',f'build rc={rc5} fired={got6!r} {o5.strip()[-160:]}')
    for nm,c1,c2 in (('* not in last column',{'1':'Y','2':'*','3':'-'},{'1':'-','2':'-','3':'N'}),('* column also has Y',{'1':'Y','2':'-','3':'*'},{'1':'-','2':'N','3':'Y'})):
        p=fresh(); rc,out=put(p,T(c1,c2,acts)); report(rc!=0 and re.search(r'last|only',out,re.I) is not None,'authoring put rejects: '+nm,f'rc={rc} {out.strip()[-200:]}')
    rc,o=run(['table','schema'],repo); report('"*"' in o,'authoring schema lists "*"','')
if which in('docs','all'):
    files=[os.path.join(dp,f) for d in ('cmd/dtrules','docs') for dp,_,fs in os.walk(f'{repo}/{d}') for f in fs if (f.startswith('doc') and f.endswith('.go') and not f.endswith('_test.go')) or f.endswith('.md')]
    text={f:open(f,errors='ignore').read() for f in files}
    bad=[(os.path.relpath(f,repo),m.group(0)[:70]) for f,t in text.items() for m in re.finditer(r'(?im)^.*(\*[^\n]{0,40}don.?t care|don.?t care[^\n]{0,40}\*(?![*\w])|use \* for).*$',t) if 'does not mean' not in m.group(0).lower() and 'not mean' not in m.group(0).lower()]
    report(not bad,'docs never describe * as "don\'t care"',str(bad[:3]))
    rc,topic=run(['docs','decision-tables'],repo); low=topic.lower()
    for need in ('otherwise','last column',"does not mean",'all table types','x in every column'):
        report(need in low,f'embedded `dtrules docs decision-tables` says: {need!r}','')
    report(re.search(r'always column',low) is None or 'no always column' in low or 'no "always" column' in low,'embedded docs do not teach an always column','')
    for md in ('docs/decision-table-xml-format.md','docs/spreadsheet-formats.md','docs/SPEC.md'):
        t=open(f'{repo}/{md}',errors='ignore').read().lower(); report('otherwise' in t and 'last column' in t,f'{md} defines the otherwise column','')
print(f'{fails} failing'); sys.exit(1 if fails else 0)
