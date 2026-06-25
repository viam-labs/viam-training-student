# Section 20: Verify pattern + diagnostic loop

Time: approximately 50 minutes.

At the end of Section 19 the palletizer drives a full pallet end-to-end. The pickup and pallet positions
were carefully chosen so the full motion is within the arm's kinematic workspace. Currently, if we hit a
reach issue, the palletizer stops moving mid-cycle and reports an error in the log.
We want to catch these errors during provisioning and not at run time. Most of these
failures are predictable in advance: the motion planner would refuse the pose if asked, but the
palletizer only asks when it tries to execute the move.

This section adds verification logic and a visualization panel to the Web Application.
We want to check every cycle motion throughout the pallet stack without executing it
and show the per-pose results in the Web Application. The operator runs the verification logic,
identifies poses the planner refuses, adjusts the cell config, and reruns verify until every pose
passes. Then they can run the palletizer for that SKU with confidence.

We will add two DoCommand verbs (`verify_pick_station`, `verify_pallet`) in a new `verify.go` file, add a visual for each in the Web Application, and then test the diagnostic loop against a
misconfigured cell.

## What you will do in this section

- Distinguish planning a motion from executing it; call the motion service's plan-only path
  via `viamkit/verify`
- Combine each pose's result — reachable or not, with a reason on failure — into one response
  the Web Application can show as a table
- Run the operator diagnostic loop: verify, identify the failing pose, adjust the cell,
  reverify, Start

## Setup check

- Section 19 runs end-to-end: pack-sequencer is in the cell; the full 8-box pallet runs from
  one Start press; placed boxes are planner obstacles; `PALLET_COMPLETE` halts cleanly.
- viamkit v0.13.0 or later (`verify.Plan` is provided by this version).

If the cell drifted, restore `prereq-machine-config.json` and reload.

**Packages used in this section.** The plan-only call and the trajectory helper come from two viamkit packages — import them with their full paths:

```go
import (
    "github.com/viam-labs/viamkit/verify"     // verify.Plan, PlanResult
    "github.com/viam-labs/viamkit/kinematics" // kinematics.LastTrajectoryJoints
)
```

**Cycle-state count (going in):** 14 working + 3 terminal (`DONE`, `ERROR`, `PALLET_COMPLETE`), carried in from §19. §20 adds the verify verbs and Web Application panels but no new state-machine states — the count is unchanged.

---

## Part 1. `verify_pick_station`: pre-check the pickup chain (15 min)

### Concept

Before the palletizer commits to a cycle, we want to ask the motion planner a question: *if I
asked you to make this move, could you?* The motion service can answer it without moving the
arm. `motion.Move` plans **and** executes a trajectory, but the SDK also exposes a plan-only
path that returns the planned trajectory — or a refusal — without touching the hardware. We run
that for every move in a cycle. If every plan comes back good, we can be reasonably certain the
real cycle will run. If one comes back bad, the planner tells us *why*, and that reason points
at what to change in the workcell — move the pallet, adjust a home pose, reposition the arm base.

`viamkit/verify` wraps the plan-only call so we don't have to handle the SDK's rough edges. The
request is a `motion.MoveReq` — the *same* struct you'd hand `motion.Move` (the arm's
`ComponentName`, the `Destination` pose, any `Constraints`/`WorldState`) — `Plan` takes a few extra
arguments:

```go
// req is a motion.MoveReq, identical to what motion.Move takes.
res, err := verify.Plan(ctx, p.motion, req,
    "builtin",   // motion service name, almost always "builtin"
    p.cfg.Arm,   // arm name — names which arm the startJoints describe and
                 // which arm's joints to read off the resulting trajectory.
                 // (req.ComponentName is the move target; this is the joint source.)
    startJoints, // nil → the planner uses the arm's current joints; or a prior plan's final joints
    15,          // planner timeout (s) — fail fast on an infeasible plan
)
if err != nil {
    // verify.Plan returns an error only when the call itself breaks — marshalling the
    // request or the DoCommand transport. An unreachable box is NOT an error: the planner
    // reports it as res.Feasible == false (next block). So an err here is a real failure
    // to surface, not a per-box reachability result.
    return nil, err
}
// res is a PlanResult: Feasible (bool), Message (string), Trajectory (interface{}).
if !res.Feasible {
    // res.Message says why the planner refused this pose
}
// res.Trajectory is the planned joint-space trajectory. Pass it (with the arm
// name) to LastTrajectoryJoints to get the joints at the end of this plan —
// feed those to the next plan's startJoints.
nextStart := kinematics.LastTrajectoryJoints(res.Trajectory, p.cfg.Arm)
```

When a plan fails, the reason falls into one of a few kinds:

- **Out of reach** — there is no joint solution for the pose; the arm physically can't get
  there (`no solution found`).
- **Constraint-limited** — a solution exists, but none that also satisfies the orientation
  constraint we asked for (`no solution found within constraint tolerance`); relaxing the
  orientation constraint tells this apart from out-of-reach.
- **Collision** — the planned path would pass through a known geometry (`collision detected: …`).

**Start joints matter.** A destination can be reachable from one arm pose and unreachable from
another, and this initial condition is an important aspect of trajectory verification. `verify.Plan`
has a `startJoints` parameter that lets you specify the arm's initial positions. If you look at
the palletizer cycle, we have two home poses, pickup-home and pallet-home. These are safe
poses that each motion returns to for consistency. For our first plan calls, we can set
`startJoints` to `nil`, and the planner will use the arm's current joints. Once we have a plan that
takes us to the home poses, we can use the `kinematics.LastTrajectoryJoints` helper to get consistent arm positions
for `startJoints` from those home poses. Each later plan starts from the previous plan's **final joints**,
pulled from its trajectory with `kinematics.LastTrajectoryJoints`, so the joint trajectory verify
checks is the one the real arm would follow.

The pickup is a multi-move chain: `pickup_home → grasp`, then `grasp → retract_normal`, then
`retract_normal → retract_along_conveyor`. We plan each segment, chaining the joints, and gather
the per-segment results into one response the Web Application can render. Creating a readable report is
just as important as running the verification steps. For the result, it would be helpful to list:

- `results` — one entry per segment: `reachable` [true or false], and a `reason` if it fails.
- `total`, `reachable`, `unreachable` — the counts.
- `all_pass` — true when every segment is reachable.
- `start_source` — which start-joint source the run used (`current` or `explicit`).

It is a good idea to log each result as you go, so a failed segment shows up in LOGS, not only in the returned
response.

### Writing your prompt

We're adding a `verify_pick_station` verb to a new `verify.go` file. We want to run `verify.Plan` for each arm
motion on the pickup side of the state machine and report whether the arm can reach each step. We also have
some specific requirements for the report. Work through these questions as you form your prompt:

1. Where does the verify code live, and which package gives you the plan-only call?
2. Which four poses does the pickup chain run through, and which helper methods give you the target poses?
3. How do you set the `startJoints` parameter for the first segment? What about the rest?
4. How do you feed each plan's final joints into the next?
5. Each segment uses a different constraint (orientation-locked for the approach, linear for the
   retracts). How do you keep those choices identical to the cycle's state handlers instead of
   hardcoding them in verify?
6. What data should the response have so the Web Application can display a pass/fail per segment and a reason?
7. How do you log each segment's result so a failure — and its reason — shows up in LOGS, not just the returned response?

### Review the code

- [ ] `verify_pick_station` uses `viamkit/verify.Plan`, not a hand-rolled DoCommand into motion
- [ ] The first plan passes `nil` for `startJoints` (the planner uses the arm's current joints)
      or an explicit value; the chosen source is reported in the response
- [ ] The three plans chain: each plan's final joints are the next plan's start joints
- [ ] Per-segment constraint selection matches the cycle's runtime constraints (orientation-locked
      approach, linear retracts), not hardcoded in verify
- [ ] The response carries `{total, reachable, unreachable, results, all_pass, start_source}`, with
      one `results[i]` per segment (`reachable`, plus a `reason` on failure)
- [ ] Each segment's result is logged (reachable, with the reason on failure), so failures show up
      in LOGS as well as the response

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] Call `verify_pick_station` with `{}` from the CONTROL tab — the response has `total: 3`,
      three results, and a populated `start_source`
- [ ] With the stock prereq config, the response reports `all_pass: true`
- [ ] The LOGS show one line per segment with its result

---

## Part 2. `verify_pallet`: pre-check every placement (20 min)

### Concept

`verify_pick_station` verified the motions around the pick station. `verify_pallet` is the same
plan-don't-execute approach across the pallet side, but the placements are more dynamic — each box
gets a slightly different trajectory. `pack-sequencer.get_pack_order` gives us the poses for the whole
pack: one entry per box, each with the box's place pose (`PoseInWorld`), an approach offset, and its
`Seq` / `Layer` / `Col` / `Row`. Let's loop over them and verify each:

```go
order, err := p.packSequencer.GetPackOrder(ctx)
if err != nil {
    return nil, err
}
for _, box := range order.Placements {
    // box.PoseInWorld is PlaceEnd; apply the approach offset for PlaceStart — the same way
    // the cycle derives them, so verify and the real run agree.
    // plan PlaceStart from the start joints, then PlaceEnd chained from it (both moveLinear),
    // and record box.Seq / box.Layer / box.Col,box.Row with the reachable/reason result.
}
```

Remember that each trajectory going down to the pallet goes from pallet-home to
`PlaceStart`, and approaches the pallet at an angle to `PlaceEnd`, then after it releases the vacuum,
moves straight up and back to pallet-home.

The chaining here works differently from the pick-station. Each box is two plans —
`PlaceStart` then `PlaceEnd` — and `PlaceEnd` chains from `PlaceStart`'s final joints, the same
way the pick-station segments chained. But the boxes do **not** chain to each other: every box's
`PlaceStart` plans from the *same* pallet-home start joints — passed as `nil` so the planner uses
the arm's current (parked) joints — because the arm doesn't actually move during verification, it
returns to pallet-home before each real place. So the chain resets at the start of every box.

### Writing your prompt

We are creating `verify_pallet` in `verify.go` and following the same pattern as Part 1, but now we are looking
at every box `get_pack_order` returns. Work through these questions as you form your prompt:

1. `get_pack_order` gives each box's place pose and approach offset — how do you turn those into
   the `PlaceStart` / `PlaceEnd` pair, matching how the cycle derives them?
2. Each placement is two plans — how do you chain them, and which constraint does each use?
3. The response mirrors `verify_pick_station`, one row per box — which fields identify the box?
4. How do you log each box's result, the same way `verify_pick_station` logs each segment?

### Review the code

- [ ] `verify_pallet` loops `p.packSequencer.GetPackOrder(...).Placements` and runs
      `viamkit/verify.Plan` per box, with start joints resolved the same way as
      `verify_pick_station`
- [ ] Each box's `PlaceStart` / `PlaceEnd` are derived to match the cycle's place poses
- [ ] Per box: `PlaceStart` is planned first, then `PlaceEnd` chained from its final joints;
      both use `moveLinear`
- [ ] The response uses the same envelope as `verify_pick_station`, with `total: 8` and each
      `results[i]` adding `seq`, `layer`, `col`, `row` to `reachable`/`reason`
- [ ] Each box's result is logged the same way (reachable, with the reason on failure)

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] Call `verify_pallet` with `{}` from the CONTROL tab — the response has `total: 8`,
      `reachable + unreachable == 8`, eight results, and a populated `start_source`
- [ ] With the stock prereq config, the response reports `all_pass: true`
- [ ] The LOGS show one line per box with its result
- [ ] If any are unreachable on the stock config, it's likely a verify bug — investigate before moving on

---

## Part 3. Show verify in the Web Application (10 min)

### Concept

Now that we have the verification tools on the back end, we want a way to visually show the results.
Add a Verify panel below the Run controls: two buttons (`Verify Pallet`, `Verify Pickup`), a results
table with one row per result (green if reachable, red if unreachable), and a summary line
at the top showing the `start_source`.

### Writing your prompt

We want to add a Verify panel in the Web Application, with a button
per verb and a colour-coded results table. Work through these questions as you form your prompt:

1. What does each button call, and what does that DoCommand look like in JavaScript?
2. How do you render the results — one row per result, coloured by `reachable`, with the reason
   shown on failures?
3. Verify takes 5–10 seconds; should we change the button name or colour while waiting for a response?

### Review the code

- [ ] Verify panel is added below the Run controls in the Web Application
- [ ] Two buttons: `Verify Pallet` and `Verify Pickup`
- [ ] Results render as one row per result; row colour is green or red based on `reachable`
- [ ] Red rows display the `reason` string
- [ ] The summary line shows `X of Y reachable, source: <start_source>`

### Verify

- [ ] Click `Verify Pallet`. The panel populates with 8 rows, all green (on a clean cell).
      The summary shows `8 of 8 reachable, source: current`
- [ ] Click `Verify Pickup`. Three rows, all green; the summary shows `3 of 3 reachable`.

---

## Part 4. The diagnostic loop drill (15–20 min)

### Concept

A misconfigured cell — a moved pallet, or a new SKU with more or bigger boxes — can put a
placement outside the arm's reach. The diagnostic loop catches that before a run: verify, read
the red rows, adjust the cell config, rerun verify, and iterate until every row is green, then
press Start.

The exercise is the iteration loop, not the diagnosis of any specific misconfiguration. In
production an operator runs this loop every time the pallet is moved or a new SKU is introduced.

### Manual exercise — no prompt

Now that we can verify the motions from end to end, we can make changes to the cell and pack-sequencer
to test the arm's reach:

- [ ] Click `Verify Pallet` and `Verify Pickup`. Confirm everything is green.
- [ ] Go to the machine configuration and change the pack-sequencer's `quantity` attribute from 8 to 32. You will need to save the configuration. What happens when you click `Verify Pallet` now?
- [ ] Read the failure reason. Common reasons:
   - `no solution found within constraint tolerance` — the orientation constraint cannot
     be satisfied for this pose (constraint-limited)
   - `no solution found` — no IK solution exists for the pose (workspace-limited)
   - `collision detected: ...` — the planned path collides with a known geometry
- [ ] Start the palletizer
   - How far does the palletizer pack boxes?
   - Did it transition to the error state?
   - What message do you see in the logs?
- [ ] Revert the quantity back to 8.
- [ ] Now move the pallet 1 meter away — increase the `pallet` component's `frame.translation.y` by 1000 mm (from 500 to 1500) — and save. What happens when you click `Verify Pallet` now?
- [ ] Revert the pallet's location and `Verify Pallet`, is everything back to green?
- [ ] Click Start and verify a successful pack.

---

## Done when

You can answer **yes** to all of these:

- [ ] `verify_pick_station` and `verify_pallet` work from the CONTROL tab — each returns the
      structured response (`total`, `reachable`, `unreachable`, `results`, `all_pass`,
      `start_source`) and logs each result
- [ ] On the stock cell both report `all_pass: true` — three results for the pickup, eight for
      the pallet
- [ ] The Verify panel sits below the Run controls in the Web Application: two buttons, one
      colour-coded row per result, the reason on red rows, and a summary line
- [ ] You ran the diagnostic loop end-to-end: introduced a reach problem, saw it as red rows,
      fixed the cell, reverified to all green, and ran a clean pack

## Takeaway

Verify is the **plan-before-execute** pattern: run the motion planner over every pose without
moving the arm, so the operator catches an unreachable placement before a cycle commits to it.
The work moves from recovering after a failed cycle to adjusting the cell before one runs — the
pre-flight check an operator runs before each pack on real hardware.

With verify in place the simulated palletizer is feature-complete: a pack-aware pick-and-place
cycle with gripper coordination, dynamic obstacles, 3D-scene visuals,
and pre-flight verification.
