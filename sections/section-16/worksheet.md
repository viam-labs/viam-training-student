# Section 16: Pick-station component and pickup states

Time: 65 minutes.

In Section 15 you replaced the sleep stubs with `moveToPose` calls, and tested with hardcoded coordinates.
In this section we will use the `pick-station` component to determine where the pickup location is and how 
to handle moving around the pick station. To coordinate these steps, the state machine will grow from two 
states (out-and-back) to five:

```
MOVING_TO_PICKUP_HOME → GRASPING → RETRACTING_NORMAL → RETRACTING_ALONG_CONVEYOR → RETURNING_TO_PICKUP_HOME
```

## What you will do in this section

- Configure the palletizer to interact with the pick-station over DoCommand
- Source the pickup-home and grasp poses from the pick-station via `get_pick_home_pose` and
  `get_vacuum_pose` DoCommand verbs
- Add the box dimensions (`box_width_mm`, `box_length_mm`, `box_height_mm`) as palletizer
  config attributes; use `box_height_mm` to query the pick-station and the box dimensions to
  size the two pickup retracts
- Grow the state machine by adding five pickup-side states wired into the state machine's
  handler map and the Web Application's state diagram

## How this section works

Each Part teaches a concept, you write the prompt yourself, you review the generated code
against a checklist, and then you verify behavior.

## Setup check

- Section 15 runs: `start` drives `IDLE → MOVING_OUT → MOVING_BACK → DONE` with calls to
  the motion service; `moveToPose` exists with three constraint styles.
- Resource graph green: `arm`, `gripper`, `floor`, `pallet`, `pick-station`,
  `workcell-scene`, `palletizer`.
- `make` builds cleanly

If the cell drifted, restore `prereq-machine-config.json` and reload.

**Cycle-state count (going in):** 3 working from §13 (`IDLE`, `MOVING_OUT`, `MOVING_BACK`) + 2 terminal. §16 replaces `MOVING_OUT` and `MOVING_BACK` with the **five** pickup states (`MOVING_TO_PICKUP_HOME`, `GRASPING`, `RETRACTING_NORMAL`, `RETRACTING_ALONG_CONVEYOR`, `RETURNING_TO_PICKUP_HOME`), so after this section the count is 6 working (`IDLE` plus the five pickup states) + 2 terminal.

---

## Part 1. Explore the pick-station's pose API (10 min)

### Concept

The `pick-station` knows where we should go to pickup our box, defines a home position for
the approach, the grasp pose, and the conveyor direction so we know which way to move so
we don't hit the other boxes. The palletizer will reach out to the `pick-station` for these
values instead of storing them itself. Each of the labels on the left has a matching
DoCommand verb on the right:

```
                ● pick-home pose                  get_pick_home_pose
                │
                │
                ▼
                ● vacuum / grasp pose             get_vacuum_pose
             +------+
             | box  |
             |      |  ───►                       get_conveyor_direction
             +------+
        ==================  conveyor surface
        ||              ||
        ||              ||
```

**pick-station attributes:**
```json
{
  "attributes": {
    "box_origin_offset_mm": { "x": 200, "y": 200 },
    "box_theta_deg": 0,
    "pick_home_z_offset_mm": 120
  },
}
```

- `box_origin_offset_mm` — The pick station may be a wide conveyor, but the box could be centered or shifted to a side wall. This specifies the X and Y position of the center of the box measured from the
  conveyor's origin corner.
- `box_theta_deg` — the box's rotation about Z on the surface. If the top of the box is rotated, we can account for it here.
- `pick_home_z_offset_mm` — how far above the box top the pick-home approach pose sits. We want to be far enough away the camera can see the box, but close enough we don't waste much travel time approaching the box.

The three DoCommand verbs in detail:

- **`get_pick_home_pose`** returns a world-frame pose describing where the arm should
  go before picking up from the pick station. We must provide a box height to get the
  correct Z offset.

  ```json
  { "get_pick_home_pose": { "box_height_mm": 85 } }
  ```

- **`get_vacuum_pose`** returns a world-frame pose describing where to pickup the box.
  We can consider this the `grasp pose`. We must provide a box height here as well to get
  the correct Z offset.

  ```json
  { "get_vacuum_pose": { "box_height_mm": 85 } }
  ```

- **`get_conveyor_direction`** returns a unit vector in the direction boxes flow along
  the conveyor. The move-along-conveyor move uses this vector.

  ```json
  { "get_conveyor_direction": true }
  ```

### Manual testing

Under the CONTROL tab in the Viam app, open the pick-station's component card and run each verb through DoCommand and record the values.

### Verify

- [ ] All three calls return values
- [ ] `get_pick_home_pose` returns a Z value approximately 120 mm above `get_vacuum_pose`
- [ ] `get_conveyor_direction` returns a vector of X/Y/Z values
- [ ] You can state in one sentence why the palletizer should query the pick-station for
      these values rather than store them itself

---

## Part 2. Tell the palletizing module about the pick-station (15 min)

### Concept

In Section 11 you added `arm` and `gripper` as required
Config attributes, returned them as dependencies from `Validate`, and resolved them out of
the `resource.Dependencies` map in the constructor. Pick-station follows the same pattern,
applied to a third component:

1. Add the new Config fields: `pick_station` (the component's configured name) and the three
   box-dimension values — `box_width_mm`, `box_length_mm`, `box_height_mm`. All required.
2. Have `Validate` return `pick_station` in the dependency list. The three box-dimension
   values are plain numbers, not dependencies — they do not go in that list.
3. In the constructor, resolve the pick-station handle out of `deps` and store it on the
   palletizer struct.

`box_height_mm` is used directly in this section — the pick-station's pose verbs take it as
an argument. `box_width_mm` and `box_length_mm` round out the box geometry: the cell knows
the box size up front for the held-box collision geometry the planner uses while a box is in
the gripper, and the box dimensions also size the two pickup retracts.

A word on those retracts. After the arm grasps a box, it can't head straight home — another
box is usually waiting right behind it on the conveyor, and sweeping the gripper sideways
through that box would knock it loose. So the pickup ends with two deliberate retracts: a
straight lift up off the box, then a straight shift along the conveyor flow direction to
carry the gripper clear of the next box before it returns home.

How far each retract travels comes straight from the box geometry you already entered, so
there's no separate attribute to define or keep in sync:

- the **normal retract** lifts `1.5 × box_height_mm` straight up off the grasp pose — enough
  to clear a box of that height with margin.
- the **along-conveyor retract** shifts `1.0 × box_length_mm` along the conveyor flow
  direction — about one box-length, enough to clear the trailing box.

Deriving the retracts from the box dimensions means a taller or longer box automatically
gets a proportionally bigger retract, with no extra knobs to tune.

One thing changes vs §11. `arm` and `gripper` had their own typed Go interfaces
(`arm.Arm`, `gripper.Gripper`) and you resolved them via `arm.FromDependencies` /
`gripper.FromDependencies`. The pick-station is configured as a `generic` component — it
has no custom Go interface; everything we ask it goes through `DoCommand`. So the resolved
handle is a plain `resource.Resource`, fetched via the generic **component** package's `Named`.

Your `module.go` already imports the generic **service**
package as `generic` — the scaffold added `go.viam.com/rdk/services/generic` for
`RegisterService`. The pick-station needs the generic **component** package,
`go.viam.com/rdk/components/generic`, which is *also* named `generic`, and Go won't allow two
imports under the same name. Leave the service import alone and alias the component one as
`gencomp`:

```go
import (
    gencomp "go.viam.com/rdk/components/generic"  // generic component — aliased to avoid the clash
    "go.viam.com/rdk/resource"
)
```

Your existing `RegisterService(generic.API, …)` stays as is. Resolve the pick-station with
`gencomp.Named`:

```go
pickStation, err := resource.FromDependencies[resource.Resource](
    deps, gencomp.Named(conf.PickStation),
)
if err != nil {
    return nil, fmt.Errorf("failed to get pick-station %q: %w", conf.PickStation, err)
}
p.pickStation = pickStation
```

Once the handle is stored, every interaction with the pick-station goes through
`DoCommand` — Part 3 will write the helpers that do that talking.

### Writing your prompt

To recap: we are adding four required Config attrs (`pick_station` plus the three box
dimensions `box_width_mm` / `box_length_mm` / `box_height_mm`), returning `pick_station`
as a dependency from `Validate`, and resolving the pick-station handle in the constructor.

Before you prompt, decide:

1. Where on the palletizer struct do the new Config fields live? Look at how `arm` and
   `gripper` are declared — same JSON-tag and validation pattern.
2. `Validate` returns a list of dependency names. Where does `pick_station` go?
   (Hint: only the component name is a dependency.)
3. How do you set the new attributes we are adding to the palletizer? (This is a manual
   step, outside of the Coding Agent.)

Now write and execute the prompt.

### Review the code

- [ ] `pick_station` field is on the Config struct with the JSON tag and is required in
      `Validate`
- [ ] `box_width_mm`, `box_length_mm`, and `box_height_mm` are each on the Config struct
      with JSON tags and each required in `Validate` (rejects zero)
- [ ] `pick_station` appears in the dependency list returned by `Validate`
- [ ] The resolved pick-station handle is stored on the palletizer struct in the `NewPalletizer` constructor
- [ ] All three box-dimension attributes are stored on the palletizer struct as `float64`
- [ ] Error messages identify which attribute is missing

### Manual config edit

In the app's CONFIGURE tab → JSON, add to the palletizer's attributes:
```json
"pick_station": "pick-station",
"box_width_mm": 282,
"box_length_mm": 240,
"box_height_mm": 85,
```

`pick_station` is wired as a dependency by `Validate` returning it (above), so there's nothing to add to `depends_on`.

### Verify

- [ ] `make` is clean
- [ ] `viam module reload` succeeds. If reload reports `Manifest not found at "module"` but
      your `meta.json` is present, this is a transient error — run the identical command again
      and it succeeds.
- [ ] The palletizer is green and the resource graph shows a new edge to `pick-station`
- [ ] Removing `pick_station` from the config causes `Validate` to reject it with a clear
      error; restoring it returns the resource to green
- [ ] Removing any of `box_width_mm`, `box_length_mm`, or `box_height_mm` causes
      `Validate` to reject the config with a clear error; restoring it returns the
      resource to green

---

## Part 3. Resolve pickup-home and grasp poses in code (15 min)

### Concept

Our palletizing module now has access to the pick-station. Next we are going to add two
helper methods that wrap the pick-station's verbs:

- `resolvePickupHomePose(ctx)` — calls `get_pick_home_pose` and returns the pickup-home
  pose as a `spatialmath.Pose`. The cycle will call this when it needs the safe waypoint
  above the box.
- `resolveGraspPose(ctx)` — calls `get_vacuum_pose` and returns the grasp pose as a
  `spatialmath.Pose`. The cycle will call this when it descends onto the box top.

Both belong in their own file. Let's create `pick_station.go` — the pick-station
integration is its own concern, the same way Section 15's motion code lives in
`motion.go`.

Let's take a look at what one of these helpers looks like:

```go
func (p *palletizer) resolvePickupHomePose(ctx context.Context) (spatialmath.Pose, error) {
    resp, err := p.pickStation.DoCommand(ctx, map[string]any{
        "get_pick_home_pose": map[string]any{"box_height_mm": p.cfg.BoxHeightMM},
    })
    if err != nil {
        return nil, fmt.Errorf("get_pick_home_pose: %w", err)
    }
    return pose6DMapToPose(resp), nil
}
```

`resolveGraspPose` follows the same pattern, just calling `get_vacuum_pose` instead.

You'll notice the helper hands the `DoCommand` response off to a `pose6DMapToPose`
function. The pick-station returns a pose as a flat map — `{"x":…, "y":…, "z":…,
"o_x":…, "o_y":…, "o_z":…, "theta":…}` — and the planner wants a `spatialmath.Pose`
value, so we need a small parser that bridges the two. Next we are going to write that
parser in the same file:

```go
func pose6DMapToPose(m map[string]interface{}) spatialmath.Pose {
    return spatialmath.NewPose(
        r3.Vector{X: asFloat(m["x"]), Y: asFloat(m["y"]), Z: asFloat(m["z"])},
        &spatialmath.OrientationVectorDegrees{
            OX: asFloat(m["o_x"]), OY: asFloat(m["o_y"]),
            OZ: asFloat(m["o_z"]), Theta: asFloat(m["theta"]),
        },
    )
}
```

`asFloat` is the tiny helper `pose6DMapToPose` uses to pull each number out of the map.
DoCommand numbers arrive as `float64` over gRPC, so a type assertion is enough:

```go
func asFloat(v any) float64 {
    f, _ := v.(float64)
    return f
}
```

Both helper methods and the two small parser functions all go in `pick_station.go`.

### Writing your prompt

To recap: we are creating `pick_station.go` with two helpers (`resolvePickupHomePose` and
`resolveGraspPose`) and the two small parser functions (`pose6DMapToPose` and `asFloat`)
they share. All four are shown above. That leaves a couple of design decisions:

1. Both helpers return `(spatialmath.Pose, error)`. What should each do when `DoCommand`
   returns an error — return the error, or swallow it and return a zero pose? (A silent
   zero pose sends the arm to the world origin, which could be dangerous.)
2. The grasp and pickup-home verbs both need `box_height_mm`. Where does that value come
   from — a literal in the helper, or the field you stored on the struct in Part 2?

### Review the code

- [ ] Both helpers take `ctx context.Context` and return `(spatialmath.Pose, error)`
- [ ] Both pass `box_height_mm` from the palletizer struct, not a magic number
- [ ] The DoCommand argument is correctly nested under the verb key
- [ ] Errors from DoCommand or pose parsing return an error, not a zero pose

### Verify

- [ ] `make` is clean
- [ ] (Optional) Add a temporary log line at construction that calls both helpers and logs
      the results. Confirm they match the values recorded in Part 1
- [ ] Remove the temporary log lines before moving on

---

## Part 4. Grow the cycle with five pickup states (25 min)

### Concept

So far the cycle just moves out and back. With the helpers from Part 3 we can finally
make it move toward and around the pick-station for real. The pickup half of a real
palletizing cycle is a sequence of moves: approach a safe waypoint above the box,
descend onto the top, retract straight up, slide along the conveyor so the gripper is
clear of the next box waiting behind it, then return to the safe waypoint. Each one of
those is a distinct state in the machine, so the operator can see where the arm is at
any moment, so the constraints can differ per move (the descent is orientation-locked,
the retracts are linear), and so `stop` parks cleanly mid-sequence.

All five replace `MOVING_OUT` and `MOVING_BACK`. Here's the full cycle:

```
IDLE
 → MOVING_TO_PICKUP_HOME        ← NEW: approach the safe waypoint above the box
 → GRASPING                     ← NEW: orientation-locked descent onto the box top
 → RETRACTING_NORMAL            ← NEW: linear lift straight up off the box
 → RETRACTING_ALONG_CONVEYOR    ← NEW: linear shift along the conveyor, clear of the next box
 → RETURNING_TO_PICKUP_HOME     ← NEW: back to the safe waypoint
 → DONE  (terminal — restart for the next cycle)
```

Let's look at the new states in detail:

- **`MOVING_TO_PICKUP_HOME`** — `moveToPose(resolvePickupHomePose())` to reach the safe
  waypoint.
- **`GRASPING`** — `moveToPose(resolveGraspPose(), moveOrientationLocked)`. The descent onto
  the box top. No vacuum actuation in this section.
- **`RETRACTING_NORMAL`** — `moveLinear` straight up by `1.5 * p.cfg.BoxHeightMM`. The
  descent in `GRASPING` left the gripper at the grasp pose, but don't cache and reuse
  that — instead, read the *current* gripper pose at state entry via
  `p.motion.GetPose(ctx, p.cfg.Gripper, referenceframe.World, nil, nil)`. The component
  name is the plain `string` you stored in Part 2 (`p.cfg.Gripper`), not a `resource.Name`.
  `GetPose` returns a `*referenceframe.PoseInFrame`, so call `.Pose()` on the result to get
  the `spatialmath.Pose`; then add `1.5 * p.cfg.BoxHeightMM`
  to its Z, and plan the linear move from there. Reading the current pose keeps the
  retract correct regardless of any state-machine entries between the descent and the
  retract.
- **`RETRACTING_ALONG_CONVEYOR`** — `moveLinear` along the conveyor flow direction
  (queried from `pick-station.get_conveyor_direction`) by `p.cfg.BoxLengthMM`,
  to clear the next box on the conveyor. `get_conveyor_direction` returns a flat
  `{"x":…, "y":…, "z":…}` unit-vector map (not a 6D pose), so pull the three components
  with the same `asFloat` helper into an `r3.Vector`. Same current-pose pattern: read the
  gripper's current pose at state entry, scale the unit vector by `p.cfg.BoxLengthMM`,
  and plan the linear move.
- **`RETURNING_TO_PICKUP_HOME`** — `moveToPose(resolvePickupHomePose())` to return to the
  safe waypoint.

### Writing your prompts

Don't try to land all five states in one prompt. That's a large change to review at once,
and if the cycle misbehaves you won't know which state to blame. Build the pickup half
**one state at a time**: prompt for a single state, hot-reload, watch it appear in the 3D
scene and the state diagram, then move to the next. The state-by-state list above is your
spec — one entry per prompt.

Each state is the same structural change: a new state constant, a handler method, a
dispatcher entry in the handler map (from Section 13), and an `OnTransition` log line. So
each prompt names one state, its constraint style, and what its handler does. Work in
cycle order:

1. **`MOVING_TO_PICKUP_HOME`** — `moveToPose(resolvePickupHomePose())` to the safe waypoint
   above the box.
2. **`GRASPING`** — orientation-locked `moveToPose(resolveGraspPose())` descent onto the
   box top. No vacuum actuation yet.
3. **`RETRACTING_NORMAL`** — linear lift straight up by `1.5 * p.cfg.BoxHeightMM`. Read the
   *current* gripper pose at entry, add to its Z, and plan from there — don't reuse the
   grasp pose.
4. **`RETRACTING_ALONG_CONVEYOR`** — linear shift by `p.cfg.BoxLengthMM` along the
   conveyor direction. Query `get_conveyor_direction` at runtime (so it tracks the
   pick-station's config rather than going stale), read the current pose, scale the unit
   vector, and plan the linear move.
5. **`RETURNING_TO_PICKUP_HOME`** — `moveToPose(resolvePickupHomePose())` back to the safe
   waypoint.

Then the wiring that ties them together:

6. Remove `MOVING_OUT` and `MOVING_BACK` entirely — their constants, handlers, and
   dispatcher entries. Leaving them defined but unreferenced just clutters the state
   diagram. The old handlers built poses from `r3.Vector{...}` literals; the new pickup
   handlers use `cur.Point()` and `dir.Mul(...)` and never name `r3` directly. Once the old
   handlers are gone, `github.com/golang/geo/r3` is no longer referenced in
   `state_machine.go`, and `make` fails with `imported and not used` until you remove that
   import.
7. Add the five new nodes to the Web Application's state diagram (and drop the two old
   ones). Where is that list, and what is the minimal edit?

### Review the code

- [ ] Five new state constants with descriptive names
- [ ] Each state has a handler method matching the existing pattern
- [ ] The handler map in `newCycleMachine` includes the five new states; `MOVING_OUT` and
      `MOVING_BACK` are removed
- [ ] `OnTransition` logging covers the new states
- [ ] `GRASPING`'s `moveToPose` uses `moveOrientationLocked` and does not call
      `gripper.Grab`
- [ ] `RETRACTING_NORMAL` and `RETRACTING_ALONG_CONVEYOR` use `moveLinear`
- [ ] The retract distances are derived from the box dimensions, not hardcoded numbers:
      `RETRACTING_NORMAL` lifts by `1.5 * p.cfg.BoxHeightMM` and `RETRACTING_ALONG_CONVEYOR`
      shifts by `p.cfg.BoxLengthMM`
- [ ] Both retract states read the *current* gripper pose at state entry via `GetPose`
      (calling `.Pose()` on the result), not a cached grasp pose
- [ ] `RETRACTING_ALONG_CONVEYOR` queries `pick-station.get_conveyor_direction` rather
      than hardcoding the direction or storing it on the palletizer
- [ ] The Web Application's state diagram contains the five new nodes; `MOVING_OUT` and
      `MOVING_BACK` are removed

### Verify

With these five states wired in, the pickup half of the cycle is real motion driven by
the pick-station's geometry: the arm approaches the safe waypoint, descends to the
grasp pose, retracts up off the box, shifts along the conveyor to clear the next box,
then returns to the safe waypoint. The gripper itself is not engaged — but the per-state
sequencing and the trajectory are end-to-end.

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] `start` drives the arm through all five states in order
- [ ] `status` reports each state as the cycle progresses
- [ ] The Web Application's state diagram highlights each state as it runs
- [ ] `stop` mid-state parks the cycle on the current state
- [ ] `start` after `stop` resumes from the parked state
- [ ] `restart` resets to `IDLE`
- [ ] On the second cycle, `MOVING_TO_PICKUP_HOME` produces no visible motion because the
      arm is already at pickup-home. This is correct: `RETURNING_TO_PICKUP_HOME` and
      `MOVING_TO_PICKUP_HOME` target the same pose.

---

## Done when

You can answer **yes** to all of these:

- [ ] The palletizer depends on `pick_station` and rejects a config that omits it
- [ ] `box_width_mm`, `box_length_mm`, and `box_height_mm` are required palletizer
      attributes; `box_height_mm` is used everywhere the pick-station's pose queries need
      it
- [ ] The cycle queries pick-station for its pickup-home pose, grasp pose, and conveyor
      direction; no hardcoded coordinates remain
- [ ] All five new states are observable via `status` and the Web Application; `MOVING_OUT`
      and `MOVING_BACK` are removed
- [ ] The retracts use `moveLinear`; `RETRACTING_ALONG_CONVEYOR` reads the conveyor
      direction from the pick-station
- [ ] `stop` / `start` / `restart` behave correctly across the longer cycle

## Takeaway

The pickup half of the cycle is now component-driven: the pick-station holds the geometry,
the palletizer queries it. No gripper actuation yet; the cycle moves the arm through the
descent and the retracts without engaging the gripper.
