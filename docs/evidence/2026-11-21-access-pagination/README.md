# Pagination access failure

When a later page returns HTTP 401 or 403, the Forms, Programs, risk and loss lists discard the previously displayed records and offer a fresh read. Other failures retain the current page and allow retry of the same cursor.

On 21 November 2026, Codex `/root` used synthetic API responses to render the 403 state for each list at 1440, 768 and 390 pixels. Each run removed the sample record, kept the retry action visible and had no horizontal overflow. The screenshots in this directory show those states; the 390-pixel loss view was visually inspected. The expected denied requests produced browser console errors. These checks do not establish production authorization or full screen acceptance.
