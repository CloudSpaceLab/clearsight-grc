import { useState, type ReactNode } from "react";
import {
  ActionLink,
  Button,
  Card,
  CheckboxField,
  DataTable,
  EmptyState,
  FilterBar,
  FocusedSheet,
  FormField,
  IconButton,
  Notice,
  SelectField,
  StatusBadge,
  Surface,
  Tabs,
  TextArea,
  TextField,
  type DataColumn,
} from "../ui";
import "../../ui-gallery.css";

const tabs = [
  { id: "PENDING", label: "Pending review" },
  { id: "COMPLETE", label: "Completed" },
] as const;
const selections = [
  { id: "OPEN", label: "Responses open" },
  { id: "LOCKED", label: "Responses locked", description: "Recipients can no longer edit responses." },
] as const;
type SampleRow = { id: string; request: string; status: string; owner: string };
const sampleRows: readonly SampleRow[] = [
  { id: "request-1", request: "Sample vendor address review", status: "Evidence needed", owner: "Sample Compliance Officer" },
  { id: "request-2", request: "Sample PCI-DSS certificate", status: "Completed", owner: "Sample Vendor Manager" },
];
const sampleColumns: readonly DataColumn<SampleRow>[] = [
  { id: "request", header: "Request", render: (row) => row.request, accessibleText: (row) => row.request },
  { id: "status", header: "Status", kind: "status", render: (row) => <StatusBadge tone={row.status === "Completed" ? "success" : "warning"}>{row.status}</StatusBadge>, accessibleText: (row) => row.status },
  { id: "owner", header: "Owner", render: (row) => row.owner, accessibleText: (row) => row.owner },
];

export function UIComponentGallery() {
  const [name, setName] = useState("Sample vendor");
  const [included, setIncluded] = useState(false);
  const [notes, setNotes] = useState("");
  const [selection, setSelection] = useState<"OPEN" | "LOCKED">("OPEN");
  const [tab, setTab] = useState<(typeof tabs)[number]["id"]>("PENDING");
  const [sheetOpen, setSheetOpen] = useState(false);

  return <main className="ui-gallery">
    <header className="ui-gallery__header">
      <p>Sample component data</p>
      <h1>ClearSight interface foundations</h1>
      <p>Review the supported component states at this viewport, theme and density before using them in a workflow.</p>
    </header>

    <GalleryGroup title="Actions">
      <Contract family="Button" job="Runs one named action." keyboard="Enter or Space activates it." prohibited="Do not style a native button in a feature.">
        <div className="ui-gallery__row"><Button variant="primary">Send sample form</Button><Button>Save sample draft</Button><Button variant="quiet">Cancel sample change</Button><Button variant="destructive">Revoke sample request</Button><Button isLoading>Saving sample draft</Button><Button isDisabled>Unavailable action</Button></div>
      </Contract>
      <Contract family="ActionLink" job="Navigates to another location." keyboard="Enter follows the link." prohibited="Do not use it for an in-place command."><ActionLink href="#sample-destination">Open sample destination</ActionLink></Contract>
      <Contract family="IconButton" job="Runs a compact, named icon action." keyboard="Enter or Space activates it." prohibited="Do not omit its accessible business name."><IconButton aria-label="Close sample panel"><span aria-hidden="true">×</span></IconButton></Contract>
    </GalleryGroup>

    <GalleryGroup title="Fields">
      <Contract family="FormField" job="Binds a label, guidance and validation to one control." keyboard="Focus moves to its supplied control." prohibited="Do not use placeholder text as the label.">
        <FormField label="Sample reference" description="Enter the reference used for this sample." isRequired>{(control) => <input {...control} value="SAMPLE-001" readOnly/>}</FormField>
      </Contract>
      <Contract family="TextField" job="Collects one short text value." keyboard="Tab focuses the field." prohibited="Do not use it for a bounded option list.">
        <div className="ui-gallery__grid"><TextField label="Sample vendor name" value={name} onChange={setName}/><TextField label="Sample response limit" type="number" value="5" min={1} max={10} step={1} onChange={() => undefined}/><TextField label="Sample invalid value" value="Unverified" onChange={() => undefined} errorMessage="Provide the verified sample value."/><TextField label="Sample read-only owner" value="Sample Compliance Officer" onChange={() => undefined} isReadOnly/><TextField label="Sample disabled field" value="Unavailable" onChange={() => undefined} isDisabled/></div>
      </Contract>
      <Contract family="TextArea" job="Collects a longer written response." keyboard="Tab focuses the field; line breaks remain available." prohibited="Do not use it for file evidence."><TextArea label="Sample review note" value={notes} onChange={setNotes} description="Describe what the sample reviewer confirmed."/></Contract>
    </GalleryGroup>

    <GalleryGroup title="Selection">
      <Contract family="CheckboxField" job="Includes or excludes one named option." keyboard="Space changes the selection after focus." prohibited="Do not use it for mutually exclusive choices."><CheckboxField label="Include sample evidence" description="Adds the sample evidence to this review only." isSelected={included} onChange={setIncluded}/></Contract>
      <Contract family="SelectField" job="Selects one value from a bounded list." keyboard="Arrow keys move; Enter selects; Escape closes." prohibited="Do not replace searchable remote results with this control."><SelectField label="Sample response status" value={selection} placeholder="Select sample status" options={selections} onChange={(value) => value && setSelection(value)}/></Contract>
    </GalleryGroup>

    <GalleryGroup title="Navigation">
      <Contract family="Tabs" job="Moves between peer views in one workspace." keyboard="Arrow keys move and activate; Home and End jump." prohibited="Do not add a second selected indicator."><Tabs ariaLabel="Sample request views" items={tabs} selectedKey={tab} onSelectionChange={setTab}>{(key) => <p>{key === "PENDING" ? "Sample requests awaiting review." : "Sample requests completed in this fixture."}</p>}</Tabs></Contract>
    </GalleryGroup>

    <GalleryGroup title="Feedback">
      <Contract family="StatusBadge" job="Names a concise stored or sample state." keyboard="No interaction." prohibited="Do not use color without a text label."><div className="ui-gallery__row"><StatusBadge tone="info">Sample review open</StatusBadge><StatusBadge tone="success">Sample completed</StatusBadge><StatusBadge tone="warning">Sample evidence needed</StatusBadge><StatusBadge tone="error">Sample request failed</StatusBadge><StatusBadge tone="unknown">Sample state unknown</StatusBadge></div></Contract>
      <Contract family="Notice" job="Explains a condition and recovery at the point of work." keyboard="Actions inside follow normal keyboard order." prohibited="Do not use it for decorative reassurance."><Notice tone="warning">The sample certificate has no verified expiry date. Review the document before approval.</Notice></Contract>
      <Contract family="EmptyState" job="Replaces a work region when its checked population is empty." keyboard="Its next action follows normal button behavior." prohibited="Do not leave an empty table scroll region behind."><EmptyState population="Sample sent forms matching this fixture" title="No sample sent forms match" description="Change the sample filters or create a sample distribution." action={<Button variant="primary">Send sample form</Button>}/></Contract>
    </GalleryGroup>

    <GalleryGroup title="Surfaces">
      <Contract family="Surface" job="Groups related work without implying a record." keyboard="No interaction." prohibited="Do not wrap every section in a surface."><Surface><p>Sample supporting information for the current task.</p></Surface></Contract>
      <Contract family="Card" job="Contains one coherent object or decision." keyboard="Interactive children keep their normal order." prohibited="Do not use it as decorative nesting."><Card><h3>Sample evidence request</h3><p>Owner: Sample Vendor Manager · Due: 12 September 2026</p></Card></Contract>
    </GalleryGroup>

    <GalleryGroup title="Data">
      <Contract family="FilterBar" job="Groups filters, result count and reset handling." keyboard="Tab follows the visible field order." prohibited="Do not compress fields below their usable width."><FilterBar label="Sample request filters" fields={<><TextField label="Sample owner" value="" onChange={() => undefined}/><SelectField label="Sample state" placeholder="All sample states" options={selections} onChange={() => undefined}/></>} resultCount={2} onClear={() => undefined}/></Contract>
      <Contract family="DataTable" job="Presents comparable populated records and page handling." keyboard="Tab reaches each focusable row and action." prohibited="Do not keep an empty horizontal scroll region."><DataTable ariaLabel="Sample requests" rows={sampleRows} rowKey={(row) => row.id} rowName={(row) => `${row.request}, ${row.status}, owned by ${row.owner}`} columns={sampleColumns} selectedKey="request-1" pagination={{ label: "Sample request pages", nextLabel: "Load next sample page", onNext: () => undefined }}/></Contract>
    </GalleryGroup>

    <GalleryGroup title="Overlays">
      <Contract family="FocusedSheet" job="Keeps one focused detail or decision above its source view." keyboard="Tab remains inside; Escape closes and restores focus." prohibited="Do not customize the backdrop in a feature."><Button onPress={() => setSheetOpen(true)}>Open sample sheet</Button></Contract>
    </GalleryGroup>
    {sheetOpen && <FocusedSheet label="Sample evidence detail" onClose={() => setSheetOpen(false)}><h2>Sample evidence detail</h2><p>This overlay contains labelled sample component data only.</p><Button onPress={() => setSheetOpen(false)}>Finish sample review</Button></FocusedSheet>}
  </main>;
}

function GalleryGroup({ title, children }: { title: string; children: ReactNode }) {
  return <section className="ui-gallery__group" aria-labelledby={`gallery-${title.toLowerCase()}`}><h2 id={`gallery-${title.toLowerCase()}`}>{title}</h2><div className="ui-gallery__contracts">{children}</div></section>;
}

function Contract({ family, job, keyboard, prohibited, children }: { family: string; job: string; keyboard: string; prohibited: string; children: ReactNode }) {
  return <article className="ui-gallery__contract" data-component-contract={family} aria-label={`Sample component data: ${family}`}>
    <header><h3>{family}</h3><p>{job}</p><p><strong>Keyboard:</strong> {keyboard}</p><p><strong>Do not substitute:</strong> {prohibited}</p></header>
    <div className="ui-gallery__sample">{children}</div>
  </article>;
}
