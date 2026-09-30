const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
async function main(){
 const handlers=new Map(),stored=[];let delivered;
 const response={ok:true,headers:new Headers(),clone(){return this;}};
 vm.runInNewContext(fs.readFileSync(path.resolve(__dirname,'../../azhan-microsite/public/sw.js'),'utf8'),{
  URL,self:{location:{hostname:'audit.example',origin:'https://audit.example'},addEventListener:(name,handler)=>handlers.set(name,handler)},
  caches:{open:async()=>({put:async(request)=>stored.push(request.url)}),match:async()=>undefined},
  fetch:async()=>response
 });
 for(const pathname of ['/invoice/synthetic','/portal/profil','/portal/aktivasi?token=synthetic','/api/portal/me']){
  delivered=undefined;handlers.get('fetch')({request:{method:'GET',url:'https://audit.example'+pathname,mode:'navigate'},respondWith:p=>{delivered=p}});
  assert.equal(delivered,undefined);assert.equal(stored.length,0);
 }
 console.log('PASS: invoice, portal, activation, and API bypass service-worker caching');
 response.headers=new Headers({'Cache-Control':'private, no-store'});
 handlers.get('fetch')({request:{method:'GET',url:'https://audit.example/public-private-response',mode:'navigate'},respondWith:p=>{delivered=p}});await delivered;await Promise.resolve();assert.equal(stored.length,0);
 console.log('PASS: private/no-store response is not cached');
 response.headers=new Headers();
 handlers.get('fetch')({request:{method:'GET',url:'https://audit.example/paket',mode:'navigate'},respondWith:p=>{delivered=p}});await delivered;await Promise.resolve();assert.equal(stored.length,1);
 console.log('PASS: eligible public navigation still caches');
}main().catch(error=>{console.error(error);process.exitCode=1});
