# Actor work count snapshot

Actor work counts cover the currently authorized workflow and document-proposal assignments only. PostgreSQL establishes a read-only repeatable-read snapshot before calculating both populations, takes the receipt timestamp from the database at that snapshot, and classifies deadlines against the same timestamp. Counts do not reuse the cursor page's update fence: excluding a row updated just before the snapshot would incorrectly remove an active assignment. The memory repository reads and timestamps while holding its shared lock.

The scope comes from the verified request actor. Counts do not imply completed work, coverage of other queue sources or enterprise-wide completeness. This snapshot is not a retained historical reconstruction.

The PostgreSQL 18 integration check uses a unique tenant in an isolated database. It commits a task update after request entry and before the source snapshot, then confirms the row is counted. It also commits completion after a repeatable-read snapshot and checks the prior snapshot retains the item while the next count excludes it. Existing authorized-source count and page parity checks remain separate.
