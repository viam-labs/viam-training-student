# Section 13: State machine with viamkit/statemachine

Time: 60 minutes.

In Section 11 you scaffolded a module; in Section 12 you added a single DoCommand verb that moves the arm. Section 13 brings in state machines for multi-step work. Once a job is more than a single call, a state machine gives you named places (states) and clear transitions between them. We are going to start small — four states with sleep stubs, using the same shape: a method per state, a transition that returns the next state's name, terminal states for the end of the cycle.

## What you will do in this section

- Frame and build a state machine in a Viam module using `viamkit/statemachine`
- Drive the cycle and inspect its state through DoCommand verbs
- Stop a long-running cycle mid-flight cleanly via `viamkit/lifecycle` context cancellation

## Setup check

- You're working in **VS Code** with the `palletizing-module` open and the Go extension installed. The integrated terminal is where you'll run `go build` and `viam module reload`.
- Section 12's `move_to_pose` verb still works — sending `{"move_to_pose": {...}}` from the palletizer's CONTROL-tab DoCommand input moves the arm.
- `go build ./...` is clean from your module's root.
- Section 13 is your first use of `viamkit` — it isn't in `go.mod` yet; this section adds it as a dependency.
- If your cell config looks broken or you don't remember the state you left it in, you can restore from [`sections/section-13/prereq-machine-config.json`](./prereq-machine-config.json). Substitute `<NAMESPACE>` for your org's namespace. If the `palletizing-module` shows unavailable afterward, re-run `viam module reload --part-id <PART_ID>` to reinstall the hot-reloaded build.

**Cycle-state count (after this section):** 3 working — `IDLE`, `MOVING_OUT`, `MOVING_BACK` — plus 2 terminal (`DONE`, `ERROR`).

## Part 1. Building the state machine (30 min)

You'll build a four-state cycle — `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE` — wired through `viamkit/statemachine`, with `start`, `restart`, `stop`, and `status` added as new DoCommand verbs alongside Section 12's `move_to_pose`. The two moving states are 4-second sleep stubs for now; real motion will get added in a later section. The cycle code goes in a new file, `state_machine.go`, and you'll update `module.go` to wire the verbs in.

### What a state machine is

A state machine structures multi-step work as a set of named **states** and the **transitions** between them. At any moment the machine is in exactly one state. Each state has a **handler** — in Go, a method — that does that state's work and returns the next state to move to. A **run loop** calls the current state's method, moves to the state it returned, and repeats — stopping at a **terminal** state (the end of the cycle) or routing to an **error** state if a method fails. Developing this way keeps each step small and testable, makes "where is the cycle right now?" a single value (which the `status` verb and the Web Application read), and lets you stop, resume, or restart cleanly at state boundaries. This separation also supports incremental expansion and lets you add in exception paths, and gives you better visibility into where the module is and what it is going to do next.

### Why the cycle runs in a background goroutine

The state machine runs in a background goroutine, not inline in the DoCommand call. Here's why. A full pick-and-place cycle is many seconds of blocking work — each move doesn't return until the arm physically finishes. If `start` ran the cycle inline, the DoCommand would block for the whole cycle, and the operator couldn't ask for `status`, `stop`, or `restart` until the blocking call ended. By launching the cycle with `go machine.Run(...)` and returning immediately, `start` keeps the module responsive: the cycle advances in the background while the other verbs stay answerable. 

### 1.1 The state type and constants

Create `state_machine.go` in your module's root, in the same package as `module.go`. Notice that `cycleState` is a `string` under the hood, so the names you'll see in `status` (`"IDLE"`, `"MOVING_OUT"`) are the literal constant values. 

```go
package palletizingmodule

import (
    "context"
    "fmt"
    "time"

    "github.com/viam-labs/viamkit/statemachine"
)

// cycleState helps track the available states.
type cycleState string

const (
    stateIdle       cycleState = "IDLE"
    stateMovingOut  cycleState = "MOVING_OUT"
    stateMovingBack cycleState = "MOVING_BACK"
    stateDone       cycleState = "DONE"
    stateError      cycleState = "ERROR"
)
```

### The `viamkit/statemachine` interface

Let's take a look at the provided state machine interface:

- `statemachine.New(initial, opts...)` — builds a machine starting in the `initial` state.
- `WithHandlers(map[S]Handler[S])` — the state→handler table, where `Handler[S]` is `func(ctx) (S, error)`: do the state's work and return the next state.
- `WithTerminal(states...)` — states that end the run (they need no handler).
- `WithErrorState(s)` — the state a handler error routes to.
- `OnTransition(func(from, to S))` — a callback after every transition; we use it just to log.
- On the built machine: `Run(ctx)` drives the loop until a terminal/error state or context cancellation; `Current()`, `Running()`, and `LastError()` are thread-safe reads; `Reset()` returns to the initial state.

You provide the states and methods; viamkit provides the loop, the terminal/error handling, the cancellation plumbing, and the "am I running" bookkeeping. 

### 1.2 The state methods
First we are going to add a Context safe sleep helper method and the three state methods to `state_machine.go`:

```go
// sleepCtx waits for duration d, returning early with ctx.Err() if ctx is cancelled.
// The state methods use it so a stop cancels an in-flight stub promptly instead
// of after the full wait.
func sleepCtx(ctx context.Context, d time.Duration) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(d):
        return nil
    }
}

// stateIdle is the resting state. Running the machine from IDLE begins the
// cycle by transitioning straight to MOVING_OUT.
func (p *palletizer) stateIdle(ctx context.Context) (cycleState, error) {
    p.logger.Info("Entered IDLE")
    return stateMovingOut, nil
}

// stateMovingOut is a 4-second sleep stub. 
func (p *palletizer) stateMovingOut(ctx context.Context) (cycleState, error) {
    p.logger.Info("MOVING_OUT: sleep stub (4s)")
    if err := sleepCtx(ctx, 4*time.Second); err != nil {
        return stateMovingOut, err
    }
    return stateMovingBack, nil
}

// stateMovingBack is a 4-second sleep stub.
func (p *palletizer) stateMovingBack(ctx context.Context) (cycleState, error) {
    p.logger.Info("MOVING_BACK: sleep stub (4s)")
    if err := sleepCtx(ctx, 4*time.Second); err != nil {
        return stateMovingBack, err
    }
    return stateDone, nil
}
```

Three things to notice:

- The wait is `select { <-ctx.Done(); <-time.After(...) }`, **not** raw `time.Sleep`. A raw sleep ignores cancellation: a `stop` during the sleep wouldn't take effect until the 4 seconds elapsed, and by then the method would have already returned `stateMovingBack` and transitioned. The `select` lets `stop` interrupt mid-wait.
- On cancellation the method returns its **own** state plus `ctx.Err()` (which is `context.Canceled`). Returning the current state with the cancel error is what makes the machine stay parked on `MOVING_OUT` after a stop, rather than advancing — the machine special-cases `context.Canceled`.
- These are **methods on `*palletizer`**, not free functions. They do nothing with `p` today, but the signature is shaped so a state method can call `p.motion.Move(...)` when it needs to.

### Context and cancellation in Go

`stop` works through Go's `context.Context` — the standard way to signal "stop what you're doing" across function calls and goroutines. It's worth pausing on, because it's the mechanism behind every stop and resume in the palletizer:

- `context.WithCancel(parent)` returns a context plus a `cancel()` function. Calling `cancel()` closes the context's `Done()` channel.
- Long-running code watches that channel — it `select`s on `<-ctx.Done()` (cancelled → return now) against its real work. That's exactly what `sleepCtx` does above: it returns the moment the context is cancelled instead of always waiting the full 4 seconds.
- When code returns because it was cancelled, it returns `ctx.Err()`, which equals `context.Canceled`. You detect that with `errors.Is(err, context.Canceled)`.

So the chain for `stop` is: cancel the cycle's context → the in-flight state method's `sleepCtx` sees `Done()` and returns `context.Canceled` → the method returns that error → the machine stops cleanly, parked on its current state (it special-cases `context.Canceled` rather than routing to `ERROR`). Without this, a `stop` could only take effect at the *next* state boundary — after the full 4-second sleep — not mid-step. Make sure this is clear before moving on; it's the core of how the cycle is interruptible.

### The lifecycle loop-context

`viamkit/lifecycle` hands you one cancellable "loop context" that the cycle runs against. `start` launches `Run(life.EnsureLive())`; `stop` calls `life.Stop()`, which cancels that context and makes `Run` exit cleanly. `EnsureLive()` returns a live context — minting a fresh one if the current one was cancelled, or returning the existing one unchanged if it's still live. That fresh-on-demand behavior is what makes the *next* `start` work: after a `stop` the loop context is cancelled, and handing that dead context straight to `Run` (e.g. `Run(life.Ctx())`) would make it exit instantly, because `Run` treats a cancelled context as "we're done." So every `start` and `restart` calls `EnsureLive()` before relaunching `Run`. Skip it and start-after-stop silently does nothing: `status` keeps reporting the stopped state because `Run` exited within milliseconds.

### 1.3 Build the machine and the lifecycle

We have methods that make up the State Machine states, now let's build the state machine around them. Add a `newCycleMachine` helper to `state_machine.go`. `newCycleMachine` takes in a palletizer objects and returns a state machine object. We pass in a mapping between our constants and the handling methods. Then we let the state machine know which states are terminal, and add a print out when switching between states.

```go
// newCycleMachine builds the four-state cycle. OnTransition is a logging hook
// only — status reads the machine's own thread-safe accessors, so there's no
// shadow copy of the state to keep in sync.
func newCycleMachine(p *palletizer) *statemachine.Machine[cycleState] {
    return statemachine.New(stateIdle,
        statemachine.WithHandlers(map[cycleState]statemachine.Handler[cycleState]{
            stateIdle:       p.stateIdle,
            stateMovingOut:  p.stateMovingOut,
            stateMovingBack: p.stateMovingBack,
        }),
        statemachine.WithTerminal(stateDone, stateError),
        statemachine.WithErrorState(stateError),
        statemachine.OnTransition(func(from, to cycleState) {
            p.logger.Infof("cycle: %s -> %s", from, to)
        }),
    )
}
```
Some background information:

- In each map entry the key (`stateIdle`) is the `cycleState` constant; the value (`p.stateIdle`, no parentheses) is a *method value* — the `stateIdle` method bound to `p`, ready to call later. The constant and the method share a name but live in different namespaces, so this compiles.
- `OnTransition` fires after every transition on the transitioning goroutine. You shouldn't run any blocking calls within this routine.
- `WithErrorState(stateError)` routes *handler errors* to the `ERROR` state, but context cancellation is **not** considered a handler error — the machine special-cases `context.Canceled`, so a `stop` leaves you parked in the current state, not moved to `ERROR`. This allows you to call stop or pause, stop the context and stay in the same state. 
 `WithTerminal(stateDone, stateError)` makes both end states exit the lifecycle `Run` cleanly.

```go
// module.go
type palletizer struct {
    resource.AlwaysRebuild
    resource.Named

    name   resource.Name
    logger logging.Logger
    cfg    *Config
    motion motion.Service

    // added this section:
    machine  *statemachine.Machine[cycleState]
    life     *lifecycle.Lifecycle
    handlers map[string]doHandler // doHandler type defined alongside the dispatch refactor
}
```

Notice the struct no longer has the `cancelCtx`/`cancelFunc` pair the Section 12 scaffold gave you — the lifecycle owns the cancellable context now, so delete those two fields.

Next we instantiate the new variables and retire that old pair. In `NewPalletizer`:

- Remove the `cancelCtx, cancelFunc := context.WithCancel(context.Background())` line and the `cancelCtx: cancelCtx, cancelFunc: cancelFunc` entries from the `p := &palletizer{...}` literal. The early-return error path that called `cancelFunc()` can simply `return` now — the lifecycle replaces all of it.
- After the `p := &palletizer{...}` literal and before `return p, nil`, build the lifecycle and machine:

```go
    // Section 13: the lifecycle owns the cancellable context the cycle runs
    // against; the machine's state methods are on p, so build both after
    // p exists.
    p.life = lifecycle.New()
    p.machine = newCycleMachine(p)
```

Now that we are using the new viamkit library, we also need to add the import. Add `"github.com/viam-labs/viamkit/lifecycle"` to `module.go` (and `"github.com/viam-labs/viamkit/statemachine"` for the struct field type). 

And finally, we need to handle some bookkeeping when closing. Replace your existing `Close` so it calls the lifecycle's `Close` — that cancels the loop context for good, so a cycle still running at shutdown unwinds cleanly:

```go
func (p *palletizer) Close(context.Context) error {
    p.life.Close()
    return nil
}
```

### 1.4 The four DoCommand verb methods

We created a state machine, a few states, and lifecycle helper to keep things running smoothly. Now we need to add a way to control the state machine and matching verbs to DoCommand to expose that control to the outside world. Let's add the state machine control methods to `state_machine.go`. Each matches the `DoCommand` signature we worked with earlier (`ctx`, args-in, result-out, error):

```go
// start runs the cycle from its current state. After a stop this resumes
// mid-cycle; from a terminal state (DONE/ERROR) Run exits immediately, so it
// is a no-op until a restart resets the machine. No Reset here.
func (p *palletizer) handleStart(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    if p.machine.Running() {
        return map[string]interface{}{"accepted": false, "reason": "cycle already running"}, nil
    }
    runCtx := p.life.EnsureLive()
    go p.machine.Run(runCtx)
    return map[string]interface{}{"accepted": true}, nil
}

// stop cancels the in-flight cycle. The machine bails out of Run cleanly,
// leaving Current() on whatever state was executing.
func (p *palletizer) handleStop(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    p.life.Stop()
    return map[string]interface{}{"stopped": true}, nil
}

// restart resets the machine to IDLE and runs a fresh cycle. Unlike start,
// it Resets first.
func (p *palletizer) handleRestart(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    if err := p.machine.Reset(); err != nil {
        return nil, fmt.Errorf("restart: %w", err)
    }
    runCtx := p.life.EnsureLive()
    go p.machine.Run(runCtx)
    return map[string]interface{}{"restarted": true}, nil
}

// status reports the current state and whether a cycle is running. state
// answers "where in the cycle"; running answers "is it advancing".
func (p *palletizer) handleStatus(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    return map[string]interface{}{
        "state":   string(p.machine.Current()),
        "running": p.machine.Running(),
    }, nil
}
```

Three things to notice:

- **`start` never calls `Reset()`.** It does `EnsureLive()` then `go Run(...)` and returns immediately — that immediate return is the point: the cycle runs in the background while the operator polls `status`. Because there's no `Reset`, a `start` after a `stop` resumes from the stopped state.
- **`restart` = `Reset` + run.** `Reset()` returns the machine to `IDLE` and clears `LastError`; it returns an error if a cycle is in flight, so `restart` assumes you've stopped or the cycle has finished. It is the only verb that throws away progress. 
- **`status` reads the state directly** — `Current()` and `Running()` are thread-safe accessors. They report two independent facts: What state are we in and is it currently running?

### 1.5 Refactor the dispatch to a verb table

`DoCommand` is the **primary interface** for a generic service — it has no typed API of its own, so every capability the palletizer exposes (the `move_to_pose` verb from Section 12 and the four cycle-control verbs here) flows through `DoCommand`. As the verb count grows, `DoCommand` can get crowded and hard to manage. We can use a flat dispatch map to keep that single entry point manageable. In Section 12 your `DoCommand` dispatched a single verb with an `if`; with five verbs now, refactor it to a map. 

First, define the handler type and refactor `DoCommand` in `module.go`:

```go
type doHandler func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error)

func (p *palletizer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
    for verb, raw := range cmd {
        handler, ok := p.handlers[verb]
        if !ok {
            continue
        }
        args, _ := raw.(map[string]interface{})
        return handler(ctx, args)
    }
    return nil, fmt.Errorf("unknown command; supported: move_to_pose, start, stop, restart, status")
}
```

Then populate the table in `NewPalletizer`:

```go
    // Register the DoCommand verbs: move_to_pose 
    // plus the four cycle-control verbs.
    p.handlers = map[string]doHandler{
        "move_to_pose": p.moveToPose,
        "start":        p.handleStart,
        "stop":         p.handleStop,
        "restart":      p.handleRestart,
        "status":       p.handleStatus,
    }
```

Three things to notice:

- Adding a verb is now one map entry plus one method — the `DoCommand` body never changes again. New verbs grow the *vocabulary*, not the plumbing.
- The new cycle control verbs are flag-style (`{"start": true}`): the value is a bare `true`, not an args object.
- Commands carry a single verb, so the loop returns on the first known key. `move_to_pose` from Section 12 keeps working unchanged — it's just another entry in the table now.

### Build it

`state_machine.go` adds your first `viamkit` dependency. We should see a bit more now when we build:

```bash
go get github.com/viam-labs/viamkit@v0.16.0   # pin the toolkit version this course is built against
make setup                                     # runs go mod tidy — syncs go.mod / go.sum
                                               # (no setup target? run `go mod tidy` directly)
make
```

If everything went well we should get a clean build. If it fails, paste the *specific* compile error into Claude Code — it's good at pinpointing a missing import or a method-value typo. Then hot-reload onto your cell (`<PART_ID>` is the part ID from your machine's **CONFIGURE** tab, same as in Section 11):

```bash
viam module reload --part-id <PART_ID>
```

### Check your understanding

Before you drive the cycle, confirm you can answer these — write your answers down:

- Which of the five states have methods, and which are terminal? ________
- What does a state method return in order to advance the cycle? ________
- When `stop` fires during a 4-second wait, what makes the method return early instead of finishing the sleep? ________
- Why does `status` read `p.machine.Current()` / `Running()` instead of fields you maintain yourself? ________
- Why does `start` call `EnsureLive()` before `Run`, and what breaks if it doesn't? ________

If any answer is fuzzy, re-read the relevant subsection above before moving on.

## Part 2. Drive the state machine from the test card (25 min)

In the Viam app, open the **CONTROL** tab and find the `palletizer` service card. The DoCommand input is where you'll exercise the state machine for the rest of this section.

### Run a full cycle

Kick it off:

```json
{"start": true}
```

Click **Execute**. The cycle runs in the background. Watch progress by sending status repeatedly (just click Execute on the same payload — the response panel updates in place):

```json
{"status": true}
```

- Transitions observed (expected: `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE`, ~4 seconds per moving state):

- When the cycle reaches `DONE`, does `running` go to `false`?

A note on the two status fields, because you'll rely on the distinction in a moment: `status` reports `state` *and* `running`, and you can't derive one from the other. `state` is *where in the cycle the machine is* — `IDLE`, `MOVING_OUT`, `DONE`. `running` is *whether the cycle is actively advancing*. They usually move together (running while `MOVING_OUT`, not running at `DONE`), but a `stop` mid-cycle splits them: the machine parks on `MOVING_OUT` (`state`) with nothing advancing it (`running: false`). That stopped-but-not-finished condition is exactly what `start`-resume handles, and it's why both fields exist.

### Stop mid-flight

Kick off a fresh cycle, then within the first four seconds (while in `MOVING_OUT`) send stop:

```json
{"stop": true}
```

Then send status:

```json
{"status": true}
```

- `state` after stop:

- `running` after stop (should be `false`):

Start again — per the verb taxonomy, `start` means "run from current state", so it should pick up from wherever `stop` left it, not from `IDLE`:

```json
{"start": true}
```

- After start: did the cycle resume from `MOVING_OUT`, or restart from `IDLE`?

**If start-after-stop does nothing**

Recall your `handleStart` from Part 1: it calls `p.life.EnsureLive()` *before* `go p.machine.Run(...)`. That `EnsureLive()` is essential. When `stop` ran, it cancelled the loop context; if `start` relaunches `Run` against that still-cancelled context (e.g. `go p.machine.Run(p.life.Ctx())` without the `EnsureLive()` first), `Run` sees a dead context and exits within milliseconds — `status` keeps reporting the stopped state and no transitions happen. If you see that symptom, check that `handleStart` calls `EnsureLive()` and runs against the context it returns. If all else fails, remember claude can be invaluable in tracking down these errors.

### Confirm the verb taxonomy

Run through these five cases. Predict the expected behavior before each `status` check, then verify against the response:

1. Send `{"start": true}`. Wait for the cycle to complete; status reports `DONE`.
2. Send `{"start": true}` again. Expected: **stays `DONE`** — `start` runs from the current state, and `DONE` is terminal, so `Run` exits immediately without advancing. (Only `restart` gets you out of a terminal state.)
3. Send `{"restart": true}`. Expected: `DONE` → `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE` (a fresh cycle).
4. Send `{"start": true}`, then `{"stop": true}` mid-cycle, then `{"start": true}` again. Expected: resumes from where stop landed (NOT reset).
5. Send `{"start": true}`, then `{"stop": true}` mid-cycle, then `{"restart": true}`. Expected: `IDLE` → fresh cycle.

If any behavior differs from your prediction, dig into `state_machine.go` and fix — then describe the symptom to Claude if the cause isn't obvious.

## Done when

You can answer **yes** to all of these:

- `make` is clean after adding `state_machine.go` and refactoring the dispatch.
- From the CONTROL tab, `{"start": true}` returns immediately, and polling `{"status": true}` shows the cycle advance `IDLE` → `MOVING_OUT` → `MOVING_BACK` → `DONE` (~8 seconds total).
- `{"stop": true}` during `MOVING_OUT` halts the cycle on `MOVING_OUT`; `status` shows `state: MOVING_OUT`, `running: false`.
- After that stop, `{"start": true}` resumes from `MOVING_OUT` (not `IDLE`), and `{"restart": true}` resets to `IDLE` and runs fresh — confirmed across the five-case taxonomy.
- Your `status` response includes both `state` and `running`, and you can explain why one isn't derivable from the other.

## Takeaway

This section gives you two patterns you'll lean on for the rest of the palletizer build:

1. **State machines for multi-step work.** Once a job is more than a single call, a state machine gives you named places (states) and clear transitions between them. You built a four-state machine with sleep stubs — each state has a method that returns the next state's name, and terminal states end the run. Growing the machine adds states the same way; the constructor and the verb wiring don't change.

2. **DoCommand is the control surface, driven by async verbs.** A generic service has no typed API, so `DoCommand` is *the* interface — every verb the operator adds lives there, making it the primary path for the palletizer's control logic. `start`, `restart`, `stop`, and `status` are a small but complete vocabulary for driving and observing a long-running operation from outside the module: `start` runs from the current state, `restart` resets and runs, `stop` cancels mid-flight, `status` reports state and running. The same four work whether the cycle takes 8 seconds with sleep stubs or longer with real motion.
