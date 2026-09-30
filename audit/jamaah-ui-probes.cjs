const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
async function main(){
 const root=path.resolve(__dirname,'../../azhan-microsite/src');
 const policy=await import(require('node:url').pathToFileURL(path.join(root,'lib/portalPolicy.mjs')));
 const profile=fs.readFileSync(path.join(root,'app/portal/profil/page.jsx'),'utf8');
 const expression=profile.match(/const noPaspor = [^;]+;/)[0];
 assert.equal(vm.runInNewContext(expression+'\nnoPaspor',{jamaah:{no_paspor:'SYNTHETIC-PASSPORT'}}),'SYNTHETIC-PASSPORT');
 console.log('PASS: API no_paspor renders its value');
 const source=fs.readFileSync(path.join(root,'app/portal/pembayaran/page.jsx'),'utf8');
 const start=source.indexOf('  function pickFile(event)'),end=source.indexOf('  async function submit',start);
 assert.ok(start>=0&&end>start);
 let selected={name:'old.png'},error='';
 const pick=vm.runInNewContext(source.slice(start,end)+'\npickFile',{paymentFileError:policy.paymentFileError,setFile:v=>{selected=v},setFormError:v=>{error=v}});
 const event={target:{value:'new.pdf',files:[{name:'new.pdf',type:'application/pdf',size:6*1024*1024}]}};
 pick(event);assert.equal(selected,null);assert.equal(event.target.value,'');assert.ok(error);
 console.log('PASS: invalid replacement clears the previous payment proof and input');
}main().catch(error=>{console.error(error);process.exitCode=1});
