# Viam Training — Student Worksheets

A hands-on, in-person course that teaches the [Viam](https://www.viam.com/) platform
end-to-end by building a **palletizing application**. You start in a fully simulated
work cell, then retarget the *same* module to physical hardware by configuration alone.

Each section below is a self-contained worksheet you work through at your own pace.
Foundations (§2–§7) build the mental model and core skills; the palletizer arc
(§10–§20) builds one Go module section by section — a state machine that drives a
simulated arm and vacuum gripper through a full pick-and-place cycle, plus an operator
web app to run it.

> Sections 1, 8, and 9 are delivered as instructor-led sessions and have no worksheet here.

## Getting set up

Clone this repo **next to** (a sibling of) the module you build during the course — not
inside it. The palletizer worksheets copy templates with relative paths like
`../viam-training/templates/...`, so clone it into a folder named `viam-training`:

```bash
git clone git@github.com:viam-labs/viam-training-student.git viam-training
```

That gives you this layout, which the worksheets assume:

```
your-work-dir/
├── viam-training/          # this repo — worksheets, prereq configs, templates
└── palletizing-module/     # the module you build (created during the palletizer arc)
    └── cmd/cli/            # e.g. §14 runs `cp ../viam-training/templates/… .` from here
```

## Worksheets

### Foundations

| Section | Worksheet |
|---|---|
| 2 | [Set up a Viam machine](sections/section-2/worksheet.md) |
| 3 | [Configure components](sections/section-3/worksheet.md) |
| 4 | [SDKs](sections/section-4/worksheet.md) |
| 5 | [The Viam CLI](sections/section-5/worksheet.md) |
| 6 | [Modules](sections/section-6/worksheet.md) |
| 7 | [Frame system, orientation, and end-effector frames](sections/section-7/worksheet.md) |

### Palletizer build

| Section | Worksheet |
|---|---|
| 10 | [Building the virtual work cell](sections/section-10/worksheet.md) |
| 11 | [Framing the palletizer module](sections/section-11/worksheet.md) |
| 12 | [First DoCommand verb](sections/section-12/worksheet.md) |
| 13 | [State machine with `viamkit/statemachine`](sections/section-13/worksheet.md) |
| 14 | [Driving the state machine from the operator web app](sections/section-14/worksheet.md) |
| 15 | [Moving with motion constraints](sections/section-15/worksheet.md) |
| 16 | [Pick-station component and pickup states](sections/section-16/worksheet.md) |
| 17 | [Pallet, place states, and the full motion cycle](sections/section-17/worksheet.md) |
| 18 | [Vacuum gripper coordination](sections/section-18/worksheet.md) |
| 19 | [Pack-sequencer service](sections/section-19/worksheet.md) |
| 20 | [Verify pattern + diagnostic loop](sections/section-20/worksheet.md) |

## How each section works

- **`worksheet.md`** — the section's instructions. Read it top to bottom and do the steps.
- **`prereq-machine-config.json`** (palletizer sections) — the machine config the section
  starts from. Apply it to your machine's **CONFIGURE** tab before you begin so your cell
  matches what the worksheet expects.

## Templates

Reference assets you drop into your work and extend instead of inventing from scratch.
See [`templates/README.md`](templates/README.md) for what each one is and which section
uses it.

- [`operator-webapp.html`](templates/operator-webapp.html) — starter operator web app (§12).
- [`state-diagram.html`](templates/state-diagram.html) — SVG state-diagram renderer (§14+).
- [`cli-main.go`](templates/cli-main.go) — local dev server that serves the web app with
  your machine credentials.
- [`env.example`](templates/env.example) — copy to `.env` and fill in your machine's
  credentials for the dev server.

## Reference modules

The palletizer worksheets align with these production Go modules — read them alongside
writing your own:

- [`viam-labs/Palletizing-Module`](https://github.com/viam-labs/Palletizing-Module) — the palletizer module the curriculum mirrors.
- [`viam-labs/robotiq-epick`](https://github.com/viam-labs/robotiq-epick) — vacuum gripper module.
