/* V2.13 form presentation. This adapter adds styling scope and accessible
 * reading/action groups only. It does not replace fields, commands or records. */
'use strict';

const PF213_FAMILIES=Object.freeze({
 'program':'response','assessment':'response','form-response':'response','form-builder':'response',
 'assessment-review':'review','assessment-history':'review','requirement-review':'review',
 'approval':'review','approval-request':'review','control-test':'review','vendor-review':'review','role-change-review':'review',
 'kri-return':'kri','kri-cycle':'kri',
 'loss-intake':'loss','loss':'loss','loss-import':'loss',
 'vendor-intake':'vendor','vendor-onboarding':'vendor','vendor-requests':'vendor','vendor':'vendor',
 'document':'document','document-upload':'document','document-request':'document','evidence-review':'document',
 'external-assessments':'external','external-assessment-request':'external','setup':'entry'
});

function pf213Family(route){return PF213_FAMILIES[route]||'';}
function pf213Group(element,label,role='group'){
 if(!element)return;
 if(!element.hasAttribute?.('role'))element.setAttribute('role',role);
 if(!element.hasAttribute?.('aria-label')&&!element.hasAttribute?.('aria-labelledby'))element.setAttribute('aria-label',label);
}
function pf213Refresh(){
 const main=document.querySelector('#main-content');if(!main)return;
 const family=pf213Family(current.name);if(family)main.dataset.pf213=family;else delete main.dataset.pf213;
 if(!family)return;
 for(const [selector,label] of [
  ['.l29-footer','Loss entry actions'],['.vo-footer','Onboarding actions'],
  ['.k28-section-footer','KRI section navigation'],['.k28-submit-bar','KRI return actions'],
  ['.dm212-version-bar','Document version controls']
 ])for(const element of main.querySelectorAll(selector))pf213Group(element,label);
 for(const [selector,label] of [['.ax211-response-nav nav','Response questions'],['.k28-section-nav nav','KRI sections']])
  for(const element of main.querySelectorAll(selector))if(!element.hasAttribute('aria-label'))element.setAttribute('aria-label',label);
 for(const preview of main.querySelectorAll('.doc-preview')){
  pf213Group(preview,'Selected document preview','region');
  if(!preview.hasAttribute('tabindex'))preview.setAttribute('tabindex','0');
 }
 for(const table of main.querySelectorAll('.ix-return-table')){
  const grid=table.closest('.table-wrap');if(!grid)continue;
  pf213Group(grid,'KRI data entry grid','region');
  if(!grid.hasAttribute('tabindex'))grid.setAttribute('tabindex','0');
 }
}

/* The recipient file has no dependency on the bank stylesheet or its state.
 * Only constant CSS is added to the existing head. JSON, CSP, source scripts,
 * field markup, download behavior and all validation stay byte-for-byte intact. */
const PF213_RECIPIENT_STYLE=`
:root{--surface:#ffffff;--soft:#f6f8fb;--ink:#17263d;--muted:#56677e;--line:#dce3ed;--blue:#245dd8;--red:#b6304b;--amber:#8b590f;--teal:#137664;font:15px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:var(--ink);background:var(--soft);color-scheme:light}
body{color:var(--ink);background:var(--soft);text-rendering:optimizeLegibility}
.page{max-width:1320px;padding:40px clamp(24px,4vw,56px)}
.brand{font-size:12px;font-weight:700;letter-spacing:.12em;color:var(--muted)}
.heading{gap:36px;margin:24px 0 28px;align-items:flex-start}
.heading h1{font-size:clamp(28px,3.2vw,42px);font-weight:650;letter-spacing:-.04em;line-height:1.15;max-width:23ch;overflow-wrap:anywhere;text-wrap:balance}
.heading p{font-size:14px;line-height:1.65;color:var(--muted);max-width:72ch;margin-top:14px}
.due{padding:16px 20px;min-width:190px;border:1px solid var(--line);border-radius:16px;background:var(--surface);font-size:12px;line-height:1.6;text-align:left}
.due strong{font-size:20px;line-height:1.3;color:var(--blue);font-weight:650;letter-spacing:-.025em}
.person{margin-top:26px;padding-top:20px;gap:12px 22px;line-height:1.6;font-size:13px;border-color:var(--line)}
.person small{font-size:12px;color:var(--muted);overflow-wrap:anywhere}
.layout{grid-template-columns:minmax(0,1fr) 292px;gap:28px;align-items:start;padding-top:8px}
.layout>aside,.main{min-width:0}
.instructions,.question,.side-card{border:1px solid var(--line);border-radius:18px;background:var(--surface);box-shadow:0 2px 6px #16355603}
.instructions,.question{padding:28px 30px;margin-bottom:20px}
.instructions h2{font-size:16px;letter-spacing:-.015em;margin-bottom:14px}
.instructions p{max-width:80ch;font-size:14px;line-height:1.75;color:var(--ink)}
.question{scroll-margin-top:24px}
.question-title{gap:14px;align-items:flex-start;margin-bottom:18px}
.question-title>span{width:30px;min-width:30px;height:30px;border-radius:9px;color:var(--muted);background:var(--soft);font-size:12px;font-variant-numeric:tabular-nums}
.question-title h2{font-size:17px;line-height:1.5;font-weight:650;letter-spacing:-.015em;margin:1px 0 0;overflow-wrap:anywhere}
.question-title small{font-size:11px;letter-spacing:0;line-height:1.5;font-weight:500;color:var(--muted);white-space:normal}
.help,.subtle{font-size:13px;line-height:1.65;color:var(--muted)}
.help{margin:0 0 20px;max-width:76ch}
label{font-size:13px;line-height:1.6;font-weight:600;margin:16px 0;color:var(--ink)}
input,select,textarea{border:1px solid var(--line);border-radius:10px;min-height:44px;padding:11px 13px;background:var(--surface);color:var(--ink);font-size:15px;line-height:1.55;box-shadow:inset 0 1px 2px #16355603}
textarea{min-height:132px;resize:vertical}
input:hover,select:hover,textarea:hover{border-color:#aabacd}
input[type=checkbox]{min-height:20px;min-width:20px;height:20px;margin:3px 10px 0 0;accent-color:var(--blue)}
.check{align-items:flex-start;gap:3px;font-weight:400;line-height:1.65}
.check:has(input:checked){color:var(--ink)}
button,.upload{min-height:42px;line-height:1.5;font-size:13px;font-weight:600;border-radius:10px;border-color:var(--line);background:var(--surface);color:var(--ink);padding:10px 14px}
button:hover,.upload:hover{background:var(--soft);border-color:#aabacd}
button.primary{background:var(--blue);border-color:var(--blue);color:var(--surface);min-height:46px;padding:12px 20px;margin-top:20px}
button.primary:hover{background:#1b4db7;border-color:#1b4db7}
:focus-visible{outline:3px solid #245dd8;outline-offset:3px}
.upload:focus-within{outline:3px solid #245dd8;outline-offset:3px}
button:disabled{opacity:.55;cursor:default}
.side-card{padding:22px;margin-bottom:18px;border-radius:16px;box-shadow:none}
.side-card h2{font-size:14px;line-height:1.5;letter-spacing:-.01em;margin-bottom:14px}
.side-card p{font-size:13px;line-height:1.65;color:var(--muted)}
.side-card nav{max-height:300px;overscroll-behavior:contain;scroll-padding-block:8px}
.side-card nav button{min-height:40px;padding:10px 12px;font-size:12px;line-height:1.5;margin:2px 0;color:var(--muted);border:1px solid transparent;border-radius:9px}
.side-card nav button:hover,.side-card nav button:focus-visible{color:var(--blue);background:#f0f4fb;border-color:var(--line)}
.document-choice{gap:12px}
.document-choice select{min-width:180px}
.document-choice .upload{min-height:44px;flex-shrink:0}
.question-doc{padding-top:20px;margin-top:24px;border-color:var(--line)}
.question-doc label{margin-bottom:8px}
.file-row,.reference{padding:14px 0;font-size:13px;line-height:1.6;border-color:var(--line);overflow-wrap:anywhere}
.file-row small,.reference small{font-size:12px;line-height:1.6;color:var(--muted)}
.file-row button{min-height:36px;padding:6px 10px;font-size:12px;margin-top:8px}
.return-channel{font-size:13px;line-height:1.7;background:var(--soft);border:1px solid var(--line);border-radius:12px;padding:18px;margin-top:22px;overflow-wrap:anywhere}
.alert,.feedback{background:#fff8e9;border:1px solid #e4cf9f;color:#624614;border-radius:12px;font-size:13px;line-height:1.7;padding:18px 20px}
.feedback strong{font-size:13px}
.question:has(#confirm-name){background:#f8faff;border-color:#cbd9ec}
.table-scroll{border:1px solid var(--line);border-radius:12px;overscroll-behavior:contain}
table{font-size:13px;line-height:1.6}
th{background:var(--soft);font-size:12px;line-height:1.5;font-weight:650;color:var(--muted)}
th,td{padding:13px 14px;border-color:var(--line)}
td input,td select{min-width:150px;font-size:14px;min-height:42px}
footer{font-size:12px;line-height:1.75;color:var(--muted);margin-top:18px}
@media(max-width:1000px){.layout{grid-template-columns:minmax(0,1fr) 254px;gap:20px}.page{padding:28px}.instructions,.question{padding:24px}.side-card{padding:18px}}
@media(max-width:850px){input,select,textarea,td input,td select{font-size:16px;min-height:44px}.layout{grid-template-columns:1fr;padding-top:0}.layout aside{display:grid;grid-template-columns:1fr 1fr;gap:16px}.heading{gap:22px}.heading h1{max-width:none}.side-card{margin:0}.person small{margin-left:0}.question:has(button[type=submit]){position:static}}
@media(max-width:560px){.page{padding:20px 16px}.heading{flex-direction:column;gap:20px;margin:20px 0 24px}.heading h1{font-size:30px;line-height:1.2}.due{width:100%;padding:14px 16px}.person{gap:6px;align-items:flex-start;flex-direction:column;margin-top:22px}.instructions,.question{padding:20px 18px;border-radius:15px;margin-bottom:16px}.question-title{gap:10px}.question-title h2{font-size:16px}.question-title small{display:block;margin:5px 0 0}.question-title>span{width:26px;min-width:26px;height:26px}.layout aside{display:block}.side-card{margin-bottom:16px}.help,.subtle{font-size:13px}input,select,textarea{font-size:16px;min-height:46px}button,.upload{min-height:44px}.document-choice{display:block}.document-choice .upload{margin-top:10px}.question button.primary{width:100%}.table-scroll{border-radius:10px}.return-channel{padding:16px}footer{font-size:12px}}
@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important;transition:none!important}}
@media(forced-colors:active){.instructions,.question,.side-card,.due,.return-channel{border:1px solid CanvasText;box-shadow:none}button.primary{border:2px solid ButtonText}.question-title>span{border:1px solid CanvasText}}
@media print{:root,body{background:white;color:black}.page{max-width:none;padding:0}.layout{display:block}.layout aside{display:block}.side-card nav{display:none}.instructions,.question{box-shadow:none;border-color:#aaa;break-inside:avoid}.question-title h2{font-size:15px}input,select,textarea{border-color:#aaa;box-shadow:none}button,.upload{display:none!important}.table-scroll{overflow:visible}.heading h1{font-size:28px}.due{background:white}}
`;

const pf213OriginalRecipientHTML=ex212RecipientHTML;
ex212RecipientHTML=function(payload){
 const html=pf213OriginalRecipientHTML(payload),anchor=html.indexOf('</head>');
 if(anchor<0||html.includes('id="pf213-recipient-style"'))return html;
 return html.slice(0,anchor)+'<style id="pf213-recipient-style">'+PF213_RECIPIENT_STYLE+'</style>'+html.slice(anchor);
};

const pf213PriorRender=render;
render=function(...args){const result=pf213PriorRender.apply(this,args);pf213Refresh();return result;};
window.CLEAR_SIGHT_PREMIUM_FORMS=Object.freeze({version:'2.13',family:pf213Family,refresh:pf213Refresh});
