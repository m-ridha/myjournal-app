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
 const listeners={};const node={children:[],hidden:false,disabled:false,textContent:'',addEventListener:(name,fn)=>{listeners[name]=fn;},fire:(name)=>listeners[name]?.(),setAttribute(name,value){this[name]=value;},classList:{add(){}},appendChild(child){this.children.push(child);},replaceChildren(...children){this.children=children;}};
 Object.defineProperty(node,'innerHTML',{set(){this.children=[];}});return node;
}
const payload=(page=1,hasNext=false,items=[])=>({ok:true,json:async()=>({page,pageSize:10,hasNext,items})});
function setup(search='',initial=payload(),calendarMode=false,listLimit=10) {
 const auth=element();auth.hidden=true;
 const list=element(),status=element(),retry=element();
 list.dataset={listLimit:String(listLimit)};
 const fields={'[data-auth-expired]':auth,'[data-entry-list]':list,'[data-entries-status]':status,'[data-entries-retry]':retry};
 const calendar=element(),label=element(),grid=element(),prev=element(),next=element(),today=element(),calendarStatus=element();
 calendar.querySelector=s=>({'[data-calendar-month]':label,'[data-calendar-status]':calendarStatus,'[data-calendar-grid]':grid,'[data-calendar-prev]':prev,'[data-calendar-next]':next,'[data-calendar-today]':today}[s]||null);
 if(calendarMode){delete fields['[data-entry-list]'];fields['[data-calendar]']=calendar;}
 const events={};globalThis.document={documentElement:{lang:'en'},body:{dataset:{messages:'{}',today:'2026-10-09'}},querySelector:s=>fields[s]||null,addEventListener:(name,fn)=>{events[name]=fn;},createElement:tag=>Object.assign(element(),{tagName:tag.toUpperCase()})};
 globalThis.window={location:{search,href:''},addEventListener(){}};
 const calls=[],queue=[initial];globalThis.fetch=async(url)=>{calls.push(url);if(!queue.length)throw new Error('unplanned request');return queue.shift();};
 if(typeof require==='function')(0,eval)(require('node:fs').readFileSync('assets/static/js/app.js','utf8'));else load('assets/static/js/app.js');
 events.DOMContentLoaded();return {auth,list,status,retry,calls,queue,label,grid,prev,next,today,calendarStatus};
}
async function run() {
 let custom=setup('',{ok:true,json:async()=>({page:1,pageSize:5,items:[]})},false,5);await tick();assert(custom.calls[0]==='/api/v1/entries?page=1&size=5' && custom.status.textContent==='', 'rendered limit controls requests and response validation');
 let f=setup('',payload(1,true,[{date:'2026-01-02',preview:'<script>plain text</script>'},{date:'2025-12-31',preview:'previous year'}]));await tick();
 assert(f.calls[0]==='/api/v1/entries?page=1&size=10','Entries requests only the latest ten');
 const row=f.list.children[0].children[1];assert(row.tagName==='A' && row.href==='/entries/2026-01-02','whole entry row is a native navigation link');assert(row.children[1].children[1].tagName==='SPAN','Open entry remains a label without a nested interactive link');
 assert(f.list.children.length===2 && f.list.children[0].children[0].textContent==='2026','group entries newest-year first');
 assert(f.list.children[0].children[1].children[1].children[0].textContent==='<script>plain text</script>','previews remain plain text');
 for(const search of ['?page=2','?page=3','?page=10000','?page=0','?page=-1','?page=10001','?page=1.5','?page=abc','?size=100']) {f=setup(search);await tick();assert(f.calls.length===1&&f.calls[0]==='/api/v1/entries?page=1&size=10','URL parameters cannot expand recent entries browsing');}
 f=setup('?page=2',{ok:false,status:503});await tick();assert(f.list.children[0].textContent==='entries.unavailable' && !f.retry.hidden,'failed recent entries offers retry');
 f.queue.push(payload(1,false,[{date:'2026-01-01',preview:'recovered'}]));f.retry.fire('click');await tick();assert(f.calls.length===2 && f.calls[0]===f.calls[1] && f.retry.hidden,'retry stays on first page and clears failure');
 f=setup('',payload(1,true,Array.from({length:11},()=>({date:'2026-01-01',preview:'unexpected'}))));await tick();assert(f.list.children[0].textContent==='entries.unavailable','oversized responses do not render more than ten entries');
 const deferred=()=>{let resolve,reject;const promise=new Promise((a,b)=>{resolve=a;reject=b;});return {promise,resolve,reject};};
 const pendingList=deferred();f=setup('',pendingList.promise);
 assert(f.status.textContent==='entries.loading' && f.list.children.length===0 && f.list['aria-busy']==='true','pending entries request uses only the shared loading status');
 pendingList.resolve(payload());await tick();assert(f.status.textContent==='' && f.list['aria-busy']==='false','loaded entries replace loading status');
 const calendarData=(year,month)=>({ok:true,json:async()=>({year,month,weeks:[[{date:`${year}-${String(month).padStart(2,'0')}-01`,inMonth:true,isFuture:false,hasEntry:false,isToday:false}]]})});
 const old=deferred();f=setup('',old.promise,true);f.queue.push(calendarData(2026,9));f.prev.fire('click');await tick();
 const latestLabel=f.label.textContent;assert(latestLabel.includes('September') && f.grid.children.length===8,'latest calendar response renders one grid');
 old.resolve(calendarData(2026,10));await tick();assert(f.label.textContent===latestLabel && f.grid.children.length===8,'older successful calendar response must not replace or append to latest grid');
 const obsolete=deferred();f=setup('',obsolete.promise,true);f.queue.push(calendarData(2026,9));f.prev.fire('click');await tick();obsolete.reject(new Error('old failure'));await tick();
 assert(f.label.textContent.includes('September') && f.grid.children.length===8,'obsolete calendar failure must not overwrite current success');
 f=setup('',calendarData(2026,10),true);await tick();f.queue.push({ok:false,status:503});f.prev.fire('click');await tick();assert(f.calendarStatus.textContent==='calendar.unavailable' && f.grid.children.length===0 && f.grid['aria-busy']==='false','current failure clears grid and loading state');
 f.queue.push(calendarData(2026,10));f.today.fire('click');await tick();assert(f.label.textContent.includes('October'),'Today recovers calendar after a failed navigation');
 const slowBody=deferred();f=setup('',{ok:true,json:()=>slowBody.promise},true);await tick();f.queue.push(calendarData(2026,9));f.prev.fire('click');await tick();slowBody.resolve({year:2026,month:10,weeks:[]});await tick();assert(f.label.textContent.includes('September'),'stale JSON parsing completion cannot replace latest month');
 f=setup('',{ok:false,status:401});await tick();assert(!f.auth.hidden && window.location.href==='', 'entries 401 offers reconnect without redirect');
 f.queue.push(payload());f.retry.fire('click');await tick();assert(f.auth.hidden,'successful explicit list retry clears reconnect banner');
 f=setup('',{ok:false,status:401},true);await tick();assert(!f.auth.hidden && f.grid.children.length===0,'calendar 401 offers shared reconnect');
 const obsoleteAuth=deferred();f=setup('',obsoleteAuth.promise,true);f.queue.push(calendarData(2026,9));f.prev.fire('click');await tick();obsoleteAuth.resolve({ok:false,status:401});await tick();assert(f.auth.hidden && f.label.textContent.includes('September'),'obsolete calendar 401 cannot replace current success or show expired prompt');
 f=setup('',calendarData(2026,10),true);await tick();assert(f.next.disabled,'next month disabled at request-resolved current month');
 f.next.fire('click');await tick();assert(f.calls.length===1,'disabled next cannot request a future month');
 f.queue.push(calendarData(2026,9));f.prev.fire('click');await tick();assert(!f.next.disabled,'next enabled for past months');
 f.queue.push(calendarData(2026,10));f.next.fire('click');await tick();assert(f.next.disabled && f.calls.length===3,'next returns to current month and stops');
 output('PASS browsing interaction regressions');
}
run().catch(error=>{output(error.stack||String(error));if(typeof quit==='function')quit(1);else process.exitCode=1;});
