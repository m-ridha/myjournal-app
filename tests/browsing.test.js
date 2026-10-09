// Dependency-free list navigation regressions: Node.js or macOS JavaScriptCore.
const output = typeof print === 'function' ? print : console.log;
const assert = (ok, message) => { if (!ok) throw new Error(message); };
const tick = async () => { for (let i=0;i<16;i++) await Promise.resolve(); };
// JavaScriptCore lacks browser URLSearchParams; the harness supplies that primitive.
if (typeof URLSearchParams === 'undefined') globalThis.URLSearchParams = class {
  constructor(search) { this.values = new Map(search.replace(/^\?/, '').split('&').map(pair => pair.split('=').map(decodeURIComponent))); }
  get(name) { return this.values.get(name) ?? null; }
};
function element() {
 const listeners={};const node={children:[],hidden:false,disabled:false,textContent:'',addEventListener:(name,fn)=>{listeners[name]=fn;},fire:(name)=>listeners[name]?.(),setAttribute(){},appendChild(child){this.children.push(child);},replaceChildren(...children){this.children=children;}};
 Object.defineProperty(node,'innerHTML',{set(){this.children=[];}});return node;
}
const payload=(page=1,hasNext=false,items=[])=>({ok:true,json:async()=>({page,pageSize:20,hasNext,items})});
function setup(search='',initial=payload()) {
 const list=element(),newer=element(),older=element(),status=element(),retry=element();
 const fields={'[data-entry-list]':list,'[data-entries-newer]':newer,'[data-entries-older]':older,'[data-entries-status]':status,'[data-entries-retry]':retry};
 const events={};globalThis.document={documentElement:{lang:'en'},body:{dataset:{messages:'{}'}},querySelector:s=>fields[s]||null,addEventListener:(name,fn)=>{events[name]=fn;},createElement:()=>element()};
 globalThis.window={location:{search,href:''},addEventListener(){}};
 const calls=[],queue=[initial];globalThis.fetch=async(url)=>{calls.push(url);if(!queue.length)throw new Error('unplanned request');return queue.shift();};
 if(typeof require==='function')(0,eval)(require('node:fs').readFileSync('assets/static/js/app.js','utf8'));else load('assets/static/js/app.js');
 events.DOMContentLoaded();return {list,newer,older,status,retry,calls,queue};
}
async function run() {
 let f=setup('',payload(1,true,[{date:'2026-01-02',preview:'<script>plain text</script>'},{date:'2025-12-31',preview:'previous year'}]));await tick();
 assert(f.newer.hidden && !f.older.hidden && f.older.href==='/entries?page=2','first page exposes only Older');
 assert(f.list.children.length===2 && f.list.children[0].children[0].textContent==='2026','group entries newest-year first');
 assert(f.list.children[0].children[1].children[1].children[0].textContent==='<script>plain text</script>','previews remain plain text');
 f=setup('?page=2',payload(2,true));await tick();assert(f.calls[0]==='/api/v1/entries?page=2&size=20' && f.newer.href==='/entries?page=1' && f.older.href==='/entries?page=3','page query selects matching API page and links');
 f=setup('?page=3',payload(3,false));await tick();assert(!f.newer.hidden && f.older.hidden,'last/empty later page keeps way back');
 for(const search of ['?page=0','?page=-1','?page=10001','?page=1.5','?page=abc','?page=99999999999999999999']) {f=setup(search);await tick();assert(f.calls[0]==='/api/v1/entries?page=1&size=20','invalid page falls back to safe first page');}
 f=setup('?page=10000',payload(10000,true));await tick();assert(f.older.hidden,'maximum page has no out-of-range Older link');
 f=setup('?page=2',{ok:false,status:503});await tick();assert(f.list.children[0].textContent==='entries.unavailable' && !f.retry.hidden && !f.newer.hidden && f.older.hidden,'failed later page offers retry and Newer without advancing');
 f.queue.push(payload(2,false,[{date:'2026-01-01',preview:'recovered'}]));f.retry.fire('click');await tick();assert(f.calls.length===2 && f.calls[0]===f.calls[1] && f.retry.hidden,'retry fetches same page and clears failure');
 output('PASS browsing interaction regressions');
}
run().catch(error=>{output(error.stack||String(error));if(typeof quit==='function')quit(1);else process.exitCode=1;});
