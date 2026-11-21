# Work queue row alignment

The approved Work source keeps compact queue rows at 768px, while the shared
table's container breakpoint converts them to tall labelled cards. Use the
existing viewport-responsive table contract for the Work queue: keep table
rows above 700px and its labelled card replacement at phone widths. Retain the
visible Inspect action, selection, source filters and authorized record readers.
Long content must remain readable and recoverable if a table needs horizontal
scrolling; no data or authority contract changes.

Compare the pinned Work HTML and runtime at 1440, 1280, 1024, 768 and 390px,
including both runtime themes, keyboard/touch inspection, long content and
partial or denied data. This is partial V2 integration, not screen acceptance.
