# Personal email preferences

Every signed-in user can open email preferences from the account control. The existing identity-scoped API supplies and saves digest, time zone and quiet-hour settings using the stored expected version. Critical email remains required. A denied read clears the form; conflicts and uncertain save results require a fresh read before another save. A successful receipt must match the tenant, person, values and next version.

The migrated component tests cover critical email, digest and quiet-hour times, scope mismatch, conflicts, unconfirmed results, access loss, retries and owner changes. The retained 1440, 768 and 390 pixel captures are isolated mocked-component evidence, not full-shell or hosted API proof; the desktop/tablet harness retains an empty shell sidebar offset.
