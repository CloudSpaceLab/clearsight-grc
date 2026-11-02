# Rejected and superseded People/access evidence

These findings were retained during direct source/runtime and independent visual review. An automated PASS did not establish visual acceptance. The [final evidence index](../README.md) records the corrected runtime and the separate limits on operational acceptance.

| Checkpoint | Observed defect | Retained diagnosis | Correction |
| --- | --- | --- | --- |
| `712cd431` | Tablet search kept a 420px cap; heading artwork overlapped the tabs | [Rejected checkpoint](rejected-712cd431.json), [image index](images.json) | Configure the public search-width token and contain the wave within the heading; add measured assertions |
| `75cb676d` | Loaded summary lacked an appropriate labelled role; the Delegations metric label fragmented at 390px | [Superseded checkpoint](superseded-75cb676d.json) | Add the group role and use Records for the compact metric label |
| `d2cbf333` | Organization cold entry showed both unstyled navigation and its compact selector; unavailable governance labels fragmented on phones | [Independent findings](d2cbf333-superseded.json) | Load the switcher in the shared component layer and stack uncertain metric label/quality through the existing geometry token |
| `d2cbf333` | Policy Return action had valid geometry but intermittently painted blank immediately after scrolling | [Negative controls](negative-control-verification.json) | Wait for settled paint, require consecutive matching frames and inspect pixels within the action text bounds |
| Initial Organization phone correction | Full-width filters reduced the title to zero width at 390px; the same collision and split table actions persisted at 600px | [Heading diagnosis](organization-phone-heading.json), [390px raw geometry](organization-phone-position-heading-rejected.json), [600px raw geometry](organization-600-position-header-rejected.json) | Stack owning card headers at 900px and reflow identity tables into labelled records at 680px; retain cold-entry and scrolled supplement checks |

## Directly inspected diagnostic images

| Finding | Rejected image | Same-build or final comparison |
| --- | --- | --- |
| Secondary navigation | [Unstyled desktop entry](d2cb-secondary-navigation-desktop.png) | [Final desktop entry](../after/configuration/configuration-organization-desktop-light-1440x900.png) |
| Unavailable metrics | [Fragmented phone labels](d2cb-unavailable-summary-phone.png) | [Final restricted view](../after/280-people-access-routes-denied-dark-390.png) |
| Return paint race | [Blank action](d2cb-policy-action-paint-race.png) | [Same-build settled action](d2cb-policy-action-settled.png), [final continuation](../after/280-people-access-routes-definition-dark-390-continuation.png) |
| Organization heading | [390px collision](organization-phone-position-heading-rejected.png), [600px collision](organization-600-position-header-rejected.png) | [Final 390px heading and filters](../after/configuration/configuration-organization-mobile-dark-390x844-position-header.png) |

The negative-control receipt retains original scratch filenames for reproducibility. Its rejected `final-d2cbf333/280-people-access-routes-definition-dark-390-continuation.png` corresponds byte-for-byte to `d2cb-policy-action-paint-race.png` here; `policy-paint-settled.png` corresponds to `d2cb-policy-action-settled.png`. The exact rejected text sample has zero contrasting pixels; the settled same-build sample has 2,526. This measures painted content, not text contrast conformance or live authority.

The rejected captures and prior manifests also remain in Git history. The original HTML design artifacts remain immutable. Whole-screen acceptance stays 0/19.
