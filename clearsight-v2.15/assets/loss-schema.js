/* Structural mapping only. No customer names, amounts or transaction narratives
   from the reference workbook are distributed with the synthetic demonstration. */
'use strict';
const CS_LOSS_REFERENCE=Object.freeze({
 version:'LOSS-DATABASE/1.0',
 source:{file:'LOSS DATA BASE.xlsx',sheet:'Sheet1',archive:'Ops Risk (2).zip',url:'https://drive.google.com/file/d/1tMdSTVFHPv0MjFrH5Uj6pURc2n4MYwgu/view'},
 columns:[
  ['ACCT_NAME','accountName','Account classification; distinct from loss group.'],
  ['LOSS GROUP','lossGroup','Bank loss grouping; independent of risk taxonomy.'],
  ['Month','reportingPeriod','Reporting month; derived from explicit reporting period.'],
  ['Month Number','reportingPeriod','Month number derived from reporting period.'],
  ['Quarter','reportingPeriod','Quarter label derived from reporting period.'],
  ['Amount','gross','Gross booked loss in the original currency, stored as integer minor units.'],
  ['TRAN_PARTICULAR','transactionParticulars','Transaction particulars and event context.'],
  ['Branch','orgScopeId','Exact organizational location; mapped explicitly during import.'],
  ['Regional Bank','region','Regional bank at the time of the record.'],
  ['Directorate','directorate','Directorate at the time of the record.'],
  ['YEAR','reportingPeriod','Year derived from reporting period.'],
  ['QTR NUMBER','reportingPeriod','Quarter number derived from reporting period.'],
  ['DATE OF RECOVERY','recoveries[].date','Dated recovery receipt; a narrative alone is not a verified receipt.'],
  ['DATE OF OCCURRENCE','date','Date the loss event occurred.'],
  ['DATE OF RECOGNITION','recognizedOn','Date the financial loss was recognized.'],
  ['ROOT CAUSE ANALYSIS','rootCause','Cause analysis or an accountable investigation plan.'],
  ['RECOVERY DATA','recoverability.note','Source recovery narrative, retained separately from received money.'],
  ['CURRENCY OF LOSS','currency','Original currency; no implicit conversion.']
 ],
 accounts:['CASH SHORTAGES','FRAUD AND FORGERIES','LEGAL EXPENSES','PENALTY','PRIOR YEAR EXPENSES'],
 groups:['CASH SHORTAGES','FRAUD & FORGERIES','LEGAL LOSSES','PENALTY','INCOME REFUNDS'],
 currencies:['NGN','USD','EUR','GBP','ZAR'],
 recoveryStates:['Under assessment','Recovery pursued','Not recoverable','No recovery expected'],
 principles:['Reporting period, occurrence and recognition are different dimensions.','A repeated row is a duplicate candidate, not automatic evidence of another event.','Recovery claims need a dated amount, a source reference and independent verification before reducing net loss.']
});
