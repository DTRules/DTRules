#!/usr/bin/env python3
"""Field-constraint ENFORCEMENT vectors for #1209 Part B. Written BEFORE the fix; not the implementer's to edit.
    A field may declare allowed_values / max_length / max_words. Every path that writes the field from OUTSIDE
    the rules must refuse a violating value: non-zero exit, an error naming entity.field, the offending value and
    the allowed set (or the limit and the actual size), and NO result printed. Matching is case-insensitive.
    A field that is absent takes its default with no error. Fields with no constraint are untouched.
usage: run.py        (builds ./cmd/dtrules from the repo this file lives in; fixture = SinusitisTherapy, patched)"""
import json,os,re,shutil,subprocess,sys,tempfile
here=os.path.dirname(os.path.abspath(__file__)); repo=os.path.abspath(here+'/../../..')
tmp=tempfile.mkdtemp(); B=tmp+'/dtrules'
b=subprocess.run(['go','build','-o',B,'./cmd/dtrules'],cwd=repo,capture_output=True,text=True)
if b.returncode: print('FAIL build:',b.stderr[-400:]); sys.exit(1)
fails=0
def report(ok,name,detail=''):
    global fails; print(('ok   ' if ok else 'FAIL ')+name+('' if ok else '  -- '+detail)); fails+=not ok
def sh(args,cwd,stdin=None): r=subprocess.run([B]+args,cwd=cwd,input=stdin,capture_output=True,text=True); return r.returncode,r.stdout+r.stderr
def project(field):
    d=tempfile.mkdtemp()+'/p'; shutil.copytree(repo+'/sampleprojects/SinusitisTherapy',d)
    rc,out=sh(['edd','patch','--project','.'],d,json.dumps({'op':'update-field','entity':'patient','field':dict(name='diagnosis',**field)}))
    if rc: print('FAIL fixture: edd patch refused',out[-300:]); sys.exit(1)
    return d
DATA='<?xml version="1.0" encoding="UTF-8"?>\n<dtrules-data>\n  <patient>\n    <penicillin_allergic>false</penicillin_allergic>\n    <age>40</age>\n%s    <lean_body_weight>80</lean_body_weight>\n    <pcr>0.9</pcr>\n  </patient>\n</dtrules-data>\n'
INPUT="<?xml version='1.0' encoding='UTF-8'?>\n<patient>\n\t<age>40</age>\n\t<lean_body_weight>80</lean_body_weight>\n\t<pcr>0.9</pcr>\n\t<penicillin_allergic>false</penicillin_allergic>\n%s</patient>\n"
def go(proj,flag,value):
    body=(DATA if flag=='--data' else INPUT)%('' if value is None else f'    <diagnosis>{value}</diagnosis>\n')
    f=proj+'/case.xml'; open(f,'w').write(body); return sh(['run','.','--entry','Determine_Therapy',flag,f],proj)
def accepted(rc,out): return rc==0 and 'recommended_drug' in out
def refused(rc,out,*must): return rc!=0 and 'recommended_drug' not in out and all(re.search(m,out,re.I) for m in must)
VOCAB=project({'allowed_values':['Acute Sinusitis','Chronic Sinusitis'],'max_length':'40'})
rc,o=go(VOCAB,'--data','Acute Sinusitis');  report(accepted(rc,o),'V1  --data   value in the set loads and runs',o[-200:])
rc,o=go(VOCAB,'--data','acute sinusitis');  report(accepted(rc,o),'V2  --data   match is case-insensitive',o[-200:])
rc,o=go(VOCAB,'--data','Banana');           report(refused(rc,o,r'patient\.diagnosis','Banana','Chronic Sinusitis'),'V3  --data   value outside the set: error names field, value, allowed set; no result',f'rc={rc} {o[-240:]!r}')
rc,o=go(VOCAB,'--input','Banana');          report(refused(rc,o,r'patient\.diagnosis','Banana','Chronic Sinusitis'),'V4  --input  same value through the mapping: same error',f'rc={rc} {o[-240:]!r}')
rc,o=go(VOCAB,'--input','Chronic Sinusitis');report(accepted(rc,o),'V4b --input  value in the set loads and runs',o[-200:])
rc,o=go(VOCAB,'--data',None);               report(accepted(rc,o),'V6  --data   field absent: default used, no error',o[-200:])
LEN=project({'max_length':'40'})
rc,o=go(LEN,'--data','x'*40);               report(rc==0,'V7a --data   40 characters passes max_length=40',o[-200:])
rc,o=go(LEN,'--data','x'*41);               report(refused(rc,o,r'patient\.diagnosis',r'\b40\b',r'\b41\b'),'V7  --data   41 characters: error names the limit and the actual length',f'rc={rc} {o[-240:]!r}')
WORDS=project({'max_words':'5'})
rc,o=go(WORDS,'--data','one two  three\tfour five');      report(rc==0,'V8a --data   5 words (mixed whitespace) passes max_words=5',o[-200:])
rc,o=go(WORDS,'--data','one two  three\tfour five six');  report(refused(rc,o,r'patient\.diagnosis',r'\b5\b',r'\b6\b'),'V8  --data   6 words: error names the limit and the actual count',f'rc={rc} {o[-240:]!r}')
PLAIN=project({}); rc,o=go(PLAIN,'--data','Banana');       report(accepted(rc,o),'V13 no constraint declared: any value still loads (nothing enforced, nothing slowed)',o[-200:])
# V12: a rule that assigns a literal outside the set gets an ADVISORY from review (not an error: rules may write what they like)
tbl={'name':'Probe_Typo','number':1091,'policy':'FIRST','conditions':[{'number':1,'dsl':'patient.age > 0','columns':{'1':'Y'}}],
     'actions':[{'number':1,'dsl':'set patient.diagnosis = "Bannana"','columns':{'1':True}}]}
rc,o=sh(['table','put','Probe_Typo','--file','therapy_dt.xml','--project','.'],VOCAB,json.dumps(tbl))
if rc: report(False,'V12 fixture: table put Probe_Typo',o[-200:])
else:
    rc,o=sh(['review','.'],VOCAB); report('Probe_Typo' in o and 'Bannana' in o,'V12 review   advisory names the table and the literal assigned outside the set',o[-200:])
print(f'{fails} failing'); sys.exit(1 if fails else 0)
