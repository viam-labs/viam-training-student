# Section 19: Pack-sequencer service

Time: approximately 115 minutes.

At the end of Section 18 the palletizer can pick up a box, carry it across, and release it
at the pallet, but the cycle releases the box at the same hardcoded pose. There is no
concept of pack order, no awareness of boxes already on the pallet, and the 3D scene viewer
has no representation of what the gripper is holding.

This section installs the pack-sequencer service, a Viam service that helps determine the
order and locations boxes will be placed on a pallet, tracks how full the pallet is, and
adds a visual of the boxes.

## What you will do in this section

- Install and configure the `viam:pack-sequencer:sequencer` service
- Call the pack-sequencer verbs (`next_box`, `report_placement`, `get_box_dims`,
  `get_status`, `get_pack_order`, `set_box_visual`, `reset_progress`)
  through the pack-sequencer's `contracts` client helpers
- Update the place pose using the pack-sequencer
- Report success and failure to the pack-sequencer so its box index advances or retries
- Show each box in the 3D scene — on the pick station, attached to the gripper, then placed on
  the pallet — by emitting `set_box_visual` updates
- Feed the placed boxes (obstacles) and the held box (a gripper-attached transform) into the
  planner's world state
- Stop the cycle cleanly when the pallet is complete

## Setup check

- Section 18 runs end-to-end: the cycle picks up at pickup-home and releases at the pallet;
  `gripper.IsHoldingSomething` transitions to `true` at
  `ENABLING_VACUUM` entry and to `false` at `RELEASING_AT_PALLET` entry.
- The palletizer has the following Config attrs: `arm`, `gripper`, `pick_station`,
  `pallet`, `box_width_mm`, `box_length_mm`, `box_height_mm`, `retract_normal_mm`,
  `retract_along_conveyor_mm`, `safety_height_mm`, `place_dwell_secs`, and the three
  suction timing attrs. Resource graph green.
- `viam module reload --part-id <PART_ID>` reinstalls cleanly.

If the cell drifted, restore `prereq-machine-config.json` and reload.

**Cycle-state count (going in):** 13 working from §18 + 2 terminal (`DONE`, `ERROR`).

---

## Part 1. Install the pack-sequencer service (15 min)

### Concept

The pack-sequencer is a Viam service that helps pack boxes on a pallet. It uses DoCommand
verbs and attributes to set the box dimensions and pallet stack height, keeps track of what's
been stacked, and provides a way to visualize the boxes as they move around the 3D scene.
Let's start by installing the pack-sequencer service on the machine.

![Pack order — one layer, boxes numbered in placement order](pack-order.svg)

**Installing the pack-sequencer**

In the Viam app's CONFIGURE tab, click **+ Configuration Block**, search `pack-sequencer`,
and pick the **viam:pack-sequencer:sequencer** model. Click **add to machine** and name it
`pack-sequencer`.

Next we need to set the attributes to configure how the pallet is packed. Set these in the
pack-sequencer's configuration. We will also look at what each attribute does.

```json
{
  "pallet": "pallet",
  "box_width_mm": 282,
  "box_length_mm": 240,
  "box_height_mm": 85,
  "pallet_area_height_mm": 200,
  "quantity": 8,
  "box_offset_x_mm": 10,
  "box_offset_y_mm": 10,
  "box_color": {"r": 200, "g": 150, "b": 100},
  "place_orientation": {"o_x": 0, "o_y": 0, "o_z": -1, "theta": 0}
}
```

What each attribute sets:

| Attribute | What it sets |
|---|---|
| `pallet` | The pallet component's name. The pack-sequencer reads the pallet's live pose and dimensions from it. |
| `box_width_mm`, `box_length_mm`, `box_height_mm` | The box dimensions. |
| `pallet_area_height_mm` | The stacking ceiling — the maximum height above the pallet surface that boxes may be stacked to. Independent of the pallet's own thickness. |
| `quantity` | How many boxes are in the pack. |
| `box_offset_x_mm`, `box_offset_y_mm` | Extra gap between adjacent boxes along the pallet's X and Y (10 mm here). A small gap keeps the planner from flagging touching boxes as colliding and gives a descending box room to settle. |
| `box_color` | The box color in the 3D scene (defaults to cardboard brown if unset). |
| `place_orientation` | The orientation boxes are placed with — here `o_z: -1`, facing straight down. |

Save the config. The pack-sequencer should come up green.

### Verify

- [ ] Pack-sequencer is in the cell, green in the resource graph

---

## Part 2. Add the pack-sequencer as a dependency (5 min)

### Concept

Now that the pack-sequencer service is installed on the machine, the palletizer needs to access it
from within the Go module. The pack-sequencer is a **World State service**, not a component. You
resolve it the same way you resolved the §16/§17 components — declare it as a dependency, look it
up in the constructor — with one change: this service uses `worldstatestore.Named`:

```go
worldstatestore.Named(conf.PackSequencer) // go.viam.com/rdk/services/worldstatestore
```

### Writing your prompt

- How did we add the pallet and pick-station dependencies? What is different about the
  Pack-Sequencer type?
- Do we need to display an error if we can't find the Pack Sequencer?
- Once we get the resource where should we store it?

*If you get stuck:*

- [ ] add a required `pack_sequencer` string Config attr; `Validate` rejects it when empty
- [ ] resolve it in the constructor with
      `resource.FromDependencies[worldstatestore.Service](deps, worldstatestore.Named(conf.PackSequencer))` —
      the same pattern as §16's pick-station, returning an error if it's missing
- [ ] store the resolved service for now

### Review the code

- [ ] `pack_sequencer` is a required Config attr; the service is resolved via
      `worldstatestore.Named(...)` as a `worldstatestore.Service` (not `generic.Named`).

### Verify

- [ ] `make` is clean
- [ ] `viam module reload` brings the palletizer up green with the pack-sequencer wired in

---

## Part 3. Explore the pack-sequencer API (10 min)

### Concept

Previously we wrapped `DoCommand` calls in helper methods. For a service you use often, it can be
worth pulling those wrappers into a library, so you write them once and share them across the
components and services that call it. The pack-sequencer publishes exactly that — a small
`contracts` library that wraps its own `DoCommand` verbs — so you call a typed method instead of
assembling a request map by hand.

Add the import (the dependency comes with it), then bind the service you resolved in Part 2 to a
client **once** and store that on the palletizer struct — from here on you pass the client
around instead of the raw service:

```go
import "github.com/viam-labs/pack-sequencer/contracts"

// in the constructor, after resolving the worldstatestore.Service into svc:
p.packSequencer = contracts.NewClient(svc)
```

Each verb is then a method on the client (`ctx` is the only plumbing it needs):

| Call | Returns |
|------|---------|
| `p.packSequencer.NextBox(ctx)` | `NextBoxResponse` — the next box's place pose and dimensions |
| `p.packSequencer.GetStatus(ctx)` | `StatusResponse` — pack progress: which boxes are placed/skipped/failed, the counts, and `complete` |
| `p.packSequencer.GetBoxDims(ctx)` | `GetBoxDimsResponse` — the pack's box dimensions (`BoxLengthMM`, `BoxWidthMM`, `BoxHeightMM`) |
| `p.packSequencer.GetPackOrder(ctx)` | `GetPackOrderResponse` — the full pack order (every box's pose and dimensions) plus pallet info |
| `p.packSequencer.ReportPlacement(ctx, req)` | `ReportPlacementResponse` — updated progress after a placement; `req` is `ReportPlacementRequest{Seq, Success, Error}` |
| `p.packSequencer.ReportSuccess(ctx)` / `ReportFailure(ctx, reason)` | `ReportPlacementResponse` — shorthands for `ReportPlacement` on the current box (no seq) |
| `p.packSequencer.SetBoxVisual(ctx, req)` | `SetBoxVisualResponse` — adds or moves a box in the 3D scene |
| `p.packSequencer.ResetProgress(ctx)` | error only — clears progress back to the first box |

Each method returns a typed struct we have not seen before. `GetBoxDims` returns a
`GetBoxDimsResponse`, `NextBox` returns a `NextBoxResponse`, and so on. Before we start working with
these methods, let's find out how to inspect the return types. There are two options: the `go doc`
cli or the IDE. Run this in your terminal to use `go doc`:

```bash
go doc github.com/viam-labs/pack-sequencer/contracts GetBoxDimsResponse
```

It prints the struct and its fields:

```go
type GetBoxDimsResponse struct {
    BoxLengthMM float64 `json:"box_length_mm"`
    BoxWidthMM  float64 `json:"box_width_mm"`
    BoxHeightMM float64 `json:"box_height_mm"`
}
```

**What does each method return?** Run `go doc` on each method's response type and write down its
fields:

- What does `NextBox` return?
- What does `GetStatus` return?
- What does `GetBoxDims` return?
- What does `ReportPlacement` return?

---

## Part 4. Pull box dimensions from the pack-sequencer (5 min)

### Concept

Up until now we have relied on the palletizer's own attributes for the box dimensions (L x W x H); the pack-sequencer also keeps track of the box dimensions, and it can get messy to have multiple sources of truth. Next, let's get the box dimensions from the pack-sequencer instead of the attributes we made earlier. Let's take a look at the `GetBoxDims` method and the `GetBoxDimsResponse` again:

```go
dims, err := p.packSequencer.GetBoxDims(ctx)
if err != nil {
    return err
}
fmt.Println("box width:", dims.BoxWidthMM)
fmt.Println("box length:", dims.BoxLengthMM)
fmt.Println("box height:", dims.BoxHeightMM)
```

### Writing your prompt

We want the pack-sequencer to be the source of truth for the box dimensions. This means we want to remove the box attributes from the palletizer and instead ask the pack-sequencer what size boxes are we expecting. Work through the following questions while you form your prompt.

- How has the palletizer kept track of those fields until now? 
- What method should we use and how do we get the data from the response type?
- Do we still need the same `Validate` checks we made previously? 
- Should we add a Log statement to check the dimensions have been set correctly?

*If you get stuck:*

- [ ] Remember we have `box_width_mm`/`box_length_mm`/`box_height_mm` in our Config struct, these come from attributes and are checked in `Validate`. We can now get these from the pack-sequencer's `contracts` client.
- [ ] Once we've resolved the pack-sequencer service (Part 2) and wrapped it in `contracts.NewClient(svc)` (Part 3), we can call `GetBoxDims` on that client, very similar to the example above. 

### Review the code

- [ ] `box_width_mm`/`box_length_mm`/`box_height_mm` are no longer user-supplied attributes
      validated in `Validate`; the constructor calls `contracts.GetBoxDims` and writes the result
      into the same Config fields (`p.cfg.BoxWidthMM`/`BoxLengthMM`/`BoxHeightMM`), so the
      pack-sequencer is the single source of truth and later code still reads `p.cfg.BoxHeightMM`.
- [ ] All pack-sequencer calls go through the `contracts` client helpers, not hand-rolled
      `DoCommand` + `map[string]any` access.

### Verify

- [ ] `make` is clean
- [ ] `viam module reload` succeeds
- [ ] the LOGS show the box dimensions pulled from the pack-sequencer at construction

---

## Part 5. Split the place state and drive it from `next_box` (15 min)

### Concept

§17 placed every box at the same hardcoded pose (`pallet.get_top_face_center`) with a single `MOVING_TO_PLACE` state — one straight linear descent onto `placePose`. Now the pack-sequencer is going to take care of this. First let's look at `NextBoxResponse`, the return type for `next_box`:

```go
type NextBoxResponse struct {
    Seq                int           `json:"seq"`
    Col                int           `json:"col"`
    Row                int           `json:"row"`
    Layer              int           `json:"layer"`
    PlaceStartInWorld  Pose6D        `json:"place_start_in_world"`
    PlaceEndInWorld    Pose6D        `json:"place_end_in_world"`
    PlaceStartInPallet Pose6D        `json:"place_start_in_pallet"`
    PlaceEndInPallet   Pose6D        `json:"place_end_in_pallet"`
    BoxDimensionsMM    BoxDimensions `json:"box_dimensions_mm"`
    IsComplete         bool          `json:"is_complete"`
}
```

The place move is a **two-pose trajectory**, not a single drop. `PlaceEnd` is the final slot —
where the box is set down. `PlaceStart` is offset up and over from it, so the box descends
*diagonally* into the slot and clears the boxes already on the pallet instead of dragging across
them. The arm needs to move to `PlaceStart`, then descend to `PlaceEnd`.

![The place move descends from PlaceStart diagonally into PlaceEnd, clearing the placed neighbour](place-trajectory.svg)

Two poses means two moves, so §17's single `MOVING_TO_PLACE` splits in two: **`MOVING_TO_PLACE_START`**
moves the arm to `PlaceStart` (above and over the slot), then **`MOVING_TO_PLACE_END`** runs the
linear descent into `PlaceEnd`. That's one new working state — the cycle goes from 13 to 14.

Each pose comes in two frames. The **`*InWorld`** pair is pre-composed with the pallet's world
pose — hand those straight to the motion service. The **`*InPallet`** pair is the same two poses
in the pallet's local frame, for visualization (not needed for motion). Let's look at the fields
in detail:

| Field | Type | What it is |
|-------|------|-----------|
| `Seq` | `int` | the box's index in the pack order |
| `Col`, `Row`, `Layer` | `int` | its slot on the pallet |
| **`PlaceStartInWorld`** | `Pose6D` | **where the place move begins** — world-frame, offset above and over the slot |
| **`PlaceEndInWorld`** | `Pose6D` | **where the box is set down** — world-frame, the final slot the arm descends to |
| `PlaceStartInPallet`, `PlaceEndInPallet` | `Pose6D` | the same two poses in the pallet's local frame (for visualization, not motion) |
| `BoxDimensionsMM` | `BoxDimensions` | this box's dimensions |
| `IsComplete` | `bool` | `true` once every box is placed |

Let's look at an example real-world response. `seq` is **1-based** — after `reset_progress` the
first box returns `seq: 1`, and the pack runs `1`…`8`:

```json
{
  "seq": 1,
  "col": 0,
  "row": 0,
  "layer": 0,
  "place_start_in_world":  {"x": 224, "y": 500, "z": 360, "o_x": 0, "o_y": 0, "o_z": -1, "theta": 0},
  "place_end_in_world":    {"x": 200, "y": 500, "z": 260, "o_x": 0, "o_y": 0, "o_z": -1, "theta": 0},
  "place_start_in_pallet": {"x": 124, "y": 75,  "z": 160, "o_x": 0, "o_y": 0, "o_z": -1, "theta": 0},
  "place_end_in_pallet":   {"x": 100, "y": 75,  "z": 60,  "o_x": 0, "o_y": 0, "o_z": -1, "theta": 0},
  "box_dimensions_mm":     {"width": 200, "length": 150, "height": 60},
  "is_complete": false
}
```

### Writing your prompt

Now that we've been introduced to the pack-sequencer, the contracts library, and the helper
methods, let's make a prompt to use the pack-sequencer's `NextBox` to control where the box goes.
Remember that we need to use both the start and end places to get the right motion. Let's work
through a few questions to help you:

- How do you split `MOVING_TO_PLACE` into `MOVING_TO_PLACE_START` and `MOVING_TO_PLACE_END`? Why
  are we splitting them?
- What type is `PlaceStartInWorld`, and what type does the `moveToPose` helper take? What has to
  happen between the two?
- What type of constraint should we use for the descent into the slot?
- What constraint did we use for `MOVING_TO_PLACE`?
- Do we need to update the state machine diagram in the Web Application?

*If you get stuck:*

- [ ] Remember that we want to split `MOVING_TO_PLACE`, so we'll end up removing the move to
      `pallet.get_top_face_center`.
- [ ] We will need two states where there was one — `MOVING_TO_PLACE_START` moves to `NextBox`'s
      `PlaceStartInWorld` (`moveLinear`), and `MOVING_TO_PLACE_END` descends to its
      `PlaceEndInWorld` (`moveLinear`, the angled descent); remember to add the new state to the
      state list.
- [ ] `PlaceStartInWorld`/`PlaceEndInWorld` are `contracts.Pose6D` (a plain wire struct, no rdk
      dependency), and `moveToPose` takes a `spatialmath.Pose`. Convert at the edge with a small
      helper — build `spatialmath.NewPose(r3.Vector{X, Y, Z}, &spatialmath.OrientationVectorDegrees{OX, OY, OZ, Theta})`
      (`Theta` is in degrees, matching `OrientationVectorDegrees`).

### Review the code

- [ ] `MOVING_TO_PLACE_START` is declared as a new state constant, added to the state list, and
      has a handler, a dispatcher entry, and an `OnTransition` log line — the same per-state
      pattern as §16/§17. The cycle goes from 13 to 14 working states.
- [ ] The Web Application's state diagram includes the new `MOVING_TO_PLACE_START` node.
- [ ] `MOVING_TO_PLACE_START` moves to `PlaceStartInWorld` and `MOVING_TO_PLACE_END` descends to
      `PlaceEndInWorld`, both `moveLinear`; each resolves its pose from `NextBox`, and the single
      `MOVING_TO_PLACE` / `placePose` / `get_top_face_center` path is gone.

### Verify

- [ ] `make` is clean and `viam module reload` succeeds.
- [ ] `start` runs the full cycle end-to-end; `MOVING_TO_PLACE_START` highlights in the state
      diagram in order (between `MOVING_TO_PALLET_HOME` and `MOVING_TO_PLACE_END`), and `status`
      reports it by name.
- [ ] In the 3D scene the place is now a two-step angled approach: the gripper arrives above and
      over the slot, then descends *diagonally* into it — not the straight-down §17 drop.

---

## Part 6. Report placements (10 min)

### Concept

The pack-sequencer expects a report after each cycle. The contracts package provides
`ReportPlacement`, plus two convenience wrappers we'll use — `ReportSuccess` and
`ReportFailure`. A report is one of two outcomes:

- **Success** — the box went down cleanly. The box index advances, and the pack-sequencer records
  the placed box's pose so later calls can return it as an obstacle.
- **Failure** — the box could not be placed. The box index does **not** advance; the next `NextBox`
  returns the same box so the cycle can retry it.

**Success.** Report it once the box is down and the arm has cleared it. `RETURNING_TO_PICKUP_HOME`
is an ideal time — by that state, the box is on the pallet and the arm has moved away:

```go
resp, err := p.packSequencer.ReportSuccess(ctx)
```

**Failure.** When a state can't complete — a move won't plan, a grasp keeps failing — the state
machine ends in the `ERROR` state and `Run` returns that error. That is where you report failure,
passing the error text as the reason so the pack-sequencer records it and holds the box for a
retry:

```go
// err is the cycle error returned by Run
resp, reportErr := p.packSequencer.ReportFailure(ctx, err.Error())
```

Let's take a look at the response from the Report methods. Both return a `ReportPlacementResponse`
— the pack-sequencer's progress snapshot after it records the report:

```go
type ReportPlacementResponse struct {
    Acknowledged bool   `json:"acknowledged"`
    NextBoxIndex int    `json:"next_box_index"`
    Placed       int    `json:"placed"`
    Failed       int    `json:"failed"`
    Skipped      int    `json:"skipped"`
    Remaining    int    `json:"remaining"`
    Complete     bool   `json:"complete"`
    LastError    string `json:"last_error,omitempty"`
}
```

### Writing your prompt

We need the palletizer to report each placement to the pack-sequencer — `ReportSuccess` once the
box is safely down and the arm has cleared it, and `ReportFailure` when the cycle errors out.
Work through these questions as you form your prompt:

- Which state means the box is down **and** the arm has cleared it, so `ReportSuccess` is safe to
  call? Why not `RELEASING_AT_PALLET`?
- A failed cycle ends in the `ERROR` state, where `Run` returns the error. Where do you catch that
  and call `ReportFailure`, and what do you pass as the reason?
- How many times should a single cycle report?

### Review the code

- [ ] On success, the cycle calls `p.packSequencer.ReportSuccess(ctx)` once, at the
      start of `RETURNING_TO_PICKUP_HOME` (the box is down and the arm has cleared it).
- [ ] On failure, when `Run` returns an error (the cycle ended in `ERROR`), the cycle calls
      `p.packSequencer.ReportFailure(ctx, err.Error())`.
- [ ] Exactly one report per cycle (success or failure), not scattered inside state methods.

### Verify

- [ ] `make` is clean; `viam module reload` succeeds
- [ ] Run two cycles. The second cycle places at the second corner of layer 1, not the
      first
- [ ] From the pack-sequencer's CONTROL pane, `get_status` shows `placed: 2`

---

## Part 7. PALLET_COMPLETE and reset_progress (10 min)

### Concept

We have two issues with our palletizer:

1. After each box the cycle stops at `DONE` — it never moves on to the next box, and it has
   no way to tell when the pallet is full.
2. The box index doesn't restart when we restart the state machine.

Let's start with the cycle stopping after every box. Right now it ends after
`RETURNING_TO_PICKUP_HOME` transitions to `DONE`, a terminal state, so the state machine
halts and waits for an operator to start it again. That's really two gaps in one: the cycle
won't move on to the next box on its own, and nothing tells us when the pallet is finally
full.

We already report every placement back to the pack-sequencer (Part 6), and the
`ReportPlacementResponse` we get from `ReportSuccess` carries a `Complete` flag. When the
pallet is full that flag comes back true, so we can transition to
a new terminal state, `PALLET_COMPLETE`. When it's still false there are more boxes to pack,
so we loop back to `MOVING_TO_PICKUP_HOME` and run the cycle again.

The new `PALLET_COMPLETE` state is terminal — its state method just logs and returns itself,
so the state machine parks there until an operator resets:

```go
func (p *palletizer) statePalletComplete(_ context.Context) (State, error) {
    p.logger.Info("pallet complete; halted until reset")
    return statePalletComplete, nil
}
```

And finally let's take a look at handling resets. The operator's Reset button resets the
palletizer's state machine, but the pack-sequencer keeps its own box index. So the `reset`
DoCommand handler (from Section 13) must reset both — add a
`p.packSequencer.ResetProgress(ctx)` call to the same handler, alongside the
existing state-machine reset.

### Writing your prompt

We are making the state machine loop, adding a terminal `PALLET_COMPLETE`, and resetting the
pack-sequencer alongside the state machine. Work through these questions as you form your
prompt:

- What is the name of our new state?
- Where and how should you decide to transition to the new state or start the cycle over?
- How and where do we call reset on the pack-sequencer?

Remember to add the new state to the Web Application's state diagram so it shows up when the
cycle reaches it.

### Review the code

- [ ] There is a new `PALLET_COMPLETE` state that is terminal (it doesn't transition to
      another state)
- [ ] `RETURNING_TO_PICKUP_HOME` has two possible transitions, depending on the `Complete` flag
- [ ] The `reset` DoCommand handler calls `p.packSequencer.ResetProgress(ctx)`
      in the same handler as the state-machine reset
- [ ] The Web Application's state diagram includes `PALLET_COMPLETE`

### Verify

- [ ] Reset the pack-sequencer and confirm it places at the first position
- [ ] Run the state machine and confirm it continues for more than one cycle
- [ ] Reset the state machine after a few runs and confirm it starts over with the first box
- [ ] Run the full 8 boxes and confirm the state machine transitions to `PALLET_COMPLETE`;
      Reset and ensure the state machine starts over from the beginning
- [ ] In the Web Application, ensure `PALLET_COMPLETE` is visible and lights up when the pack
      is complete

---

## Part 8. 3D scene visuals (15 min)

### Concept

Up until now we have seen the arm move, but we haven't had a visual for the boxes themselves as
they're placed and the pallet stacks up. Next we are going to display boxes in the 3D scene.
Visuals in Viam can either be static — typically defined using the `geometry` tag as part of a
component's JSON — or dynamic, handled by the world state store service. For this application we
are going to use the pack-sequencer's `contracts.SetBoxVisual` method. We want a visual for every
box in the scene; the box should start on the pick station, attach to the arm once the vacuum
gripper grasps it, and disconnect from the arm once the vacuum gripper releases it. Let's look in
detail at these three visual events:

**1. Adding a box to the Pick Station.** We want a box to appear every time we start the
palletizer cycle. `MOVING_TO_PICKUP_HOME` is our first state, so it seems like a logical place
to call `SetBoxVisual`. Every visual is published under the box's `seq` — its sequence identity in the
pack — and we read that from `NextBox`, which returns the current box without advancing to the
next. Let's take a look at how we use `SetBoxVisual`:

**`MOVING_TO_PICKUP_HOME`**

```go
box, _ := p.packSequencer.NextBox(ctx)
pt := graspPose.Point()
ov := graspPose.Orientation().OrientationVectorDegrees()
p.packSequencer.SetBoxVisual(ctx, contracts.SetBoxVisualRequest{
    Seq:    box.Seq,
    Parent: "world",
    Pose: contracts.Pose6D{
        X: pt.X, Y: pt.Y,
        Z: pt.Z - p.cfg.BoxHeightMM/2, // grasp pose is the box's top face; drop to its center
        OX: ov.OX, OY: ov.OY, OZ: ov.OZ, Theta: ov.Theta,
    },
})
```

A few things to note in that call. `Pose6D.Theta` is in degrees, which is why we read the
orientation with `OrientationVectorDegrees()` (also degrees) — the units line up, so the values
copy across directly. Publishing another visual
under the same `seq` later moves this same box instead of adding a new one. `Parent: "world"`
places the box at an absolute pose in the scene. And the `Pose` is the box's *center*, but the
grasp pose sits on the box's top face, so we drop `Z` by half the box height to line them up.

**2. Attached to the arm.** A box only belongs on the gripper once the arm has actually picked
it up, so emit this visual *after* the grasp motion completes — calling it earlier would show
the box stuck to the gripper before the arm has reached it. Once the box is in hand (during
`ENABLING_VACUUM`), call `SetBoxVisual` again with the same sequence number to update its pose
and parent frame. By changing the parent frame to the gripper, the box becomes attached to the
gripper, and the `Pose` is the transform from the gripper frame to the box center:

**`ENABLING_VACUUM`**

```go
box, _ := p.packSequencer.NextBox(ctx)
p.packSequencer.SetBoxVisual(ctx, contracts.SetBoxVisualRequest{
    Seq:    box.Seq,
    Parent: p.cfg.Gripper, // the gripper component's name, e.g. "gripper"
    Pose:   contracts.Pose6D{Z: p.cfg.BoxHeightMM / 2, OZ: 1}, // gripper frame -> box center
})
```

**3. Released onto the pallet.** The moment the gripper opens (`RELEASING_AT_PALLET`),
re-publish the same box's `seq` back in the world frame at the final place pose, so the box
appears to land on the pallet immediately. `NextBox` gives us both the `seq` and the place pose
(`PlaceEndInWorld`):

**`RELEASING_AT_PALLET`**

```go
box, _ := p.packSequencer.NextBox(ctx)
end := box.PlaceEndInWorld
p.packSequencer.SetBoxVisual(ctx, contracts.SetBoxVisualRequest{
    Seq:    box.Seq,
    Parent: "world",
    Pose: contracts.Pose6D{
        X: end.X, Y: end.Y,
        Z: end.Z - p.cfg.BoxHeightMM/2, // place pose is the box's top face; drop to its center
        OX: end.OX, OY: end.OY, OZ: end.OZ, Theta: end.Theta,
    },
})
```

### Writing your prompt

We are adding three visual "events" — one at the pick station, one on the gripper, one on the
pallet — by calling `SetBoxVisual` directly in the relevant state methods. Work through these
questions as you form your prompt:

1. Which state does each of the three calls go in, and where in the grasp state should the
   attach call fire?
2. How does each call identify which box it's drawing?
3. For the attached box, what should `Parent` be, and what pose puts the box on the gripper?

### Review the code

- [ ] `SetBoxVisual` is called in `MOVING_TO_PICKUP_HOME`, `ENABLING_VACUUM`, and
      `RELEASING_AT_PALLET`.
- [ ] `MOVING_TO_PICKUP_HOME` publishes with `Parent: "world"` (absolute pose in the scene).
- [ ] `ENABLING_VACUUM` reuses the same `box.Seq` so the box is moved, not re-added, and
      switches `Parent` to the gripper.
- [ ] `RELEASING_AT_PALLET` publishes with `Parent: "world"` again, at `PlaceEndInWorld`.

### Verify

- [ ] Run a cycle while watching the 3D scene tab
- [ ] A box appears at the pick station when the cycle starts
- [ ] After the grasp, the box attaches to the gripper and tracks the arm through the transit
- [ ] When the gripper opens, the box appears at its place pose on the pallet
- [ ] Only one in-flight box is visible at any moment

---

## Part 9. Placed boxes and the held box in the world state (15 min)

### Concept

When the planner plans a move, it routes around whatever is in the `WorldState` you pass to
`motion.Move`. Every `moveToPose` call so far has passed `nil` — no world state. As boxes
accumulate on the pallet, the planner needs to know they're there so it never plans a path
through one (even though the tuned homes and box spacing usually keep the path clear). We'll add
a `combinedWorldState` helper that builds that world state and pass it on every `moveToPose` call.

A `WorldState` takes geometry two ways, and the choice comes down to whether it moves. **Static**
geometry — fixed in the world — goes in as an **obstacle (collision geometry)**. **Dynamic**
geometry that rides a component — a box on the gripper — goes in as a **transform** attached to
that component's frame, so the planner carries it along as the frame moves. So the placed boxes
and the held box go in different slots:

- **Placed boxes — obstacles.** A placed box sits at a fixed spot in the world. `GetStatus`
  returns which boxes are placed (`DoneSeqs`) and `GetPackOrder` returns every box's pose and
  dimensions, so you build one **world-frame obstacle** per placed seq.
- **The held box — a transform attached to the gripper.** Once the gripper grasps a box, the box
  becomes *part of the robot*: it rides the gripper, so the planner has to account for it on every
  move while it's held. You don't add it as a fixed obstacle — you attach its geometry to the
  **gripper's frame** as a transform, so it moves with the arm. `viamkit/worldstate`'s
  `GripperHeldBox` builds that: it returns a `*referenceframe.LinkInFrame` (the attached form,
  with the right offset baked in), which goes in the world state's **transforms**, not its
  obstacle list.

`Combined` takes the two lists separately:

```go
// combinedWorldState builds the WorldState the planner checks: the boxes already on the
// pallet (fixed obstacles) plus the box currently held by the gripper (a transform that
// rides the gripper frame).
func (p *palletizer) combinedWorldState(ctx context.Context) (*referenceframe.WorldState, error) {
    var placed []spatialmath.Geometry
    var transforms []*referenceframe.LinkInFrame

    status, err := p.packSequencer.GetStatus(ctx)     // DoneSeqs
    order, err := p.packSequencer.GetPackOrder(ctx)   // pose + dims per box
    for _, seq := range status.DoneSeqs {
        // one box geometry per placed seq, named "placed_box_<seq>"
        // NewBoxObstacle(pose spatialmath.Pose, dimsMM r3.Vector, label string)
        box, _ := worldstate.NewBoxObstacle(pose, dims, name)
        placed = append(placed, box)
    }

    if p.gripperIsHolding() {
        // the held box rides the gripper, so attach it to the gripper frame as a transform
        // GripperHeldBox(gripperName, linkName string, boxDimsMM r3.Vector)
        held, _ := worldstate.GripperHeldBox(p.cfg.Gripper, "held_box", dims) // dims is an r3.Vector built from the Part 4 box dims
        transforms = append(transforms, held)
    }

    // placed boxes become world-frame obstacles; the held box stays a gripper-frame transform
    return worldstate.Combined(worldstate.WorldObstacles(placed...), transforms)
}
```

### Writing your prompt

We're feeding the planner the boxes it needs to avoid — the ones already on the pallet, plus
the one in the gripper — through a `combinedWorldState` helper wired into every `moveToPose`
call. Work through these questions as you form your prompt:

1. Where should the `combinedWorldState` helper live, and will you copy in the sample above or
   have the Coding Agent write it?
2. Where do the placed boxes' poses and dimensions come from, and how do you turn each one
   into an obstacle the planner can see?
3. While the gripper holds a box it rides the arm — does that go into the world state as an
   obstacle, or as something attached to the gripper frame? Which `viamkit/worldstate` helper
   builds it?
4. Every `moveToPose` currently passes `nil` for the world state — what has to change at each
   call site?
5. The placed-box list only changes when a box is placed — where would you build it, and when
   does it need rebuilding?

### Review the code

- [ ] `combinedWorldState` returns a `WorldState` with the placed boxes as obstacles and the held
      box as a transform
- [ ] The placed boxes come from the pack-sequencer — `GetStatus` for which seqs are placed,
      `GetPackOrder` for their pose and size — and each becomes one named world-frame obstacle
      (`placed_box_0`, `placed_box_1`, …)
- [ ] The held box is added only while the gripper is holding, as a gripper-frame transform via
      `worldstate.GripperHeldBox` (a `LinkInFrame` in the world state's transforms, not the
      obstacle list)
- [ ] Every `moveToPose` now passes this `WorldState` instead of `nil`
- [ ] The placed-box list is cached, and rebuilt after each successful placement so the new box
      becomes an obstacle on the next cycle

### Verify

- [ ] Reset the pack-sequencer (`reset_progress` in CONTROL, args `{}`); the 3D scene shows an
      empty pallet
- [ ] Run several cycles. They should complete cleanly, and the trajectories should look the
      same as before — the pickup and pallet homes and the box spacing keep the path clear, so
      adding the obstacles doesn't change the visible motion
- [ ] Because the motion doesn't change, confirm the world state is actually reaching the planner
      another way: temporarily log `len(worldState.Obstacles())` and `len(worldState.Transforms())`
      in `moveToPose` — the obstacle count grows as boxes are placed, and the transform count is 1
      while a box is held (0 otherwise)
- [ ] Remove the temporary log once it reads as expected

---

## Part 10. Drive a full pallet (15 min)

### Concept

After all of our hard work, it is time for the full integration test of our simulated
palletizer. This Part runs the full pallet pack: Reset, Start, and watch all 8 boxes accumulate
on the pallet.

### Manual exercise — no prompt

Reset the box index, click Start, and watch the run. If something fails, fix it and re-run
from Reset.

### Verify

- [ ] The full pallet completes: 4 boxes in layer 1, then 4 boxes in layer 2
- [ ] Placed boxes appear in the 3D scene at their actual placed poses
- [ ] After the 8th placement, the cycle transitions to `PALLET_COMPLETE`
- [ ] Clicking Start again does nothing until Reset is clicked

---

## Done when

You can answer **yes** to all of these:

- [ ] `viam:pack-sequencer:sequencer` service is in the cell, green, and configured for
      the 8-box pack
- [ ] The palletizer has a `pack_sequencer` attribute and looks the service up via the
      worldstatestore API
- [ ] `next_box` drives the place pose for every cycle
- [ ] `ReportSuccess` is called once the box is down and the arm has cleared it, and
      `ReportFailure` when the cycle errors out
- [ ] The 3D scene shows each box on the pick station, attached to the gripper through the
      transit, then placed on the pallet
- [ ] Placed boxes appear in the planner's world state as obstacles
- [ ] `PALLET_COMPLETE` halts the cycle cleanly after 8 boxes; Reset clears both state machines
- [ ] A full 8-box, 2-layer pallet runs end-to-end from one Start click

## Takeaway

In this section we built a simulated palletizer that picks boxes from the pick station and
stacks a full pallet in pack order. The work split across two pieces: the palletizer module
handles the motion, the gripper, and the state machine, while the pack-sequencer service owns
the pack order, the box dimensions, and the record of what's been placed. And because the same
service that tracks the pack state also draws the boxes in the 3D scene, the visualization stays
in sync with the cycle on its own.
