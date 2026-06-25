# Section 15: Moving with motion constraints - defining the `moveToPose` helper

Time: 50 minutes.

Section 14 ended with a state machine whose `MOVING_OUT` and `MOVING_BACK` handlers waited
on a `sleepCtx` timer. This section replaces those waits with calls to the motion service.

We will also look at three types of motion constraints: *default*, in which the planner finds
collision-free motion in the world but is allowed to take any path to the
target; *linear*, in which the end-effector should move in a straight line from the current
pose to the target pose; and *orientation-locked*, in which the end-effector is constrained
to stay between the current orientation and the target orientation.

## What you will do in this section

- Replace a state's sleep stub with a `motion.Move` call wrapped in a `moveToPose` helper
- Switch between three planner constraint styles and describe what each constrains

## Setup check

- The Section 14 work runs: `start` advances the cycle through its sleep stubs, the Web
  Application's state diagram highlights each state, and `stop` / `start` / `restart` behave correctly.
- Resource graph is green: arm, gripper, floor, pallet, pick-station, workcell-scene,
  palletizer.
- `make` builds cleanly.

If the cell drifted, restore `prereq-machine-config.json` (replace `<NAMESPACE>`) and reload.

**Cycle-state count (going in):** 3 working from §13 (`IDLE`, `MOVING_OUT`, `MOVING_BACK`) + 2 terminal (`DONE`, `ERROR`). §15 replaces the sleep stubs with motion calls but does not change the state set.

---

## Part 1. `moveToPose` with three constraint styles (20 min)

### Concept

The motion service provides a `Move` method we have used previously. Up to now we passed
it what we want moved and the destination pose. As the cycle grows into real pickup and
place motions, every move is going to need more than just a destination. Some moves need
to follow a straight line so the gripper does not swing through other boxes. Some need to
keep the end-effector pointed down so a held box stays oriented. A plan that gets stuck
searching should not be allowed to hang the cycle for minutes. And when a plan fails or
the arm does something unexpected, we want a record of what was asked for.

A single `moveToPose` helper around `Move` gives the cycle's states one call for all of
that — a constrained, time-bounded, logged move. The rest of this Part covers what the
helper needs to know: the motion constraints and the planner timeout.

First, let's talk about motion constraints and how they work.

A motion constraint is a rule the planner has to satisfy in addition to "reach the
destination" — it governs *how* the trajectory gets there. Without one, the planner is free
to take any collision-free path; with one, only paths that satisfy the rule are
considered. The motion service supports these constraint types:

The code blocks below use three RDK packages: `motionplan`
(`go.viam.com/rdk/motionplan`) for the constraint types, `referenceframe`
(`go.viam.com/rdk/referenceframe`) for `NewPoseInFrame` / `World`, and
`spatialmath` (`go.viam.com/rdk/spatialmath`) for the pose helpers.

- **default**

  ```go
  req := motion.MoveReq{
      ComponentName: p.cfg.Gripper,
      Destination:   referenceframe.NewPoseInFrame(referenceframe.World, pose),
      // Constraints left nil — the planner picks any collision-free path.
  }
  ```

  No `Constraints` value. The planner is free to take any collision-free path to the
  target. It will still try to avoid colliding with obstacles.

- **linear**

  ```go
  constraints := &motionplan.Constraints{
      LinearConstraint: []motionplan.LinearConstraint{{
          LineToleranceMm:          10, // stay within 10 mm of the straight line
          OrientationToleranceDegs: 7,  // and within 7° of the interpolated orientation
      }},
  }
  ```

  The end-effector follows a path within `LineToleranceMm` of the straight line between
  start and goal, and within `OrientationToleranceDegs` of the interpolated orientation.
  If you set either tolerance to zero — or leave it undefined — that portion of the
  constraint is disabled, so be sure to define both.

- **orientation-locked**

  ```go
  constraints := &motionplan.Constraints{
      OrientationConstraint: []motionplan.OrientationConstraint{{
          OrientationToleranceDegs: 30, // up to 30° beyond the start→goal interpolation band
      }},
  }
  ```

  The end-effector's orientation is constrained to stay within `OrientationToleranceDegs`
  of the interpolation between the start and target orientations. Unlike the linear
  constraint, zero does *not* disable this constraint — it makes it as tight as possible.

- **pseudo-linear**

  ```go
  constraints := &motionplan.Constraints{
      PseudolinearConstraint: []motionplan.PseudolinearConstraint{{
          LineToleranceFactor:        1.0, // within (1.0 × travel distance) of the line
          OrientationToleranceFactor: 1.0,
      }},
  }
  ```

  The same idea as linear, but the tolerances are *factors* of the travel distance rather
  than absolute values. `LineToleranceFactor: 1.0` on a 100 mm move keeps the end-effector
  within 100 mm of the straight line; the same factor on a 300 mm move allows 300 mm. Use
  it when you want the path tightness to scale with move length instead of pinning a fixed
  millimeter budget. The fields default to zero, which (as with linear) turns the
  corresponding check off.

The `moveToPose` helper you build in this section wraps three of these — default, linear, and
orientation-locked — which are the three the cycle uses. Pseudo-linear is here so you
recognize it in the SDK; we won't be using pseudo-linear for the palletizer.

**Planner timeout.** The motion planner has to search for a solution, and for a complicated problem it can take
some time to find a valid one. We typically use a planner called CBiRRT (Constrained Bi-directional Rapidly-Exploring Random Tree) to find a collision-free motion that satisfies the constraints. The motion service provides a settable timeout to make sure we don't search for too long, and the default is 300 seconds. For palletizing, we can lower this to 15 seconds. The timeout value
goes in `motion.MoveReq.Extra` under the key `"timeout"` (as a float, in seconds):

  ```go
  extra := map[string]interface{}{
      "timeout": 15.0, // seconds
  }
  // serializes to JSON as: {"timeout": 15.0}
  ```

Let's take a look at it all together. Below is a linear-constrained move with the planner timeout, assembled into one `MoveReq`:

  ```go
  constraints := &motionplan.Constraints{
      LinearConstraint: []motionplan.LinearConstraint{{
          LineToleranceMm:          10,
          OrientationToleranceDegs: 7,
      }},
  }

  req := motion.MoveReq{
      ComponentName: p.cfg.Gripper,
      Destination:   referenceframe.NewPoseInFrame(referenceframe.World, pose),
      Constraints:   constraints,
      Extra: map[string]interface{}{
          "timeout": 15.0, // seconds
      },
  }
  ```

**Selecting a constraint using enums in Go.** As we work toward building the helper, we need
to look at how Go handles enums. `moveToPose` needs the caller to say *which* of the three
styles to apply, and passing a bare string like `"linear"` works until someone typos `"linear"`. 
The fix is an **enum** — a small, fixed set of named values that stand for one concept. 
Go has no `enum` keyword; you build one from a named type plus a block of constants. 
The `iota` keyword numbers the constants for you (0, 1, 2, …):

  ```go
  // moveOption selects the motion constraint moveToPose applies.
  type moveOption int

  const (
      moveDefault          moveOption = iota // 0
      moveOrientationLocked                  // 1
      moveLinear                             // 2
  )
  ```

The caller now writes `moveLinear`, never `"linear"` or `2`: a typo becomes a compile error,
and the whole valid set lives in one place. Give the type a `String()` method so it prints as
a readable word in logs instead of a bare number:

  ```go
  func (m moveOption) String() string {
      switch m {
      case moveOrientationLocked:
          return "orientation-locked"
      case moveLinear:
          return "linear"
      default:
          return "default"
      }
  }
  ```

You have seen this pattern already: Section 13's `State` values (`IDLE`, `MOVING_OUT`, …) are
the same idea in its string-backed form — a named type with one constant per value.

**Writing to the logs:** It's very helpful to log motion calls. Before calling `Move`, log where the gripper is, where it is going, and which constraint is in effect. If a plan fails or the arm does something unexpected, the log shows what we asked the arm to do:

  ```go

  // get the current gripper pose
  current, err := p.motion.GetPose(ctx, p.cfg.Gripper, referenceframe.World, nil, nil)
  if err != nil {
      return err
  }

  // log a helpful message about the motion call
  p.logger.Infow("moveToPose dispatch",
      "constraint", opt, // moveOption.String() prints "default" / "linear" / ...
      "destination", spatialmath.PoseToProtobuf(pose).String(),
      "current", spatialmath.PoseToProtobuf(current.Pose()).String(),
  )
  ```

  We append `.String()` to each `PoseToProtobuf` call because the protobuf
  pose is a struct the logger can't format directly — without `.String()`
  the log line shows `destinationError: PANIC=...` and the pose values are lost.
  Every protobuf message has a `.String()` method that turns it into readable
  text, so the log shows something like
  `x:400 y:0 z:400 o_z:-1` instead of a panic placeholder.

### Remove §12's `move_to_pose` first

As the state machine starts driving real motion, we need a more capable `moveToPose` —
one that picks a motion constraint, bounds the planner's search time, and logs each move.
Section 12's `moveToPose` was a basic DoCommand handler: it decoded a pose from the
request and called `motion.Move`, with no constraints or timeout. Nothing past §12 uses
it, so delete it and build the better one in its place:

1. Remove the `"move_to_pose"` entry from the verb table in `NewPalletizer`.
2. Delete the `moveToPose` method we wrote in Section 12 (in `module.go`). That method was
   the only user of four imports — `github.com/golang/geo/r3`,
   `go.viam.com/rdk/referenceframe`, `go.viam.com/rdk/spatialmath`, and
   `go.viam.com/rdk/utils` — so once it is gone, prune those four from `module.go`'s import
   block (let your IDE organize imports, or delete the lines by hand); they move to the new
   `motion.go`. A terminal `make` reports `imported and not used` for each one until you do.
3. Rebuild — `make` should be clean. That also frees the `moveToPose` name for the new
   helper you write next.

### Writing your prompt

Now that we have been introduced to the constraints, understand how to set the timeout, and
know how to write to the logs, we can either create a prompt or write the helper by hand.

We want to add the `moveToPose` helper around the motion service's `Move` call, so the
cycle's future states can request a constrained, time-bounded, logged move in one call.

Before you implement, work through these questions:

1. Where does this helper live? Maybe we want to make a new file called `motion.go` to build up as we add more helpers to the module package.
2. What is the function signature? What does it take, what does it return?
3. How will we pass in the constraint type we want to use? How does each constraint style
   correspond to the different SDK types?
4. What should we write to the logs? Should we add the current position as well as the
   destination pose? What about the constraint type?
5. What `ComponentName` should we use in our `motion.MoveReq` — the arm or the gripper?

Now write the prompt. Be specific about the signature and the constraint-selection
mechanism. Those are your design decisions, not the Coding Agent's.

### Review the code

Check the generated code against this list before running it:

- [ ] The helper is defined and takes a destination pose plus a constraint
- [ ] The three constraint styles are selected through a named `moveOption` enum constant
      (default, orientation-locked, linear), not a bare integer or string literal
- [ ] The planner timeout is 15.0 seconds, passed in `Extra["timeout"]`
- [ ] `ComponentName` on the `MoveReq` is the gripper, not the arm
- [ ] An info log line at dispatch contains: destination pose, constraint label, and
      current gripper pose
- [ ] The helper takes a `context.Context` and respects cancellation

If anything is missing or wrong, ask the Coding Agent to fix it before moving on.

### Verify

- [ ] `make` is clean
- [ ] `viam module reload` succeeds and the palletizer is green
- [ ] You can call `moveToPose` from a test verb (or by temporarily calling it from
      `MOVING_OUT`) with each of the three constraint styles
- [ ] The dispatch log line appears for each call with the correct constraint label
- [ ] Setting the planner timeout to 1 second and calling with an unreachable target fails
      in about 1 second, not the 15 seconds the helper normally allows; restore 15 seconds before moving on

---

## Part 2. Replace the sleep stubs with `moveToPose` calls (20 min)

### Concept

We have our `moveToPose` helper; now let's actually use it. `MOVING_OUT` and `MOVING_BACK`
currently sleep — we are going to replace each sleep with a `moveToPose` call. Along with
the change, we need to make sure the method's context is passed through to `moveToPose`.

Work through these before you code:

1. Which file holds `MOVING_OUT` and `MOVING_BACK`? Which lines have the `sleepCtx` calls
   today?
2. What is a reachable target pose for the simulated UR5e? A pose around
   `(300, 0, 400)` with the gripper pointing down works. How do you express "gripper
   pointing down" as an orientation?
3. What context is passed into the state handler? How do you forward it into `moveToPose`?
4. Which constraint style do you want to test? 
5. Where does `MOVING_BACK` move to? We recommend a near-origin pose that does not intersect the
   floor or the arm — for example `(0, 300, 400)` with the gripper pointing down, parallel to
   the `(300, 0, 400)` target used for `MOVING_OUT`.

Now write the code. Keep it scoped: you are replacing the two `sleepCtx` waits with two
`moveToPose` calls. 

### Verify

- [ ] `start` drives the arm to the target in the 3D scene, then back, then to `DONE`
- [ ] `stop` mid-move halts the arm immediately, and the state machine remains on the
      current state
- [ ] `start` after `stop` resumes from the state it was in when `stop` was called
- [ ] `restart` resets and runs again from `IDLE`
- [ ] The Web Application's state diagram still highlights each state correctly

---

## Part 3. Try the three motion styles (10 min)

### Concept

Change `MOVING_OUT`'s `moveToPose` option from `default`
to `orientation-locked`, rebuild, reload, run a cycle, and watch the trajectory. Then
switch to `linear` and run again.

What you should see:

- **Default:** the planner chooses any collision-free path.
- **Orientation-locked:** the path keeps the gripper's orientation within tolerance of the
  start-to-end interpolation. Path geometry is often similar to default, but the wrist does
  not rotate freely.
- **Linear:** the end-effector follows a path near the straight line between start and goal.

The default motion could look exactly like the linear or orientation-locked motion. It doesn't 
mean anything is wrong, just that there are usually many collision-free motions within the arm's 
workspace, and the solver can find different answers depending on many factors.

### Verify

- [ ] You ran each of the three options and observed the trajectory in the 3D scene tab
- [ ] You can describe in one sentence what changed between the three trajectories

---

## Done when

You can answer **yes** to all of these:

- [ ] `moveToPose` exists with three constraint options (default, orientation-locked, linear)
      and a 15-second planner timeout
- [ ] `MOVING_OUT` and `MOVING_BACK` move the arm and the cycle can be run using the `start`, `stop`, and `restart` buttons from the Web Application

## Takeaway

The state machine and Web Application are unchanged. The sleeps have been replaced with
motion-service calls through a single helper. Every future state that moves the arm will
call the same helper.
