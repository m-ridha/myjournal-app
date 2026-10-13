// Dependency-free month/year picker regressions; Node.js or macOS JavaScriptCore.
const output = typeof print === 'function' ? print : console.log;
const assert = (ok, message) => { if (!ok) throw new Error(message); };
function element() {
  const events = {};
  return {children: [], attrs: {}, hidden: false, disabled: false, textContent: '',
    addEventListener(name, fn) {events[name] = fn;},
    fire(name, event = {}) {events[name]?.(event);},
    setAttribute(name, value) {this.attrs[name] = value;},
    appendChild(node) {this.children.push(node);},
    replaceChildren() {this.children = [];},
    contains(node) {return node === this || this.children.some(child => child.contains(node));},
    focus() {document.activeElement = this;},
  };
}
const trigger=element(), panel=element(), heading=element(), previous=element(), next=element(), close=element(), items=element();
panel.hidden=true;panel.children=[heading,previous,next,close,items];
const nodes={'[data-month-trigger]':trigger,'[data-month-picker]':panel,'[data-month-year]':heading,'[data-month-prev]':previous,'[data-month-next]':next,'[data-month-close]':close,'[data-month-options]':items};
const calendar=element();calendar.children=[trigger,panel];calendar.querySelector=s=>nodes[s];
const events={};globalThis.document={documentElement:{lang:'en'},activeElement:trigger,createElement:element,addEventListener(name,fn){events[name]=fn;}};
globalThis.window={};
if(typeof require==='function')(0,eval)(require('node:fs').readFileSync('assets/static/js/calendar-picker.js','utf8'));else load('assets/static/js/calendar-picker.js');
const selections=[];
const picker=window.mjInitMonthPicker(calendar,{month:'2026-10',max:'2026-10',text:key=>key,onSelect:month=>selections.push(month)});
trigger.fire('click');
assert(!panel.hidden && items.children.length===12 && document.activeElement===items.children[9],'opening focuses selected month');
assert(items.children[10].disabled && items.children[11].disabled && next.disabled,'future months/year disabled');
items.children[10].fire('click');assert(selections.length===0,'disabled future month cannot select');
assert(items.children[9].attrs['aria-pressed']==='true','selection conveyed to assistive technology');
next.fire('click');assert(heading.textContent==='2026','disabled year navigation cannot advance');
const key=value=>panel.fire('keydown',{key:value,target:document.activeElement,preventDefault(){}});
key('ArrowLeft');assert(document.activeElement===items.children[8],'left arrow moves month focus');
key('End');assert(document.activeElement===items.children[9],'End stops at last enabled month');
key('ArrowDown');assert(document.activeElement===items.children[9],'arrow navigation never moves into future months or past grid bounds');
key('Escape');assert(panel.hidden && document.activeElement===trigger && selections.length===0,'Escape closes without selecting');
trigger.fire('click');previous.fire('click');assert(heading.textContent==='2025' && items.children.every(item=>!item.disabled),'past year has all twelve months');
heading.fire('click');assert(items.children.length===12 && items.children.some(item=>item.textContent==='2026'),'year heading opens twelve-year grid');
assert(items.children.filter(item=>Number(item.textContent)>2026).every(item=>item.disabled),'future years disabled');
previous.fire('click');assert(heading.textContent==='2005 – 2016','year ranges page backward by twelve');
next.fire('click');assert(heading.textContent==='2017 – 2028' && next.disabled,'year ranges return to the last allowed page');
const pastYear=items.children.find(item=>item.textContent==='2024');pastYear.fire('click');
events.click({target:pastYear,composedPath:()=>[pastYear,items,panel]});
assert(!panel.hidden,'year selection stays open when its replaced button event bubbles');
assert(heading.textContent==='2024' && items.children[0].textContent==='Jan','choosing year returns to months without selecting');
items.children[1].fire('click');assert(panel.hidden && selections.join()==='2024-02' && document.activeElement===trigger,'month selection closes and emits YYYY-MM once');
picker.setMonth('0001-01');trigger.fire('click');assert(previous.disabled,'year one is the lower bound');
heading.fire('click');assert(previous.disabled && items.children[0].textContent==='1','first year range starts at one');
key('Escape');picker.setMonth('2026-10');trigger.fire('click');
events.click({target:element()});assert(panel.hidden,'outside click dismisses');
trigger.fire('click');panel.fire('focusout',{relatedTarget:element()});assert(panel.hidden,'Tab/focus leaving dismisses without trapping');
trigger.fire('click');close.fire('click');assert(panel.hidden && document.activeElement===trigger,'close restores trigger focus');
assert(selections.length===1,'browsing years and dismissal never select a month');
document.documentElement.lang='id';picker.setMonth('2026-10');trigger.fire('click');
assert(items.children[9].attrs['aria-label'].includes('Oktober'),'month names follow Indonesian locale');
assert(items.children[10].disabled && items.children[11].disabled,'localized view preserves future bounds');
close.fire('click');
output('PASS calendar-picker interaction regressions');
