# ClearSight — 19 / Contextual AI Copilot

## Design contract

- The Copilot is a **dismissible contextual surface** within an existing Program or assessment, not another dashboard or independent governance engine.
- Context derives from the active authorized Program, framework edition, tenant, legal entity, organization group and a coherent source/revision boundary. The mock values are explicitly illustrative.
- One response at a time, three suggestion options maximum. One suggestion is visible; arrows manually select the next. No cycling or continuous animation.
- Evidence and framework views are distinct. Source links open a review dialog retaining ownership, status, source edition and review limitations.
- Proposed updates open a permission-aware **read-only proposal preview**. The real ClearSight runtime must perform a fresh authorization and version check, then require the appropriate governed review/approval; AI output cannot grant authority, mark evidence accepted or conclude legal compliance.
- The blue orb is small, still, and separated from all labels/actions. It can indicate assistance availability without advertising autonomous authority. No idle animation.
- Missing/unknown data must reduce confidence visibly. If the model is unavailable, the rest of the Program remains usable and the assistant input states its limitation rather than fabricating an answer.
- Desktop: Program content + a 425px side panel. Tablet: the assistant moves below the program overview. Mobile: a compact top bar, the same primary content and an open/close assistant control. Preserve keyboard operation, focus and reduced-motion.

## Interactions in the HTML prototype

- Switch the Program section: Overview, Controls, Evidence, Assessments.
- Switch the Copilot mode: Ask, Evidence, Mapping.
- Rotate exactly three suggestions with previous/next buttons.
- Open a source by clicking its reference chip or a record.
- Review a draft proposal by selecting “Prepare assessment summary.”
- Close/reopen Copilot with the header action; use Escape to close a modal.
- Submit a sample question: an offline notice is shown, no fabricated response.
- Global search control and Ctrl/Command+K focus the Program question input.
- Change the sample group scope for visual demonstration only (does not make a backend query).

## Implementation and validation

- Use existing ClearSight Program/Forms/Evidence/Framework catalog/Work/AI Gateway foundations.
- Do not introduce a new scoring engine, AI-specific source database, independent action engine or duplicate browser-side permissions.
- Before real wiring, bind source citations to exact authorized references and their snapshots; make action proposals require explicit review, preview, and routed approval.
- The local browser prototype was checked at 390, 620, 768, 1024, 1280 and 1880 CSS px for layout overflow, JS errors, keyboard dismissal, mode switching, source details, preview and reduced-motion. Screenshots are illustrative; no backend calls, uploads or actual AI analysis were performed.
