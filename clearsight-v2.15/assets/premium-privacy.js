/* V2.14 privacy and source-monitoring presentation only.
 * Canonical route handlers, fields, commands and records remain authoritative. */
'use strict';

const PV214_FAMILIES=Object.freeze({
 privacy:'workspace',activity:'record','activity-wizard':'capture','activity-version':'snapshot',
 dpia:'record','dpia-wizard':'capture','privacy-case':'case','case-intake':'capture',
 integrations:'monitoring','data-health':'health'
});

function pv214Family(route){return PV214_FAMILIES[route]||'';}
function pv214Name(element,label,role){
 if(!element)return;
 if(role&&!element.hasAttribute('role'))element.setAttribute('role',role);
 if(!element.hasAttribute('aria-label')&&!element.hasAttribute('aria-labelledby'))element.setAttribute('aria-label',label);
}
function pv214Refresh(){
 const main=document.querySelector('#main-content');if(!main)return;
 const family=pv214Family(current.name);
 if(family)main.dataset.pv214=family;else delete main.dataset.pv214;
 if(!family)return;

 // Keep the real editing table intact while exposing its horizontal scroll
 // region to keyboard users. Existing names and tab order are respected.
 for(const region of main.querySelectorAll('.j-table-scroll')){
  if(!region.querySelector('.j-edit-table'))continue;
  const heading=region.closest('.j-section')?.querySelector('h2')?.textContent?.trim();
  pv214Name(region,(heading||'Privacy record')+' — table','region');
  if(!region.hasAttribute('tabindex'))region.setAttribute('tabindex','0');
 }
 for(const [selector,label] of [
  ['.j-actions','Privacy draft actions'],
  ['.pv211-case-actions','Case preparation actions']
 ])for(const region of main.querySelectorAll(selector))pv214Name(region,label,'group');

 for(const [kind,label] of [
  ['activity-decision','Independent processing record review'],
  ['dpia-decision','Independent privacy impact review'],
  ['case-decision','Independent privacy case review'],
  ['case-update','Privacy case preparation']
 ])for(const form of main.querySelectorAll('form[data-j-submit="'+kind+'"]'))pv214Name(form,label);
}

const pv214PriorRender=render;
render=function(...args){const result=pv214PriorRender.apply(this,args);pv214Refresh();return result;};
window.CLEAR_SIGHT_PREMIUM_PRIVACY=Object.freeze({version:'2.14',family:pv214Family,refresh:pv214Refresh,presentationOnly:true});
