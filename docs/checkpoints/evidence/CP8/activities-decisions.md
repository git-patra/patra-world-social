# Activities execution decisions

These decisions are copied from the execution ledger in chronological order. They do not change the approved gameplay or deferred acceptance gates.

1. **Reuse the existing feat/world-activities checkout.** This preserves the configured toolchain, database and local preview. Cost if wrong: switching branches affects that preview; a separate worktree would have isolated it.
2. **Use separate implementation and independent review agents.** The cross-layer plan benefits from fresh review context. Cost: additional review time and coordination.
3. **Develop Task4 frontend against the agreed Task3 interface while authority work proceeds.** Both were integrated and reviewed before dependent games. Cost if the interface changed: coordinated updates and integration rework.
4. **Treat fishing coordinates as visual anchors.** Spot3 remains at53,33; the reachable standing tile54,33 is within the approved1.25tile interaction radius. Cost if wrong: visual standing alignment would need adjustment. The approved map remains unchanged.
5. **Use procedural arena art and timestamp-derived hit flashes.** These satisfy the existing-canvas and bounded-rendering constraints without an extra raster bank or accumulating particle pool. Cost if insufficient: visual polish rework; authority and layout are unchanged.

The ignored execution workspace is retained for continuity and raw local logs; portable reports and these decisions are committed. No working tree reset or source-repository edits were used.
