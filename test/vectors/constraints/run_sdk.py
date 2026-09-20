#!/usr/bin/env python3
"""SDK write-path vectors for #1209 (Project.SetAttribute, DebugSession.SetAttribute). Written BEFORE the fix.
Builds ./cmd/dtrules, declares allowed_values on a copy of SinusitisTherapy, runs the Go vectors in sdk/ against it."""
import json,os,shutil,subprocess,sys,tempfile
here=os.path.dirname(os.path.abspath(__file__)); repo=os.path.abspath(here+'/../../..'); tmp=tempfile.mkdtemp(); B=tmp+'/dtrules'
if subprocess.run(['go','build','-o',B,'./cmd/dtrules'],cwd=repo).returncode: sys.exit('FAIL build')
proj=tmp+'/p'; shutil.copytree(repo+'/sampleprojects/SinusitisTherapy',proj)
r=subprocess.run([B,'edd','patch','--project','.'],cwd=proj,capture_output=True,text=True,input=json.dumps({'op':'update-field','entity':'patient','field':{'name':'diagnosis','allowed_values':['Acute Sinusitis','Chronic Sinusitis']}}))
if r.returncode: sys.exit('FAIL fixture: '+r.stdout[-300:])
r=subprocess.run(['go','test','-count=1','-v','./test/vectors/constraints/sdk/'],cwd=repo,env=dict(os.environ,DTR_VECTOR_PROJECT=proj),capture_output=True,text=True)
out=r.stdout+r.stderr
for l in out.splitlines():
    if l.startswith(('--- ','    sdk_test.go','FAIL','ok','panic')): print(l)
if '--- SKIP' in out: print('FAIL: vectors skipped'); sys.exit(1)
sys.exit(r.returncode)
