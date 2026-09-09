import { useState } from "react";
import { FormProposalReview } from "./components/forms/FormProposalReview";
import { Notice } from "./components/ui";
import { findingFollowUpFixture, findingSourceElements } from "./findingFollowUpFixture";
import "./forms-foundation.css";

export function FindingFollowUpEvidence() {
  const [proposal, setProposal] = useState(() => findingFollowUpFixture());
  return <main style={{ maxWidth: 1400, margin: "0 auto", padding: 24 }}>
    <Notice tone="info">Sample data · Historical vendor findings. No requests are sent from this review.</Notice>
    <FormProposalReview proposal={proposal} sourceTitle="Sample findings register.xlsx" sourceElements={findingSourceElements} onProposalChange={setProposal}/>
  </main>;
}
