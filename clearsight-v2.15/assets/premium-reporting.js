/* V2.13 reporting presentation only. Existing view handlers, source checks,
 * populations, amounts and snapshot renderers remain authoritative. */
'use strict';

function pr213WrapView(view,surface){
 return function(...args){return `<div class="pr213-report pr213-report--${surface}">${view.apply(this,args)}</div>`;};
}
homeView=pr213WrapView(homeView,'dashboard');
reportsView=pr213WrapView(reportsView,'insights');
for(const [route,surface] of [['loss-insights','losses'],['assessment-insights','responses'],['insight-detail','sources'],['report-builder','builder'],['report-pack','pack']]){
 const previous=window.CS_JOURNEYS[route];if(typeof previous==='function')window.CS_JOURNEYS[route]=pr213WrapView(previous,surface);
}

// Closed analytical details should not hide methodology or chart equivalents
// in a printed report. Restore exactly the reader's prior disclosure state.
// This touches report DOM nodes only; it never writes application state or
// changes the captured/frozen report markup returned by its source renderer.
let pr213PrintDisclosureState=null;
function pr213BeforePrint(){
 if(pr213PrintDisclosureState!==null)return;
 const selector='.pr213-report :is(details.br-chart-data,details.br-method,details.br-kri-group,details.arp-history)';
 pr213PrintDisclosureState=Array.from(document.querySelectorAll(selector))
  .filter(node=>!node.closest('.br-builder')||node.closest('.br-frozen-pack'))
  .map(node=>({node,open:node.open}));
 for(const item of pr213PrintDisclosureState)item.node.open=true;
}
function pr213AfterPrint(){
 if(pr213PrintDisclosureState===null)return;
 for(const item of pr213PrintDisclosureState)item.node.open=item.open;
 pr213PrintDisclosureState=null;
}
window.addEventListener('beforeprint',pr213BeforePrint);
window.addEventListener('afterprint',pr213AfterPrint);
window.CLEAR_SIGHT_PREMIUM_REPORTING=Object.freeze({version:'2.13',presentationOnly:true});
