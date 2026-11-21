# Forms to Documents navigation

The Forms navigation links to the shared Documents workspace, replacing the retired embedded Forms document browser. The migrated FormsDocuments test exercises the real route and verifies that submitted documents are loaded in that workspace.

Synthetic browser captures show Forms and its Documents destination at 1440, 768 and 390 pixels. The check verifies the route, a sample submitted document and absence of horizontal overflow; the Forms navigation was visually inspected at all three widths. It uses fixture data and does not establish backend authorization, production data, dark-theme behavior or full-screen acceptance.
