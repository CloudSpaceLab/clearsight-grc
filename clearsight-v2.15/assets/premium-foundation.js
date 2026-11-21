/* ClearSight V2.13 · display preferences and shared presentation.
 * Presentation never grants authority, changes scope, or rebuilds an open form.
 * Theme uses the existing S.theme; density is a browser-workspace preference.
 */
'use strict';

function pd213Preferences(){
 return {theme:S.theme==='dark'?'dark':'light',density:S.displayPreferences?.density==='compact'?'compact':'comfortable'};
}
function pd213Summary(value){return (value.theme==='dark'?'Dark':'Light')+' appearance · '+(value.density==='compact'?'Compact':'Comfortable')+' density. Adjust readability and the space between records.';}
function pd213ApplyAttributes(){
 const value=pd213Preferences();
 document.documentElement.dataset.theme=value.theme;
 document.documentElement.dataset.density=value.density;
 for(const control of $$('.topbar [data-action="pd213-display"]')){
  control.setAttribute('aria-label','Display preferences');
  control.setAttribute('title','Display preferences · '+(value.theme==='dark'?'Dark':'Light')+' · '+(value.density==='compact'?'Compact':'Comfortable'));
 }
 for(const summary of $$('[data-pd213-summary]'))summary.textContent=pd213Summary(value);
 return value;
}
function pd213SetPreferences(value){
 if(!value||!['light','dark'].includes(value.theme)||!['comfortable','compact'].includes(value.density))throw new Error('Choose a valid theme and display density.');
 S.theme=value.theme;
 S.displayPreferences={...(S.displayPreferences||{}),density:value.density};
 pd213ApplyAttributes();
 save();
 return pd213Preferences();
}
function pd213PreferenceFields(){
 const value=pd213Preferences();
 const choice=(name,id,label,description,example,selected)=>`<label class="pd213-choice" for="${id}"><input type="radio" id="${id}" name="${name}" value="${example}" ${selected?'checked':''} required><span class="pd213-choice-copy"><strong>${label}</strong><small>${description}</small></span><span class="pd213-swatch ${example}" aria-hidden="true"><i></i><i></i><i></i></span></label>`;
 return `<div class="pd213-preferences"><fieldset><legend>Appearance</legend><p id="pd213-theme-help">Choose the surface that is comfortable for your environment.</p><div class="pd213-choice-grid" aria-describedby="pd213-theme-help">${choice('pfTheme','pd213-light','Light','Crisp surfaces and clear contrast.','light',value.theme==='light')}${choice('pfTheme','pd213-dark','Dark','Deep surfaces with the same status colours.','dark',value.theme==='dark')}</div></fieldset><fieldset><legend>Information density</legend><p id="pd213-density-help">Compact tightens tables and desktop toolbars. Touch controls and longer forms keep their comfortable spacing.</p><div class="pd213-choice-grid" aria-describedby="pd213-density-help">${choice('pfDensity','pd213-comfortable','Comfortable','More room between records and actions.','comfortable',value.density==='comfortable')}${choice('pfDensity','pd213-compact','Compact','More records in the same desktop view.','compact',value.density==='compact')}</div></fieldset><p class="pd213-preference-note">Applies to this browser workspace. Your current form, filters and position stay in place.</p></div>`;
}
actions['pd213-display']=()=>{
 if($('#dialog')?.open){toast('Close the current dialog before changing display preferences.');return false;}
 openDialog('Display preferences','Adjust the workspace to the way you work.',pd213PreferenceFields(),'Apply preferences',form=>{
  const data=new FormData(form);
  pd213SetPreferences({theme:data.get('pfTheme'),density:data.get('pfDensity')});
  toast(storageOK?'Display preferences saved.':'Display preferences applied for this session.');
 });
};
// The existing quick theme command also avoids a route render or a draft reset.
actions.theme=()=>{
 const value=pd213Preferences();
 pd213SetPreferences({...value,theme:value.theme==='dark'?'light':'dark'});
};

const pd213PriorShell=shell;
shell=function(){
 pd213PriorShell();
 const control=$('.topbar [data-action="theme"]');
 if(control){control.dataset.action='pd213-display';control.classList.add('pd213-display');control.setAttribute('aria-haspopup','dialog');}
 for(const item of $$('.sidebar .nav-link')){
  const label=item.querySelector('span')?.textContent?.trim();
  if(label){item.setAttribute('aria-label',label);item.setAttribute('title',label);}
 }
 const searchHint=$('.search-trigger kbd');if(searchHint)searchHint.textContent='⌘ / Ctrl K';
 pd213ApplyAttributes();
};

const pd213PriorSettings=settingsView;
settingsView=function(){
 const html=pd213PriorSettings();
 if(current.params.get('tab')&&!['demo'].includes(current.params.get('tab')))return html;
 const value=pd213Preferences();
 return html+`<section class="panel pd213-settings" aria-labelledby="pd213-settings-title"><div><span class="eyebrow">Your workspace</span><h2 id="pd213-settings-title">Display preferences</h2><p data-pd213-summary>${pd213Summary(value)}</p></div>${btn('Adjust display','pd213-display','aria-haspopup="dialog"')}</section>`;
};

const pd213PriorRender=render;
render=function(){
 pd213PriorRender();
 const main=$('#main-content');if(main)main.dataset.csRoute=current.name;
 pd213ApplyAttributes();
};
window.CLEAR_SIGHT_PREMIUM=Object.freeze({version:'2.13',preferences:pd213Preferences,apply:pd213SetPreferences});
