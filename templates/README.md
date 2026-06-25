# Templates

Reference assets students drop into their work and extend, instead of inventing each one from scratch. Each template lands in a specific palletizer-arc section.

| File | Section | What it is |
|---|---|---|
| [`operator-webapp.html`](operator-webapp.html) | §12 | Starter webapp — cookie-based credential discovery, a connected `cmd({verb: args})` helper that wraps `Struct.fromJson` correctly, a connection-status line, a Reconnect button, and a results panel. The pieces students would otherwise discover from external repos. |
| [`state-diagram.html`](state-diagram.html) | §14 (and onward) | Self-contained SVG state-diagram renderer driven by `STATE_LAYOUT` + `STATE_EDGES` arrays. Open in a browser to see how it looks, then copy the CSS + JS + SVG markup into the operator webapp. Adding a state in a later section is one new entry per state — the renderer never changes. |

## Why templates and not "let students invent"

Two places where every student would otherwise have to discover the same friction independently:

1. **The cookie-auth shape for operator webapps** — undocumented in the curriculum prose; without the starter, students rediscover it by reading other webapp modules.
2. **The state-diagram renderer** — every student would hand-roll a different layout. A shared template keeps every student's diagram the same shape, which keeps instructor review tractable as the state machine grows.

## How to update them

When the templates need to evolve:

- Update the relevant template file.
- Update the row in this README if the file's purpose or section changed.
- The worksheets reference each template by path; those references shouldn't need to change unless you rename or move a file.
