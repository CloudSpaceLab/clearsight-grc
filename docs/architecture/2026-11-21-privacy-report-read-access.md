# Interim privacy and reporting read authorization

Privacy activity reads and material commands require the verified identity's global `PRIVACY_READ` permission. Neither oversight nor department-only grants imply this permission, and no role receives it automatically. Governed role configuration and appropriately scoped privacy access remain outstanding.

Report definitions, definition commands, revisions, owner options, runs and run history are temporarily restricted to global `PRIVACY_READ`, including non-privacy reports. The existing report repository cannot yet enforce dataset-level authorization for mixed metadata queries. This intentionally limits availability until bounded dataset-aware queries and an approved authorization policy are implemented.

Generating or downloading reports for processing activities or processing-activity exceptions requires `PRIVACY_READ` in addition to existing command or download authorization. The permission is checked again when replaying an idempotent run request and when opening an artifact; requester status or `REPORT_DOWNLOAD` alone cannot bypass it. Revoked permissions therefore prevent both operations. Verified current identity and entity scope remain required.

These restrictions do not establish complete privacy policy, delegation, actor-scoped access, or production readiness. Reports and privacy workspaces remain unavailable until approved policy, grants and end-to-end evidence are in place.
