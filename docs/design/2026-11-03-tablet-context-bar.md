# Shared tablet context bar

The approved Framework and Program references keep the compact navigation rail
and a flat, full-width context bar at 1024px. The runtime's desktop context
rules stopped at 1025px and its compact rules started at 820px, leaving a
floating legacy bar and extra page spacing between those breakpoints. Extend
the existing flat context-bar arrangement through 821px while retaining the
verified identity, role, scope and navigation controls. No route, permission or
scope logic changes.

Compare Frameworks and Programs with their pinned HTML at 1440, 1280, 1024,
768 and 390px, and exercise the transition boundaries and other shared-shell
destinations. Verify keyboard, touch, denied Settings, role/scope controls,
long labels and no horizontal overflow. This is partial V2 integration.
