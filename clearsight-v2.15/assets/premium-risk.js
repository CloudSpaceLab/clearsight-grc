/* V2.14 operational record presentation. No source record, command, route
 * parameter, permission or review logic is replaced by this adapter. */
'use strict';
const PR214_ROUTES=Object.freeze(['risks','risk','risk-edit','rcsa','rcsa-review','indicators','indicator','losses','controls','control','control-editor','control-test','kri-returns','kri-library','vendors']);
const PR214_FILTER_NAMES=Object.freeze({risks:'Risk register filters',rcsa:'Assessment cycle filters','rcsa-review':'Assessment context and review mode',indicators:'Indicator register filters',losses:'Loss register filters',controls:'Control library filters','kri-returns':'Branch return filters','kri-library':'KRI definition filters',vendors:'Vendor engagement filters'});
function pr214Name(element,label,role='group'){
 if(!element)return;
 if(!element.hasAttribute('role'))element.setAttribute('role',role);
 if(!element.hasAttribute('aria-label')&&!element.hasAttribute('aria-labelledby'))element.setAttribute('aria-label',label);
}
function pr214Refresh(){
 const main=document.querySelector('#main-content');if(!main)return;
 if(!PR214_ROUTES.includes(current.name)){delete main.dataset.pr214;return;}
 main.dataset.pr214=current.name;
 for(const toolbar of main.querySelectorAll('.toolbar,form.l29-filters'))pr214Name(toolbar,PR214_FILTER_NAMES[current.name]||'Record controls');
 for(const actions of main.querySelectorAll('.wf29-actions'))pr214Name(actions,'Record actions');
 for(const [selector,label] of [
  ['.heatmap-layout','Residual risk matrix: likelihood by impact'],
  ['.op211-comparison','Saved and proposed source comparison'],
  ['.k28-history-chart','Dated indicator observations']
 ])for(const region of main.querySelectorAll(selector)){
  pr214Name(region,label,'region');
  if(!region.hasAttribute('tabindex'))region.setAttribute('tabindex','0');
 }
}
const pr214PriorRender=render;
render=function(...args){const result=pr214PriorRender.apply(this,args);pr214Refresh();return result;};
window.CLEAR_SIGHT_PREMIUM_RISK=Object.freeze({version:'2.14',presentationOnly:true,routes:PR214_ROUTES,refresh:pr214Refresh});
