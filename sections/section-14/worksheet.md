# Section 14: Driving the State Machine from the Operator Web Application

Time: 80 minutes.

In this section you'll build a Web Application — an HMI (human-machine interface) — to drive and observe everything you've built so far, starting with Section 13's state machine. Viam's SDKs make this quick: the page connects straight to your machine and gets the full resource API — every component and service method, plus the generic `DoCommand` verbs you've been writing — so you can run the hardware in one place and monitor and control it over the cloud from anywhere.

A Viam Web Application can be hosted two ways. You can run it **locally**, on a small web server next to the machine — the path this section takes, which keeps the frontend under your control and is the quickest way to iterate while developing. Or you can deploy it as a Viam **application** that Viam hosts for you and operators open from the **Apps** tab in the Viam app. Here we'll host locally, with a small Go web server.

## What you will do in this section

- Set up the operator Web Application on a local `viamkit/operatorapp` dev server
- Wire the four existing state-machine verbs (`start`, `restart`, `stop`, `status`) into the operator Web Application
- Render an SVG state diagram that updates as the cycle moves through states
- Extend the palletizer with operator control verbs (`step`, `goto`, `reset`) and matching Web Application controls
- Track cycle duration with rolling stats using viamkit's `cycle` package and display them in the Web Application

## How this section works

Sections 11–13 built the module by hand, so you know the patterns. Section 14 is where you guide the Coding Agent through the larger pieces — the SVG diagram, the polling code, the stats display — but **you write the prompts**. Each part teaches the concept and the interface first, then you write a prompt that asks for it, then you verify the result against a checklist. The skill here is specifying *what* you want precisely enough that the Coding Agent builds the right thing; the verification steps are how you catch when it didn't.

## Setup check

- You're working in **VS Code** with the `palletizing-module` project open.
- Section 13's `start`, `restart`, `stop`, and `status` work from the CONTROL tab, and you've verified the verb taxonomy (start = run from current; restart = reset + run; stop = cancel mid-flight).
- `viamkit` is pinned at `v0.16.0` in your `go.mod` (you added it in Section 13); the `operatorapp` package you'll use in Part 1 is available in that version.
- If your cell config looks broken, you can restore from [`sections/section-14/prereq-machine-config.json`](./prereq-machine-config.json) (substitute `<NAMESPACE>`); then re-run `viam module reload --part-id <PART_ID>`.

- Part 1 sets up the Web Application; after that, every time you change it, refresh `http://localhost:8080`.
- Every time you change the module, `viam module reload --part-id <PART_ID>` first.

**Cycle-state count (going in):** 3 working from §13 (`IDLE`, `MOVING_OUT`, `MOVING_BACK`) + 2 terminal (`DONE`, `ERROR`). §14 adds new verbs and a diagram but does not add new states; the cycle structure is unchanged.

## Part 1. Set up the Web Application (15 min)

In Section 12 you drove `move_to_pose` from the CONTROL-tab test card. The test card is a developer tool, though — to give an operator something friendlier, we'll build a single HTML page that uses the Viam JS SDK to connect to your palletizer and call its verbs, served by a small Go web server. To connect, the SDK needs your machine's address and credentials: the Go server reads them from environment variables and sets them as cookies the page can read, and the SDK uses the API key in those cookies to open an authenticated, encrypted connection to the machine.

`viamkit/operatorapp` serves your static frontend and sets the credential cookies the browser SDK reads, so the page authenticates to your cell. Your job is two files and three environment variables.

**1. Copy the web template.** You'll start from an existing template that already handles the connection and authentication and pulls in the Viam JS SDK, so you can focus on the controls rather than the plumbing. Copy it into a `static/` folder under a new `cmd/cli/`:

```bash
mkdir -p cmd/cli/static
cp ../viam-training/templates/operator-webapp.html cmd/cli/static/index.html
```

It goes under `cmd/cli/` because that's where the dev-server's `main.go` will live, and Go's `//go:embed` can only bundle files at or below the embedding file's own directory — so the page has to sit beside the program that serves it. (By Go convention, each runnable program gets its own `cmd/<name>/` directory; the module binary stays separate.)

The template provides a simple example: a connection-status line, a `cmd({verb: args})` helper that calls the service's `doCommand`, and a response panel. It discovers credentials from the cookies `operatorapp` sets.

**2. Add the dev-server entrypoint.** Copy the CLI main and example environment variable file:

```bash
cp ../viam-training/templates/cli-main.go cmd/cli/main.go
cp ../viam-training/templates/env.example .env
```

The `main.go` `//go:embed`s `static/` and calls `operatorapp.ListenAndServe(":8080", static)`. `operatorapp` is available in the `viamkit v0.16.0` you pinned in Section 13, so there's nothing to add to `go.mod` — `go mod tidy` resolves the new import path.

The `.env` goes at the **repo root**, not under `cmd/cli/`: `main.go` reads `.env` relative to the directory you run `go run ./cmd/cli` from, which is the repo root. A `.env` placed elsewhere is silently ignored, and the page connection stays red.

**3. Run it with your machine's credentials.** The server reads three environment variables from a local `.env` file and turns them into the cookies the page authenticates with. From the Viam app, get an API key (your machine → **Connect** → API keys → copy the key ID and key) and the main-part Fully Qualified Domain Name (FQDN): the "remote address" found in the machine details panel.

Update the `.env` file with your credential information:

```bash
VIAM_ROBOT_FQDN=<machine-main-part-fqdn>
VIAM_API_KEY_ID=<api-key-id>
VIAM_API_KEY=<api-key>
```

Run the CLI program:

```bash
go run ./cmd/cli
```

Open `http://localhost:8080` in a browser.

**Verify.**

- The connection-status line turns green; the source in parentheses reads `operatorapp-cookie`.
- Record what the status line shows: ___

If it stays red, confirm all three `VIAM_*` variables are in the `.env` file at the **repo root** (the directory you run `go run ./cmd/cli` from — not `cmd/cli/`), and that the FQDN is the machine's *main part* FQDN.

## Part 2. Wire the cycle-control row into the Web Application (15 min)

**Concept.** In Section 13 you added the `start`, `restart`, `stop`, and `status` verbs to the palletizer service and exercised them from the CONTROL tab — typing JSON into the DoCommand box by hand. Those verbs live on the service; the CONTROL tab was just one client calling them. In this part you give an operator a friendlier user experience through a dedicated interface of buttons.

Each button sends one verb. A **Start** button calls `cmd({start: true})`; the template's `cmd()` helper turns that into a `DoCommand` call to the palletizer service, invoking the same `start` verb — one click instead of hand-typed JSON. That call reaches the machine as a gRPC request over the SDK's connection, but the Viam TypeScript SDK handles all of that communication for you, so the code you write is just the `cmd()` call. **Restart** and **Stop** follow the same pattern.

To show what the cycle is doing, add a **status grid**: a timer polls `cmd({status: true})` and renders the `state` and `running` fields the verb returns, so the operator watches the cycle advance — `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE` — without opening the CONTROL tab.

Two design points worth getting right:

- **Poll fast enough to see the middle.** The cycle passes through `MOVING_OUT` and `MOVING_BACK` for ~4 seconds each. A 250 ms poll catches them; if the poll rate is too slow, the state could look like it went straight from `IDLE` to `DONE` and miss the intermediate states.
- **Visual hierarchy = mistake prevention.** Group `Start` and `Restart` together and style `Stop` in a danger color. An operator reaching for Stop in a hurry shouldn't have to read button labels.

**Writing the prompt.** Ask the Coding Agent to update `cmd/cli/static/index.html`. Your prompt should specify:

- What buttons you want, what you want them to look like, and what they should do. We want to wire these buttons to the verbs you created in the last section.

- How you want to visualize the status state and running variables. You mentioned a "status grid". Just let the Coding Agent know that you want to see it, and you can always style it afterwards.

Beyond that, this is *your* HMI — change the look and feel to match what you'd expect on a factory floor. Push the Coding Agent on colors, layout, button size, and font until it reads the way an operator panel should; there's no single right answer here.

Ask the Coding Agent to add comments as it makes changes.

**Verify.** After you have updated the code using the Coding Agent, refresh `http://localhost:8080` and view the changes. Can you drive a cycle from the Web Application?

- Click **Start**. Does the status grid show `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE`?
- Click **Stop** mid-cycle. Does the grid freeze on `MOVING_OUT` (or `MOVING_BACK`) with `running: false`?
- Click **Restart** from `DONE`. Does the cycle run again from `IDLE`?

**Review the Code:**

Let's take a look at the changes to the index.html. Ask the Coding Agent to explain what it changed and why.

- How did it wire the buttons to send the DoCommand verbs?
- Did the Coding Agent send commands through the `cmd()` helper rather than calling `doCommand` directly?
- How is the polling for the state machine status implemented?

## Part 3. Add the SVG state diagram (15 min)

**Concept.** The status grid from Part 2 reports the current state as text. A diagram adds what the grid can't: it places that state in the whole cycle, so an operator sees at a glance where the cycle is and what comes next. We've provided a diagram template in `templates/state-diagram.html` — two arrays and a function. Open it in VS Code and review the source; you should see:

- `STATE_LAYOUT` — one entry per state (`id` plus `x`/`y`/`w`/`h` coordinates).
- `STATE_EDGES` — one entry per transition (`from`/`to`).
- `setActiveState(id)` — removes `.active` from every node and adds it to the named one; the `.active` CSS class pulses (the template includes the `@keyframes pulse`).

`setActiveState` will need to be called from the status poll function you built in Part 2, so the active node lights up as the cycle advances.

**Writing the prompt.** Tell the Coding Agent:

- What you want — a state-machine diagram added to the Web Application, showing the cycle's states.
- What it should look like — use the provided `templates/state-diagram.html` as an example, but you can change the look and feel to match your HMI.
- How it should behave — the active node lights up and tracks the live state as the cycle runs.

Ask the Coding Agent to add comments as it makes changes.

**Some helpful notes if you get stuck:**
- Use `templates/state-diagram.html` as the template, and define which states to visualize (the five states from Section 13: `IDLE`, `MOVING_OUT`, `MOVING_BACK`, `DONE`, `ERROR`).
- Drive `setActiveState` from the status poll you built in Part 2 so the diagram follows the live `state`.

**Verify.** Refresh `http://localhost:8080` and view the diagram:

- Do you see the new diagram?
- Does it have all of the states?
- If you click **Start**, does the active node visibly move through `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE`?
- If you Stop mid-cycle, then Restart — does the diagram freeze and then update correctly?

**Review the Code:**

Ask the Coding Agent to walk you through the diagram code and explain what it changed and why.

- Where does the diagram read the current state from — is it driven by the same status poll, or something separate?

If you have any issues, ask your Coding Agent to troubleshoot. This is also a good time to adjust the look and feel — colors, fonts. Tweak it until you like it.

## Part 4. Extend the state machine with step / goto / reset (15 min)

**Concept.** We can now loop through the state machine, but sometimes we need to restart from a specific state, or test a single step. This is useful during development, when debugging an issue, or when an operator has to fix something and rerun the last state. It would be helpful if the service we are building could advance one transition, jump to a specific state, or clear back to `IDLE`. Each of these verbs maps to an existing state machine method:

- **`step`** → `machine.Step(ctx)`: run exactly one transition, then stop. Refuse (return an error) if a cycle is already running — `Step` and `Run` are mutually exclusive.
- **`goto`** → `machine.Goto(state)`: force the machine to a named state *without* running its handler. Takes the target state name as a JSON argument; refuse while running.
- **`reset`** → `machine.Reset()`: clear back to `IDLE` (and clear counters) without launching the run loop.

**The difference between `restart` and `reset`:** Sometimes we want to reset the state machine without starting up again; other times we want to run a new cycle. `restart` is `reset` + `Run`. If your `reset` also calls `Run`, you've accidentally just built a second `restart`. You may decide your automation workflow doesn't need so many verbs, but during development it's good to have them for testing thoroughly.

We also want to expand the Web Application at the same time. For Goto, we need a list of available states. Hardcoding that list in the JavaScript would let it drift out of sync with the service, so let the service provide it: add a small `{states: true}` verb that returns `machine.States()`, and the dropdown stays correct on its own.

**Writing the prompt.** Now that we have a working Web Application and a running module, we can prompt the Coding Agent to build the module and the Web Application changes together. Prompt the Coding Agent with the answers to the following questions:

- What you want on the module side: What are the three new state machine verbs and what does each one do? How would we get the available states for goto?

- What you want on the Web Application side: How would we want to control the new verbs?

Ask the Coding Agent to add comments as it makes changes.

**Some helpful notes if you get stuck:**
- Three new control verbs — `step`, `goto`, and `reset` (advance one transition; jump to a named state; clear back to `IDLE`).
- `step` should refuse if a cycle is already running; `goto` takes the target state as an argument and also refuses while running; `reset` clears to `IDLE` *without* launching the run loop.
- Populate the Goto dropdown from a small `{states: true}` verb that returns `machine.States()` rather than hardcoding the list in JS, so it stays correct on its own.

**Verify.** Rebuild and hot-reload the module (`viam module reload --part-id <PART_ID>`), refresh the Web Application, then drive each new control and watch the diagram:

- Clicking **Step** from `IDLE` advances the state machine to: ___ (expected: `MOVING_OUT`)
- The Goto dropdown lists which states? ___ (expected: the five state names from `machine.States()` — `IDLE`, `MOVING_OUT`, `MOVING_BACK`, `DONE`, `ERROR`)
- Clicking **Reset** from `DONE` (or any state) — does the active node return to `IDLE` on the diagram, and does `running` stay `false`?

**Review the Code:**

Ask the Coding Agent to walk you through the new verbs and explain what it changed and why.

- Does `reset` clear the machine *without* calling `Run` — i.e., is it genuinely different from `restart`?
- How do `step` and `goto` refuse when a cycle is already running?
- Where does the Goto dropdown get its list of states — the `{states: true}` verb, or a hardcoded array?

## Part 5. Add cycle stats (15 min)

**Concept.** Once the cycle runs end to end, the operator wants timing info: how long the last cycle took, the running average, and whether things are drifting slower or faster. We already have a `status` verb that reports what state the state machine is in — that's a good place to add cycle-time fields too. viamkit's `cycle` package gives you a **`Tracker`** for this: think of it as a stopwatch for repeated work, a *separate* object from the state machine. Construct one with `cycle.New(cycle.WithWindow(100))` — it returns a `*cycle.Tracker`; call it `tracker`. The window (100 here, a fine default) sets how many recent cycles the rolling stats cover: `Mean` and `P95` average the last 100, so older cycles age out and the numbers stay responsive to recent drift. `Count` is the exception — it counts every cycle.

You drive the timer by calling these methods on `tracker`:

- `tracker.Start()` — begin timing a cycle.
- `tracker.End()` — finish timing; records the cycle's duration.
- `tracker.Cancel()` — abandon the in-flight timing without recording it.
- `tracker.Stats()` — a snapshot: `Count`, `Last`, `Mean`, `P95` (the last 3 values are of type `time.Duration`; use `.Seconds()` to display).

**Watch out: the `Tracker` and the state `Machine` have lookalike methods.** They are different objects, and two names collide exactly — always qualify with the variable so it's clear which you mean:

| If you mean… | state machine | cycle timer |
| --- | --- | --- |
| reset it | `machine.Reset()` — back to `IDLE` | `tracker.Reset()` — clears the stats |
| is it active? | `machine.Running()` — a cycle is executing | `tracker.Running()` — a cycle is being timed |

And don't confuse `tracker.Start()` with the `start` verb or `machine.Run()`: the **machine runs** the cycle, the **tracker only times** it.

**Wiring the timer to the cycle.** Hook the tracker's Start / End / Cancel to the cycle's boundaries:

- **Start** at the cycle's beginning — when `start` launches the run.
- **End** on *entry* to `DONE` — register it through the machine's construction option `WithOnEntry(DONE, ...)`, calling `tracker.End()`. Use `WithOnEntry`, **not** `WithOnExit`: a terminal state's exit hook never fires, because the run loop exits the moment it reaches `DONE`.
- **Cancel** on stop — in your `stop` handler (the one that calls `life.Stop()`), also call `tracker.Cancel()`, so an interrupted cycle doesn't skew the average.

**Writing the prompt.** Add the cycle timer and its display together. Tell the Coding Agent:

- What you want on the module side: How should the module time each cycle using viamkit's `cycle` package, and where do the tracker's `Start`/`End`/`Cancel` calls belong? Which verb should report the stats?
- What you want on the Web Application side: Which timing numbers do you want to see, and where should they appear?

Ask the Coding Agent to add comments as it makes changes.

**Some helpful notes if you get stuck:**
- We want to track each cycle using viamkit's `cycle` package, and report the totals from `status`.
- We want to use the tracker's `Start`/`End`/`Cancel` to control the cycle timer: `End` on `OnEntry(DONE)`, and `Cancel` on stop so an interrupted cycle doesn't skew the average.
- Have `status` report total cycles, average duration, and p95, in seconds.

**Verify.** Rebuild and hot-reload. From the CONTROL tab, send `{"status": true}` to see the new fields:

- New cycle-stat fields that appeared (paste the field names): ___ (expected: the three you had `status` report — cycle count, average duration, and p95, in seconds — drawn from `tracker.Stats()`'s `Count`, `Mean`, and `P95`)

Then run cycles from the Web Application's Start button and watch the grid:

- What does the cycle count read after one completed cycle? After three?
- Does the average duration land around 8 seconds (the cycle has two 4-second moving states)?

**Review the Code:**

Ask the Coding Agent to explain how it wired the cycle tracker.

- Is `End` registered on `OnEntry(DONE)`? 
- Does a `stop` mid-cycle `Cancel` the tracker, so the interrupted cycle doesn't pollute the average?

## Done when

You can answer **yes** to all of these:

- The Web Application's six controls (Start, Restart, Stop, Step, Reset, Goto) are wired, and the **LOGS** tab in the Viam app (machine → **LOGS**) shows each verb landing when clicked.
- The status grid renders `state`, `running`, the cycle count, and (after the first cycle) the rolling stats.
- The Web Application shows a state-machine diagram that updates as the cycle moves through each state.
- A stop mid-cycle freezes the diagram at the correct spot, and a start resumes the cycle from there.

If you are having any issues, capture the symptoms and raise your hand.

## Takeaway

This section did two things on top of Section 13's state machine: it added new operator controls (`step`, `goto`, `reset`) and started tracking cycle duration. Each was a **paired change** — a new verb or stat on the module side and a matching button or grid field on the Web Application side.

That pairing is the pattern for the palletizer build: a change grows the state machine *and* the operator Web Application together. Now that both sides are structured, prompting the Coding Agent to consider them together lets you build the "Human Machine Interface" (HMI) and the module in parallel.
