# Architecture diagrams

Self-contained HTML diagrams generated with [Archify](https://github.com/Cocoon-AI/architecture-diagram-generator) (dark/light theme + export to PNG/SVG).

Open any `.html` file in a browser.

| Diagram | File | Type |
|---------|------|------|
| System architecture | [go-imapsync-architecture.html](./go-imapsync-architecture.html) | components / packages / IMAP hosts |
| Sync run workflow | [sync-run-workflow.html](./sync-run-workflow.html) | CLI → dial → folders → messages → summary |
| Message transfer sequence | [message-transfer-sequence.html](./message-transfer-sequence.html) | per-folder SELECT / FETCH / APPEND |
| Message data path | [message-path-dataflow.html](./message-path-dataflow.html) | identity keys and stateless transfer |

Source JSON (for re-render):

```bash
# From a machine with Archify installed:
ARCHIFY=~/.agents/skills/archify/bin/archify.mjs
node "$ARCHIFY" render architecture src/go-imapsync.architecture.json go-imapsync-architecture.html
node "$ARCHIFY" render workflow src/sync-run.workflow.json sync-run-workflow.html
node "$ARCHIFY" render sequence src/message-transfer.sequence.json message-transfer-sequence.html
node "$ARCHIFY" render dataflow src/message-path.dataflow.json message-path-dataflow.html
```

All project documentation and source comments for go-imapsync are written in **English**.
