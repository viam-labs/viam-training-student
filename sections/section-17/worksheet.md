# Section 17: Pallet, place states, and the full motion cycle

Time: 80 minutes.

Section 16 finished the pickup half: the arm reaches pickup-home, descends to the grasp
pose, retracts up and along the conveyor, and returns to pickup-home. This section builds
the place half. By the end of this section the cycle will grow from 6 working states to 11
and trace the entire pick-and-place motion path.

We are going to add states for placing the box on the pallet: `MOVING_TO_PALLET_HOME`,
`MOVING_TO_PLACE`, `RELEASING_AT_PALLET`, `LIFTING`, and `RETURNING_TO_PALLET_HOME`.

## What you will do in this section

- Wire `pallet` as a required palletizer dependency
- Query the pallet-home and placement poses from the pallet component
- Add 5 place-side states and update the Web Application's state diagram
- Use linear (straight-line) constraints for `MOVING_TO_PLACE` and `LIFTING`
- Run the full motion cycle end-to-end: pickup → transit → place → lift → return

## How this section works

Each Part teaches a concept, you write the prompt yourself, you review the generated code
against a checklist, and then you verify behavior.

## Setup check

- Section 16 runs end-to-end: the cycle traces all five pickup states and the arm descends
  and retracts.
- The palletizer has `arm`, `gripper`, `pick_station`, `box_width_mm`,
  `box_length_mm`, `box_height_mm`. Resource graph green.
- `viam module reload --part-id <PART_ID>` reinstalls cleanly.

If the cell drifted, restore `prereq-machine-config.json` and reload.

**Cycle-state count (going in):** 6 working from §16 (`IDLE` plus the five pickup states) + 2 terminal.

---

## Part 1. Explore the pallet's pose API (10 min)

### Concept

The pallet component knows its own size, where its top face is, and where its corners are
— and exposes those as DoCommand verbs the place states will use. Each label on the left
of the diagram below has a matching verb on the right:

```
                   ● pallet-home pose             get_pallet_home_pose
                   ┊  (safety_height_mm above
                   ┊   the top face)
                   ▼
          ●━━━━━━━━━━━━━━━●                        get_corner_poses
         ╱       ◆       ╱┃                         (the four top corners ●,
        ●━━━━━━━━━━━━━━━●  ┃                         CCW from bottom-left)
        ┃               ┃  ┃
        ┃     pallet    ┃  ●                       get_top_face_center
        ┃               ┃ ╱                         (◆ center of the top face)
        ●━━━━━━━━━━━━━━━●

        ● = corner   ◆ = top-face center
```

**pallet attributes and frame:** the pallet is already in the cell from Section 10. Its
attributes and frame are:

```json
"attributes": {
  "width_mm": 500,
  "length_mm": 350,
  "thickness_mm": 100
},
"frame": {
  "parent": "world",
  "translation": {"x": 200, "y": 500, "z": 100}
}
```

The `frame.translation` is the center of the pallet's **geometry** (its centroid, not a
face) — the top face sits half the thickness above it, which is why the corner and
top-face poses come back about `thickness_mm / 2` higher in Z than the translation.

- `width_mm`, `length_mm`, `thickness_mm` — the pallet's own physical dimensions. They
  determine where the top face sits and where the corners are.

The three DoCommand verbs in detail:

- **`get_pallet_home_pose`** returns a world-frame pose above the pallet top, gripper
  pointing down. `safety_height_mm` controls the distance from the pallet top to the
  waypoint.

  ```json
  { "get_pallet_home_pose": { "safety_height_mm": 200 } }
  ```

- **`get_top_face_center`** returns the center of the pallet's top face as a flat 6D
  pose, gripper pointing down. This section uses it as the placeholder place pose (a later
  section replaces it with the real placement).

  ```json
  { "get_top_face_center": true }
  ```

- **`get_corner_poses`** returns the four corners of the pallet's top face,
  counter-clockwise from bottom-left, with gripper-down orientations, wrapped in a
  `"corners"` array. The cycle doesn't use it in this section — a later section
  drives the real placements.

  ```json
  { "get_corner_poses": true }
  ```

### Manual exploration — no prompt

In the Viam App, open the pallet's component card. Run each call. Record the values.
Sanity-check that they are near the pallet's configured translation in the cell config
(around 200, 500, 100).

### Verify

- [ ] All three calls return world-frame poses
- [ ] `get_pallet_home_pose` Z is approximately `safety_height_mm` above the pallet top
- [ ] `get_top_face_center` Z is at the pallet top — the translation Z plus half the thickness
- [ ] `get_corner_poses` returns 4 poses, all with gripper-down orientations

---

## Part 2. Add the pallet resource (15 min)

### Concept

This Part adds two palletizer config attributes:

- `pallet` — required, a string holding the pallet component's configured name. Same
  dependency pattern as `pick_station` from Section 16.
- `safety_height_mm` — the height above the pallet the arm will move during each cycle
  before lowering to place the box. This is used to define the pallet-home pose.
  **Required.** Used as the argument to `get_pallet_home_pose`.

The mechanics here are the same as the `pick_station` addition in Section 16: declare the
Config field, return it as a dependency from `Validate`, resolve the handle out of the
dependency map in the constructor. Pallet is also a `generic` component, so the resolved
handle is a `resource.Resource` fetched via `gencomp.Named` (the aliased generic-component
import from Section 16), just like the pick-station.

### Writing your prompt

To recap: we are adding two Config attrs to the palletizer — `pallet` (the dependency name)
and `safety_height_mm` (the waypoint distance) — returning `pallet` as a dependency from
`Validate`, and resolving the pallet handle in the constructor.

Before you prompt, decide:

1. The `pallet` field on the Config struct mirrors `pick_station`. What is the type? What
   is the JSON tag?
2. `safety_height_mm` is a required `float64`. Like the box-dimension attrs in Section 16,
   `Validate` rejects a config that's missing it or has it as zero.
3. The machine-config edit (adding `"pallet"` and `"safety_height_mm"` to the palletizer's
   attributes) is a manual step you make after the code change.

Now write the prompt.

### Review the code

- [ ] `pallet` field on Config, required in `Validate`, appears in the dependency list
- [ ] `safety_height_mm` field on Config, **required in `Validate`** (rejects a missing or
      zero value)
- [ ] Resolved pallet handle is stored on the palletizer struct after `NewPalletizer`

### Manual config edit

In the app's CONFIGURE tab → JSON, add to the palletizer's attributes:
```json
"pallet": "pallet",
"safety_height_mm": 200
```

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] The palletizer is green and the resource graph shows a new edge to `pallet`
- [ ] Removing `pallet` causes `Validate` to reject the config with a clear error;
      restoring it returns the resource to green

---

## Part 3. Helpers for the place-side poses (15 min)

### Concept

Now that the pallet is wired in, the place states need two pieces of information: where
the pallet home pose is, and where the box will be placed on the pallet. The two have
different call patterns, so we handle them two different ways.

We are going to make a new `pallet.go` file with helper methods around the pallet's
DoCommand verbs — the same pattern as the `pick_station.go` you wrote in Section 16.

**Pallet-home pose:** the same pattern as Section 16's
`resolvePickupHomePose`. The cycle queries `pallet.get_pallet_home_pose` whenever it
needs to move to the safe waypoint above the pallet, and two states call it
(`MOVING_TO_PALLET_HOME` and `RETURNING_TO_PALLET_HOME`), so a helper that returns the
typed pose fits:

```go
func (p *palletizer) resolvePalletHomePose(ctx context.Context) (spatialmath.Pose, error) {
    resp, err := p.pallet.DoCommand(ctx, map[string]any{
        "get_pallet_home_pose": map[string]any{"safety_height_mm": p.cfg.SafetyHeightMM},
    })
    if err != nil {
        return nil, fmt.Errorf("get_pallet_home_pose: %w", err)
    }
    return pose6DMapToPose(resp), nil
}
```

It reuses the `pose6DMapToPose` parser you wrote in Section 16.

**Placement pose:** the placement is used twice in a cycle — by `MOVING_TO_PLACE` for the
descent and by `LIFTING` for the lift back. Both states need the *same* pose, and it
shouldn't change part-way through a cycle, so it makes sense to look it up once and hold
onto it rather than re-query it in each state. A single `placePose` field on the cycle
struct is a natural place to keep it. (This is the same per-cycle state you added in
Section 13; the pallet-home pose, by contrast, is re-queried each time a state needs it,
since the safe waypoint is fixed.) For this section the placement is just the pallet's
top-face center — a stand-in that a later section will replace with a real placement.

`get_top_face_center` returns a flat 6D-pose map, so the `pose6DMapToPose` parser from
Section 16 reads it directly — no unwrapping:

```go
resp, err := p.pallet.DoCommand(ctx, map[string]any{"get_top_face_center": true})
if err != nil {
    return nil, fmt.Errorf("get_top_face_center: %w", err)
}
placePose := pose6DMapToPose(resp) // §16's helper, reused
```

`LIFTING`'s target is just `placePose` translated up in Z; the offset is computed inline
in the `LIFTING` handler — no separate cached pose needed.

### Writing your prompt

To recap: we are adding `resolvePalletHomePose` in a new `pallet.go` file, a single
`placePose` field on the cycle struct, and the on-entry logic in the
`MOVING_TO_PALLET_HOME` handler that queries `pallet.get_top_face_center` and caches
it as `placePose`.

Before you prompt, decide:

1. Where should `resolvePalletHomePose` live? How did we handle the pick-station and its
   DoCommand verbs?
2. Where should `placePose` live during a cycle? How did we handle per-cycle variables
   previously? If you get lost, look at the cycle struct we used in Section 13.

### Review the code

- [ ] `resolvePalletHomePose(ctx)` lives in `pallet.go`, returns
      `(spatialmath.Pose, error)`, and passes the configured `safety_height_mm`
- [ ] The cycle struct has one new `Pose` field, `placePose`
- [ ] On entry to `MOVING_TO_PALLET_HOME`, the handler queries `pallet.get_top_face_center`
      once and stores it as `placePose`
- [ ] The placeholder placement is marked with a comment noting a later section replaces it
- [ ] `MOVING_TO_PLACE` and `LIFTING` read `placePose` from cycle state, not by
      re-querying the pallet

### Verify

Nothing calls these helpers yet, so there's nothing to run here — a clean compile is all
we're after. The place states wire them up in Part 4.

- [ ] `make` is clean

---

## Part 4. Five new place-side states + full motion cycle (35 min)

### Concept

With the helpers in place, we can add the five new states that carry the box over and
place it on the pallet. They join the six working states from Section 16 (`IDLE` plus the
five pickup states) — eleven working states total. Here's the full cycle:

```
IDLE
 → MOVING_TO_PICKUP_HOME
 → GRASPING (descent)
 → RETRACTING_NORMAL
 → RETRACTING_ALONG_CONVEYOR
 → MOVING_TO_PALLET_HOME      ← NEW: transit to the pallet, orientation-locked (box stays level)
 → MOVING_TO_PLACE            ← NEW: linear descent onto the placement
 → RELEASING_AT_PALLET        ← NEW: release the vacuum (placeholder — gripper.Open comes later)
 → LIFTING                    ← NEW: linear lift up and away from the box
 → RETURNING_TO_PALLET_HOME   ← NEW: back to the pallet waypoint, above the boxes
 → RETURNING_TO_PICKUP_HOME
 → DONE  (terminal — restart for the next cycle)
```

**Why different constraint styles?** Each state picks the lightest constraint that still
protects the box:

- **`moveOrientationLocked`** while the box is in transit (`MOVING_TO_PALLET_HOME`, and
  §16's `GRASPING` descent) — keep the gripper pointed down so the carried box stays
  level and doesn't twist. We pin the orientation, not the path.
- **`moveLinear`** on the descent and the lift (`MOVING_TO_PLACE`, `LIFTING`) — the
  gripper must travel a straight line so a held or just-placed box doesn't clip a
  neighbouring box on the way down or up.
- **`moveDefault`** on the empty return (`RETURNING_TO_PALLET_HOME`) — the arm is
  carrying nothing, the planner has room, and we only care that it gets back
  collision-free. A tighter constraint here just burns planner budget for no gain.

Let's look at the new states in detail:

- **`MOVING_TO_PALLET_HOME`** — `moveToPose(resolvePalletHomePose(), moveOrientationLocked)`.
  The transit carrying the box from the pickup side over to the pallet's safe waypoint,
  gripper-down the whole way. On entry the handler also queries `pallet.get_top_face_center`
  once and caches the result as `placePose` on the cycle state — the place states
  downstream read that cached pose instead of re-querying.
- **`MOVING_TO_PLACE`** — `moveToPose(cycle.placePose, moveLinear)`. A linear descent
  from the pallet-home waypoint down to the placement pose. The linear constraint keeps the held box
  travelling on a predictable straight line through any neighbouring placed boxes.
- **`RELEASING_AT_PALLET`** — a stop at the placement pose. A later section adds
  `gripper.Open` at the start of this handler; the vacuum then needs a beat to break
  suction before the arm lifts, or the box gets dragged along. That pause is
  `place_dwell_secs`. In this section the gripper is still inert, so the handler is just
  the dwell — but write it as a **context-friendly** wait (a `select` on `ctx.Done()` /
  `time.After`, not a bare `time.Sleep`) so a cancel during the dwell returns promptly.
- **`LIFTING`** — `moveToPose(placePose translated +100 mm in Z, moveLinear)`. A linear
  lift straight up off the placement before the transit back, so the
  (now-released) box stays put. The offset is computed inline in the handler from
  `cycle.placePose`.
- **`RETURNING_TO_PALLET_HOME`** — `moveToPose(resolvePalletHomePose(), moveDefault)`.
  The empty transit back to the pallet waypoint — same target as `MOVING_TO_PALLET_HOME`,
  but `moveDefault` since the arm is no longer carrying anything.
- **`RETURNING_TO_PICKUP_HOME`** — exists from Section 16. Confirm the dispatcher routes
  through it correctly under the new cycle.

`place_dwell_secs` is a new Config attribute — optional, defaulted to 0.5 seconds at the
call site when the state is `RELEASING_AT_PALLET`, matching the `safety_height_mm` pattern from
Part 2. Tuning knob, not a wiring fact, so `Validate` stays out of it.

### Writing your prompts

Don't try to land all five states in one prompt. That's a large change to review at
once, and if the cycle misbehaves you won't know which state to blame. Build the place
half **one state at a time**: prompt for a single state, hot-reload, watch it light up
in the 3D scene and the state diagram, then move to the next. The state-by-state list
above is your spec — one entry per prompt.

Each state is the same structural change you made in Section 16: a new state constant, a
handler method, a dispatcher entry, and an `OnTransition` log line. So each prompt names
one state, its constraint style, and what its handler does. Work in cycle order:

1. **`MOVING_TO_PALLET_HOME`** — orientation-locked transit to the pallet waypoint; on
   entry, query `pallet.get_top_face_center` once and cache it as `placePose` on the
   cycle state. Do this one first — the next two states read that cached pose.
2. **`MOVING_TO_PLACE`** — linear descent to the cached `placePose`.
3. **`RELEASING_AT_PALLET`** — the context-friendly dwell. This prompt also adds the
   `place_dwell_secs` Config attribute: `float64`, optional, defaulted to 0.5 s at the
   call site in the handler (not in `Validate`, not in `NewPalletizer` — same pattern as
   `safety_height_mm` in Part 2).
4. **`LIFTING`** — linear lift to `placePose` translated +100 mm in Z, computed inline.
5. **`RETURNING_TO_PALLET_HOME`** — default-constraint transit back to the pallet
   waypoint (the arm is empty now, so no orientation lock).

Then two small follow-ups:

6. Confirm the dispatcher routes through `RETURNING_TO_PICKUP_HOME` (from Section 16) and
   on to DONE under the new cycle. That path already exists — you're checking it, not
   rebuilding it.
7. Add the five new nodes to the Web Application's state diagram. Where is that list, and
   what is the minimal edit?

One `start` runs the whole cycle once and ends at DONE (terminal, per §13); the operator
presses `restart` to run it again. A later section introduces a `PALLET_COMPLETE`
terminal that distinguishes "this cycle finished" from "this pallet is done"; until then,
one cycle per `start`.

### Review the code

- [ ] Five new state constants declared with descriptive names
- [ ] Five new handler methods following the existing pattern
- [ ] Each handler uses the correct constraint style: `moveOrientationLocked` for
      `MOVING_TO_PALLET_HOME`; `moveLinear` for `MOVING_TO_PLACE` and `LIFTING`;
      `moveDefault` for `RETURNING_TO_PALLET_HOME`; no `moveToPose` call in
      `RELEASING_AT_PALLET`
- [ ] `RETURNING_TO_PALLET_HOME` targets the same pallet-home pose as
      `MOVING_TO_PALLET_HOME` but with `moveDefault` (the arm is empty on the way back)
- [ ] `RELEASING_AT_PALLET` is a context-friendly dwell only in this section — the wait
      selects on `ctx.Done()` (not a bare `time.Sleep`), and there is no `gripper.Open`,
      no `gripper.IsHoldingSomething`, no gripper API calls of any kind (a later section
      adds the `gripper.Open` call)
- [ ] `MOVING_TO_PALLET_HOME` entry queries `pallet.get_top_face_center` once and stashes
      it as `placePose` on cycle state; the place states read it without
      re-querying
- [ ] `LIFTING`'s target is computed inline as `placePose` translated +100 mm in Z
- [ ] `place_dwell_secs` is an optional Config attribute, defaulted to 0.5 s **at
      the call site in `RELEASING_AT_PALLET`** (not in Validate, not in NewPalletizer)
- [ ] The dispatcher's handler map covers the five new states
- [ ] `OnTransition` logging covers the five new states
- [ ] The Web Application's state diagram includes the five new nodes

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] `start` traces the full cycle in the 3D scene: pickup → grasp descent → retracts →
      long transit → place descent → dwell → lift → transit back → DONE
- [ ] The state diagram highlights each new state in order as the cycle reaches it
- [ ] `status` reports each new state by name
- [ ] A `stop` issued during `RELEASING_AT_PALLET` returns promptly — the dwell is a
      context-aware wait, so cancellation doesn't block for the full `place_dwell_secs`
- [ ] `stop` mid-cycle parks the cycle; `start` resumes; `restart` resets
- [ ] Three consecutive cycles each kicked off by `restart` run cleanly (DONE remains
      terminal per §13; a later section introduces the multi-cycle loop)
- [ ] Visually: the arm reaches the center of the pallet top, dwells for approximately 0.5 s, lifts,
      and returns

---

## Done when

You can answer **yes** to all of these:

- [ ] The palletizer depends on `pallet` and rejects a config that omits it
- [ ] `safety_height_mm` is a required Config attribute (`Validate` rejects a missing or
      zero value); `place_dwell_secs` is optional, defaulted to 0.5 s at its call site
- [ ] The full motion cycle traces end-to-end in the 3D scene: 11 working states, each
      observable via `status` and the state diagram
- [ ] `MOVING_TO_PALLET_HOME` uses `moveOrientationLocked` for the loaded transit (the box
      stays level); `RETURNING_TO_PALLET_HOME` uses `moveDefault` for the empty return
- [ ] `MOVING_TO_PLACE` and `LIFTING` use `moveLinear`
- [ ] `RELEASING_AT_PALLET` is a dwell only in this section — no gripper API calls
- [ ] On entry to `MOVING_TO_PALLET_HOME`, the cycle queries `pallet.get_top_face_center`
      and stashes it as `placePose` on cycle state; the placeholder placement is marked
      as a stub a later section will replace
- [ ] `stop` / `start` / `restart` behave correctly across the longer cycle
- [ ] The arm runs three consecutive cycles without error, each kicked off by `restart`

## Takeaway

The full motion cycle runs. The arm moves through pickup, transit, place, lift, and return
without engaging the gripper. The motion path is complete; gripper coordination is the next
piece.
