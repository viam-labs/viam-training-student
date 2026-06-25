# Section 18: Vacuum gripper coordination

Time: 55 minutes.

Section 17 ended with the arm running the full pick-and-place motion path without
engaging the gripper. This section adds gripper coordination: engage suction at the
pickup, verify the seal formed before lifting the box away, and release suction at the
placement. We will be using the simulated gripper, but we need to think about how the
real gripper might work in this scenario.

### What the gripper does

The gripper on a vacuum pick-and-place cell does three things:

1. **Engage suction** to grasp the box.
2. **Release suction** to drop the box.
3. **Report whether the seal is good** so the cycle can catch a missed grasp or a
   mid-transit drop before running into trouble.

### The gripper we are using

The gripper in the cell is a `simulated-epick-vacuum-gripper` (configured back in
Section 10). It is a software model of the Robotiq EPick vacuum gripper — same API as a
physical EPick. After suction is engaged it takes a short moment to report a sealed
grasp, modelling the time a vacuum cup takes to pull a seal.

### The three methods we will call

The gripper exposes three methods relevant to this section. Each takes a trailing
`extra map[string]interface{}` — an rdk convention for implementation-specific
pass-through you will see on many component methods; we pass `nil`:

- `Grab(ctx, extra)` — engages vacuum suction. Returns when the command is acknowledged;
  the seal forms shortly after.
- `IsHoldingSomething(ctx, extra)` — returns a status reporting whether the gripper
  currently has a sealed grasp.
- `Open(ctx, extra)` — disengages vacuum. The seal breaks shortly after.

### How they fit together in this section

Once we call `Grab`, the seal does not form instantly. In real life the vacuum pump
takes time to pull a good seal, so we have to monitor `IsHoldingSomething` after the
call to confirm we actually have the box — otherwise we might lift a box we don't have
a good grasp on. That's the work for two new states: `ENABLING_VACUUM`, which will call
`Grab`, and `CHECKING_SUCTION`, which polls `IsHoldingSomething` until it reports
holding (or the poll times out). Then when we go to place the box down, the existing
`RELEASING_AT_PALLET` state finally gets its `gripper.Open` call, followed by the
existing dwell so the seal has time to release before the `LIFTING` state runs.

Two new states are added to the pickup half:

- **`ENABLING_VACUUM`** — runs between `MOVING_TO_PICKUP_HOME` and `GRASPING`. Calls
  `gripper.Grab`.
- **`CHECKING_SUCTION`** — runs between `GRASPING` and `RETRACTING_NORMAL`. Polls
  `IsHoldingSomething` until the seal is confirmed or a timeout fires.

One existing state gets its body filled in:

- **`RELEASING_AT_PALLET`** from Section 17 was a dwell-only stub. This section adds a
  `gripper.Open` call at entry, followed by the existing `place_dwell_secs` dwell so the
  seal has time to release before `LIFTING` runs.

The cycle's overall structure is unchanged; only the two new states and the upgraded
state's body are different.

## What you will do in this section

- Confirm the gripper behaves as expected from its component card
- Resolve the typed `gripper.Gripper` handle out of `deps` and store it on the struct
- Add `ENABLING_VACUUM` to engage suction before the descent
- Add `CHECKING_SUCTION` plus the `waitForSeal` helper to verify the seal formed
- Wire `gripper.Open` into the existing `stateReleasingAtPallet` method

## How this section works

Each Part teaches a concept, you write the prompt yourself, you review the generated code
against a checklist, and then you verify behavior. Parts that are small enough may skip the
prompt and review and deliver a code snippet directly.

## Setup check

- Section 17 runs end-to-end: the full motion cycle traces all 11 working states; the
  arm dwells at the center of the pallet top before returning.
- The palletizer has the following Config attrs: `arm`, `gripper`, `pick_station`,
  `pallet`, `box_width_mm`, `box_length_mm`, `box_height_mm`, `safety_height_mm`,
  `place_dwell_secs`. Resource graph green.
- `viam module reload --part-id <PART_ID>` reinstalls cleanly.

If the cell drifted, restore `prereq-machine-config.json` and reload.

**Cycle-state count (going in):** 11 working from §17 + 2 terminal. §18 inserts two new states on the pickup half (`ENABLING_VACUUM` between `MOVING_TO_PICKUP_HOME` and `GRASPING`; `CHECKING_SUCTION` between `GRASPING` and `RETRACTING_NORMAL`) — after this section the count is **13 working** + 2 terminal.

---

## Part 1. Verify the gripper and explore its API (5 min)

### Concept (covered in the section intro)

Before we write any coordination code, a quick hands-on warmup with the three gripper
methods from the gripper's component card.

### Explore the gripper API by hand — no prompt

In the Viam App, open the gripper's component card. Use its `Grab` and `Open` controls
and watch the `IsHoldingSomething` status:

- Before any `Grab`: not-holding.
- Immediately after `Grab`: still not-holding (the seal has not formed yet).
- A moment after `Grab` (once the seal forms, ~1 s): holding.
- After `Open`: not-holding again.

### Verify

- [ ] `Grab` and `Open` drive cleanly from the card.
- [ ] `IsHoldingSomething` reads not-holding before `Grab`, holding shortly after `Grab`,
      and not-holding after `Open`.
- [ ] You can name the three methods and describe what each returns.

---

## Part 2. Resolve the gripper handle, then add ENABLING_VACUUM (15 min)

### Resolve the gripper handle

Up to now the palletizer has only ever referred to the gripper by *name* — the
`cfg.Gripper` string you added in Section 11, passed as the `ComponentName` to
`motion.Move` and `motion.GetPose`. Section 11 returned that name as a required dependency
from `Validate`, so viam-server brings the gripper up before the palletizer, but the
constructor never stored a handle to it. Calling `gripper.Grab` / `gripper.Open` /
`gripper.IsHoldingSomething` needs the typed `gripper.Gripper` handle, not the name.

So before the new states can call those methods, resolve the gripper out of `deps` and
store it on the struct — the same dependency-resolution step you did for `pick_station` in
Section 16 and `pallet` in Section 17, applied to the gripper. The gripper has its own typed
Go interface, so unlike those two generic components, the handle is a `gripper.Gripper`, not
a `resource.Resource`:

- Add a `gripper gripper.Gripper` field to the palletizer struct.
- Import `go.viam.com/rdk/components/gripper`.
- In `NewPalletizer`, resolve the handle and store it:

```go
grip, err := resource.FromDependencies[gripper.Gripper](deps, gripper.Named(conf.Gripper))
if err != nil {
    return nil, fmt.Errorf("failed to get gripper %q: %w", conf.Gripper, err)
}
p.gripper = grip
```

No Config or `Validate` change is needed — `gripper` is already a required dependency from
Section 11. This step only adds the field, the import, and the constructor resolution.

### Concept

The first new state is `ENABLING_VACUUM`. It sits between `MOVING_TO_PICKUP_HOME` and
`GRASPING`, and its only job is to call `gripper.Grab` before the arm descends onto the
box.

Why a separate state instead of just calling Grab at the top of `GRASPING`? Two reasons:

- The vacuum needs to be engaged *before* the gripper touches the box, so the cup is
  already pulling when it makes contact. Calling Grab during the descent leaves the
  gripper pressed against the box with no suction yet.
- A separate state shows up in `status` and the state diagram. The operator can see the
  cycle engaging suction, and `stop` mid-cycle parks here cleanly without leaving the
  vacuum in an undefined state.

The state method is small — one `gripper.Grab(ctx, nil)` call, then transition to
`GRASPING`. A call to `Grab` looks like:

```go
if _, err := p.gripper.Grab(ctx, nil); err != nil {
    return stateEnablingVacuum, fmt.Errorf("gripper.Grab at ENABLING_VACUUM: %w", err)
}
```

`Grab` takes a context plus an `extra map[string]interface{}` and returns `(bool, error)`. The
bool reports whether the gripper grabbed *something*, useful on real hardware
that can sense seal-vs-empty during the Grab call itself; we drop it here with
`_` because the dedicated `CHECKING_SUCTION` state owns the
seal-confirmation question. Wrap any non-nil error with state-context so the
LOGS show which state was running when the call failed; the state machine
routes any non-nil error to `stateError` regardless of which `cycleState` you
return. `Open` takes the same `(ctx, extra)` signature and returns
just an `error`; `IsHoldingSomething` takes
`(ctx, extra)` and returns `(gripper.HoldingStatus, error)`, where `gripper` is
`go.viam.com/rdk/components/gripper` and `HoldingStatus` is a struct with an
`IsHoldingSomething bool` field — so `status.IsHoldingSomething` reads the seal state.

`GRASPING` itself is unchanged from Section 16: still the `moveToPose` descent. The
descent now runs with the vacuum already engaged, since `ENABLING_VACUUM` ran first.

### Writing your prompt

This is the same kind of work you have already done in §16 and §17 — adding a state to
the cycle. The pattern is by now familiar: a new state constant, a state method, an
entry in the dispatcher's handler map, an `OnTransition` log line, and a new node in
the Web Application's state diagram. One prompt covers all of it.

To recap: we are resolving the gripper handle in the constructor (the
`resource.FromDependencies[gripper.Gripper]` step above), then adding `ENABLING_VACUUM`
between `MOVING_TO_PICKUP_HOME` and `GRASPING`. The new method calls
`gripper.Grab(ctx, nil)` and transitions to `GRASPING`.

Before you prompt, decide:

1. Where does the gripper-handle resolution go, and what type is the field? It mirrors how
   you resolved `pick_station` in §16 and `pallet` in §17 — but the gripper has a typed
   interface, so name the field type and the resolver form (see the snippet above).
2. Which existing state most closely matches what the new state method looks like? Point
   the agent at that as a template so the new code matches the style of what is
   already there.
3. `stateGrasping` is unchanged from Section 16 — call this out in the prompt so the
   agent does not "helpfully" edit it.

### Review the code

- [ ] The gripper handle is resolved via `resource.FromDependencies[gripper.Gripper]` and
      stored on the palletizer struct (a `gripper gripper.Gripper` field), with
      `go.viam.com/rdk/components/gripper` imported
- [ ] `stateEnablingVacuum` constant and method: call `gripper.Grab(ctx, nil)` and transition
      to `stateGrasping`
- [ ] The state method logs a line at the `gripper.Grab` call (e.g. "enabling vacuum via Grab")
      so the Verify step below is reachable
- [ ] Dispatcher's handler map includes the new state in the correct position
- [ ] `stateGrasping` is unchanged from Section 16
- [ ] `OnTransition` logging covers the new state
- [ ] The Web Application's state diagram includes the new node

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] `start` traces the cycle through
      `MOVING_TO_PICKUP_HOME → ENABLING_VACUUM → GRASPING` in order
- [ ] LOGS show `Grab` called at `ENABLING_VACUUM` entry
- [ ] After the cycle's `ENABLING_VACUUM` runs and before `RELEASING_AT_PALLET` runs,
      `IsHoldingSomething` (from the gripper card) reads holding

---

## Part 3. Add CHECKING_SUCTION + the seal-wait helper (25 min)

### Concept

Confirming good suction is critical in vacuum gripping applications. That is why the
second new state is `CHECKING_SUCTION`. It sits between `GRASPING` and
`RETRACTING_NORMAL`, and its job is to verify the seal formed before the arm lifts the
box away. Without this check, a failed grasp would only be noticed when the box never
gets lifted, or worse, when it falls off mid-transit.

A vacuum grasp is a timed sequence: turn the pump on, wait for the seal to form, then
poll the seal sensor until it reports holding or the window expires. The "wait" and
"poll" parts mean we need three configurable timings, all on the palletizer side:

- `suction_settle_secs` — how long the method waits after `Grab` returns before
  reading `IsHoldingSomething` for the first time. The vacuum cup needs a moment to
  start pulling; reading the sensor too soon just gets back "not holding" before any
  seal has had a chance to form. Default `0.5` (500 ms).
- `suction_check_interval_secs` — once the settle dwell is over, how often the
  method re-reads `IsHoldingSomething` until it sees holding (or until time runs
  out). A small interval catches the seal sooner; a large one reduces gRPC traffic to
  the gripper. Default `0.2` (200 ms).
- `suction_check_timeout_secs` — the total time the method is willing to wait *after
  the settle dwell* for a positive read. If the deadline passes with no holding read,
  the method gives up and returns a timeout error. Default `1.0` (1 s).

Visually, with the defaults:

```
  Grab returns
       │
       ▼
       │←── settle ──→│←──── timeout window ───────→│
       │   (0.5 s)    │           (1.0 s)           │
       │              │                             │
       │              ◇     ◇     ◇     ◇     ◇     X
       │            t=0.5 t=0.7 t=0.9 t=1.1 t=1.3 t=1.5
       │             read  read  read  read  read give up
       │
       └─ no reads during the settle dwell (no point asking yet)
```

All three are optional `float64` seconds with the listed defaults. `Validate` does not
require any of them.

**The seal-wait helper.** We are going to make a helper method called `waitForSeal`
that:

- **Blocks** the calling goroutine. The `stateCheckingSuction` method calls
  `waitForSeal` and waits for it to return; the method does *not* spawn a goroutine.
  The cycle should not advance to `RETRACTING_NORMAL` (or to `stateError`) until the
  seal question is settled.
- Returns `(true, nil)` when the seal is confirmed, `(false, nil)` on a timeout
  (no error — timeout is a legitimate outcome the caller decides what to do with),
  and `(false, ctx.Err())` when the context is cancelled (e.g., the operator pressed
  `stop`).
- Treats *every* wait as cancellable. Both the initial `settle` dwell and the
  inter-poll `interval` use `sleepCtx` / `select { ctx.Done(); time.After(...) }`
  rather than raw `time.Sleep`. A `stop` pressed anywhere in the helper — during
  the settle, between polls, or mid-poll — returns immediately instead of blocking
  until the current sleep elapses.

```go
// waitForSeal blocks until the gripper reports a sealed grasp, the
// timeout elapses without a positive read, or the context is cancelled.
//
// Return values:
//   - (true,  nil)        : seal confirmed
//   - (false, nil)        : timeout — no holding read inside the window
//   - (false, ctx.Err())  : context cancelled (e.g. operator stop)
func (p *palletizer) waitForSeal(ctx context.Context, settle, interval, timeout time.Duration) (bool, error) {
    // Initial settle dwell. No reads during this window — the
    // vacuum cup has not had time to form a seal yet, so any
    // reads here would just come back not-holding. Use sleepCtx
    // (the helper from §13) so a stop pressed during the settle
    // returns immediately instead of blocking for the full window.
    if err := sleepCtx(ctx, settle); err != nil {
        return false, err
    }

    // After the settle, poll IsHoldingSomething until it reports
    // holding or we run past the timeout deadline.
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        status, err := p.gripper.IsHoldingSomething(ctx, nil)
        if err == nil && status.IsHoldingSomething {
            return true, nil // seal confirmed
        }

        // Context-aware sleep until the next poll. A plain
        // time.Sleep(interval) here would block cancellation for
        // up to `interval` after the operator hits stop.
        select {
        case <-ctx.Done():
            return false, ctx.Err()
        case <-time.After(interval):
        }
    }

    // Window elapsed with no holding read.
    return false, nil
}
```

The Config values are `float64` seconds; convert each to a `time.Duration` with
`time.Duration(secs * float64(time.Second))` before passing them to `waitForSeal`. The
`stateCheckingSuction` method calls the helper, transitions to `RETRACTING_NORMAL` when
it returns `true`, and returns an error when it returns `false` (the state machine
routes any non-nil error from a state method to `stateError`).

No retry on timeout in this section. A timeout is terminal: the cycle stops in the
error state and the operator restarts.

The cycle becomes:
```
MOVING_TO_PICKUP_HOME → ENABLING_VACUUM → GRASPING → CHECKING_SUCTION → RETRACTING_NORMAL → ...
```

### Writing your prompt

To recap: we are adding one new state (`CHECKING_SUCTION`), three optional `float64`
timing Config attrs (`suction_settle_secs`, `suction_check_interval_secs`,
`suction_check_timeout_secs`), and the `waitForSeal` helper that the state uses. The
helper is given above.

Before you prompt, decide:

1. The three timing Config fields are optional `float64`s with defaults 0.5 / 0.2 / 1.0.
   `Validate` does not require any of them.
2. How are defaults applied — in `Validate`, in `NewPalletizer`, or both? A zero value for
   a `float64` field means "unset"; substitute the default where you read it. The module
   framework decodes Config separately for `Validate` and the constructor, so a default
   set in only one place can be silently dropped — apply defaults where the value is
   consumed.
3. What does the `stateCheckingSuction` method do on timeout? Return an error so the
   machine ends in its terminal error state.

### Review the code

- [ ] Three `float64` timing fields on Config with descriptive JSON tags; all optional
- [ ] Timing defaults (0.5, 0.2, 1.0) applied where consumed (not just in `Validate`)
- [ ] `stateCheckingSuction` constant and method: call `waitForSeal` with the
      configured timings, transition to `stateRetractingNormal` on a true return,
      return an error on a false return
- [ ] The state method logs a "seal confirmed" line on the positive (true) path so the
      Verify step below is reachable
- [ ] `waitForSeal` blocks and the inter-poll wait uses
      `select { ctx.Done(): ...; time.After(interval): ... }` so a `stop` returns
      immediately
- [ ] Timeout returns a clear error that appears in `status` and LOGS; it does not
      retry, does not call `gripper.Open`, does not transition back to
      `MOVING_TO_PICKUP_HOME`
- [ ] Dispatcher's handler map includes the new state in the correct position
- [ ] `OnTransition` logging covers the new state
- [ ] The Web Application's state diagram includes the new node

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] `start` traces the cycle through
      `ENABLING_VACUUM → GRASPING → CHECKING_SUCTION → RETRACTING_NORMAL` in order
- [ ] LOGS show `CHECKING_SUCTION` confirming the seal: the gripper reports a sealed
      grasp about a second after `Grab`, so a read partway through the poll window
      returns holding and the cycle continues

---

## Part 4. Add gripper.Open to RELEASING_AT_PALLET (5 min)

### Concept

Parts 2 and 3 added the gripper coordination's two main pieces — engaging suction and
verifying the seal. All that's left is the release at the placement: a single
`gripper.Open` call inside the existing `stateReleasingAtPallet` method. Small enough
that we'll skip the prompt and add it directly.

Section 17's `stateReleasingAtPallet` method is a dwell-only stub: a
context-aware `sleepCtx` wait with no gripper call. Open that method and add a
`gripper.Open(ctx, nil)` call at the top of the body, before the dwell:

```go
func (p *palletizer) stateReleasingAtPallet(ctx context.Context) (cycleState, error) {
    p.logger.Info("gripper.Open at RELEASING_AT_PALLET")
    if err := p.gripper.Open(ctx, nil); err != nil {
        return stateReleasingAtPallet, fmt.Errorf("gripper.Open at RELEASING_AT_PALLET: %w", err)
    }
    // existing dwell — let the seal break. Context-aware (sleepCtx from §13) so a
    // stop during the dwell returns promptly instead of blocking the full window.
    if err := sleepCtx(ctx, time.Duration(p.cfg.PlaceDwellSecs*float64(time.Second))); err != nil {
        return stateReleasingAtPallet, err // cancelled — stay parked on this state
    }
    return stateLifting, nil
}
```

The Open-then-dwell ordering matters: `gripper.Open` initiates the release, and the
seal takes a moment to actually break. On physical hardware, lifting immediately after
`Open` can pull the box up with the gripper. The §17 dwell was put there in advance
for this reason; this Part just adds the `Open` before it.

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] LOGS show `Open` called at `RELEASING_AT_PALLET` entry
- [ ] `IsHoldingSomething` reads holding through transit and reads not-holding after
      `RELEASING_AT_PALLET` runs
- [ ] The arm follows the same motion as Section 17, with `Grab` logged at
      `ENABLING_VACUUM` and `Open` logged at `RELEASING_AT_PALLET`

---

## Done when

You can answer **yes** to all of these:

- [ ] The gripper is `viam:robotiq:simulated-epick-vacuum-gripper` and the
      `robotiq-epick` module is in the config (both from Section 10)
- [ ] `ENABLING_VACUUM` calls `gripper.Grab` and transitions to `GRASPING`
- [ ] `CHECKING_SUCTION` polls `IsHoldingSomething` using the three timing attrs and
      transitions to `RETRACTING_NORMAL` on a positive read
- [ ] `CHECKING_SUCTION` returns a timeout error if no positive read arrives within the
      window, and the machine ends in its terminal error state (no retry path in this section)
- [ ] `RELEASING_AT_PALLET` calls `gripper.Open` and then dwells `place_dwell_secs`
- [ ] The cycle runs end-to-end: `Grab` engages, the seal is confirmed, `Open` releases
- [ ] The three timing attrs are optional with the listed defaults
- [ ] The cycle runs at least 3 cycles consecutively from a single `start`

## Takeaway

The cycle is now coordinated with the gripper end to end:

- `ENABLING_VACUUM` engages suction before the descent.
- `GRASPING` descends onto the box with the cup already pulling.
- `CHECKING_SUCTION` waits for the seal to form (via `waitForSeal`) and routes the
  cycle to `stateError` if it never does.
- `RELEASING_AT_PALLET` opens the gripper and dwells while the seal breaks, before
  `LIFTING` runs.

Three optional Config attrs (`suction_settle_secs`, `suction_check_interval_secs`,
`suction_check_timeout_secs`) tune the seal-check window without code changes. The
`waitForSeal` helper handles the settle-then-poll-with-context-cancellation pattern
that any future gripper-coordination code in this module will reuse.

A timeout in `CHECKING_SUCTION` is terminal in this section.
