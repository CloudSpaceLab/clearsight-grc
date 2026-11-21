# Program register continuation recovery

The bounded Program register retains loaded records when a continuation request
fails, but did not tell the user that the next page failed. Show a concise error
and retry action alongside the existing continuation control. Retry the same
cursor and keep the current filters and loaded records; a new query clears the
old error. No Program, catalog or authority semantics change.
