# Section 12: First DoCommand verb

Time: 60 minutes.

In the previous section you scaffolded a module that loads on your cell and rejects configurations with missing or incorrect dependencies. In this section you'll extend the module by hand to implement a `move_to_pose` DoCommand verb that drives the arm through the motion service, and exercise it from the Viam app's CONTROL tab. You'll build on this module in every later section.

## What you will do in this section

- Implement a DoCommand verb in a Viam Go module
- Move an arm using the Viam motion service via `motion.Move` 
- Explain how the framesystem composes child→parent frames and how that affects `motion.Move`
- Execute a DoCommand verb from the Viam app's CONTROL tab
- Add input validation to a DoCommand verb's method so bad requests fail loudly

## Setup check

Before you start, confirm:

- Section 11's palletizer loads on your cell with a green status; the resource graph shows `arm`, `gripper`, `floor`, `pallet`, `pick-station`, `workcell-scene`, and `palletizer` all green.
- VS Code is open to your module
- If your cell config looks broken or you don't remember the state you left it in, you can restore from [`sections/section-12/prereq-machine-config.json`](./prereq-machine-config.json). Substitute `<NAMESPACE>` for your org's namespace, then push it via the Viam app's raw config editor. If the `palletizing-module` shows unavailable afterward, re-run `viam module reload --part-id <PART_ID>` to reinstall the hot-reloaded build.

If anything is off, raise your hand.

## Part 1. Concepts recap (5 min)

In one sentence describe each:

- **What is the difference between a normal cloud build and hot-reloading?**

- **What frame is the gripper attached to and does the gripper move with the arm?**

- **What file lists the dependencies for our Go Module?**

Raise your hand if any of these feel uncertain.

## Part 2. Wire `move_to_pose` by hand (20 min)

Next we are going to get the module to do something with the two resources it has access to; the arm and gripper. DoCommand is a nice way to add verbs that are not already provided by the module or service's built in API. We are going to make a "move_to_pose" verb and provide a cartesian pose for the arm to move to.

### Find the DoCommand dispatch

Open `module.go` in the IDE and locate the `DoCommand` method on the palletizer (search for `func (p *palletizer) DoCommand`). The scaffold's version returns a "not implemented" error for any input — we'll replace its body with something more useful.

### Add the dispatch table

Replace the body of `DoCommand` with:

```go
func (p *palletizer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
    if args, ok := cmd["move_to_pose"].(map[string]interface{}); ok {
        return p.moveToPose(ctx, args)
    }
    return nil, fmt.Errorf("unknown command, supported: [move_to_pose]")
}
```

A few things to notice:

**The DoCommand receives two parameters, `ctx` and `cmd`.**

- `ctx` is a Context object used to help manage requests across processes. This may contain timeout requirements, or let you know if the caller canceled. 

- `cmd` can be thought of as a user definable Struct that contains information about the action DoCommand should take. When we send our verbs to DoCommand, we can typically use a key : value pair: `{ "stop" : true }` or nested such as `{ "move_to_pose" : {"x": 400, "y": 0, "z": 400}}`.  

**DoCommand also returns two values**
- a response Struct, which we can use to include the results, or provide any information in response to the requested verb

- an error field. If everything is good, we will return nil, otherwise we can return an error message. It is good practice to be descriptive here. `fmt.Errorf("a clear error message")`

**Pulling out the Verb and the supplied arguments**

In Go, the code below says "if this key exists and is the right type, store the value in args and set ok to true, then do something". We will use this to unmarshal received structs to something usable. 

```go
if args, ok := cmd["move_to_pose"].(map[string]interface{}); ok {
    return p.moveToPose(ctx, args)
}
```
### Add the `moveToPose` method
Next, let's add a method called `moveToPose` below `DoCommand` to process the inputs and call the Motion Service to move the arm:

```go
func (p *palletizer) moveToPose(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    x, _ := args["x"].(float64)
    y, _ := args["y"].(float64)
    z, _ := args["z"].(float64)
    roll, _ := args["roll"].(float64)
    pitch, _ := args["pitch"].(float64)
    yaw, _ := args["yaw"].(float64)

    point := r3.Vector{X: x, Y: y, Z: z}
    orient := &spatialmath.EulerAngles{
        Roll:  utils.DegToRad(roll),
        Pitch: utils.DegToRad(pitch),
        Yaw:   utils.DegToRad(yaw),
    }
    dest := referenceframe.NewPoseInFrame("world", spatialmath.NewPose(point, orient))

    _, err := p.motion.Move(ctx, motion.MoveReq{
        ComponentName: p.cfg.Gripper,
        Destination:   dest,
    })
    if err != nil {
        return nil, err
    }
    return map[string]interface{}{"moved": true}, nil
}
```

A few things to notice:

- First we extract each value from `args` with a type assertion. The two-value form `x, ok := args["x"].(float64)` reads as "if this key exists and is a `float64`, store it in `x` and set `ok` to true." Here we use `_` in place of `ok` to ignore that check for now — a missing field just yields a zero value. Part 4 comes back to enforce good structure and handle bad input.
- Watch for Units. We are going to let users provide the desired orientation in degrees, but often we will need to convert to radians behind the scenes. Be sure to check the documentation to see which units are expected. 
- When sending a desired pose, we use an object of type `referenceframe.PoseInFrame`. `Destination` is a `referenceframe.PoseInFrame` of `"world"` plus a `spatialmath.NewPose(point, orient)`. The `"world"` string is the framesystem's root frame; 

***motion.Move***
 `motion.Move` takes two arguments: the context and a `MoveReq`. `MoveReq` is a struct with several fields (the full set is covered in "What `motion.Move` does" below); the one to get right here is:

- `ComponentName` is a `string` referencing the component you want to move. With our current setup, we can use either the arm or the gripper. If you use the name of the arm, the motion service will move the Arm's output frame to the destination pose. If you specify the Gripper, it will move the Gripper's frame to the destination pose. 

### Wire the motion-service dependency

The method uses `p.motion` (a motion-service client) and `p.cfg.Gripper` (the configured gripper name — already on the struct via `cfg`). `p.motion` is the part that doesn't exist yet. We need to declare the motion service as a dependency, look it up in the constructor, and store it in the palletizer struct.

First, add one field to the `palletizer` struct (find `type palletizer struct {...}` near the top of `module.go`). The scaffold already gave it `name`, `logger`, `cfg`, and a `cancelCtx`/`cancelFunc` pair — keep all of those and add `motion`. (The arm and gripper names you need are already in `cfg.Arm` / `cfg.Gripper`, so there's nothing to copy out.)

```go
type palletizer struct {
    resource.AlwaysRebuild
    resource.Named

    name   resource.Name
    logger logging.Logger
    cfg    *Config

    // added this section:
    motion motion.Service

    cancelCtx  context.Context
    cancelFunc func()
}
```

Next, update `Validate` so that it lists the motion service as a dependency. `Validate` returns two slices of dependencies: the first for required dependencies, the second for optional ones. The palletizer can't move the arm without the motion service, so it's required too — add it to the first slice, alongside the arm and gripper:

```go
func (cfg *Config) Validate(path string) ([]string, []string, error) {
    if cfg.Arm == "" {
        return nil, nil, resource.NewConfigValidationFieldRequiredError(path, "arm")
    }
    if cfg.Gripper == "" {
        return nil, nil, resource.NewConfigValidationFieldRequiredError(path, "gripper")
    }
    return []string{cfg.Arm, cfg.Gripper, motion.Named("builtin").String()}, nil, nil
}
```

Finally, lets look up the motion service in the constructor. Add the `motion.FromProvider` lookup before we make the palletizer struct, and set the new `motion` field on it:

```go
func NewPalletizer(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (resource.Resource, error) {
    cancelCtx, cancelFunc := context.WithCancel(context.Background())

    ms, err := motion.FromProvider(deps, "builtin")
    if err != nil {
        cancelFunc()
        return nil, fmt.Errorf("palletizer needs the builtin motion service: %w", err)
    }

    p := &palletizer{
        name:       name,
        logger:     logger,
        cfg:        conf,
        motion:     ms,
        cancelCtx:  cancelCtx,
        cancelFunc: cancelFunc,
    }
    return p, nil
}
```

Things to notice:

- `"builtin"` is the default name of the motion service that ships with viam-server and is provided automatically.
- New imports `module.go` needs: `"go.viam.com/rdk/referenceframe"`, `"go.viam.com/rdk/services/motion"`, `"go.viam.com/rdk/spatialmath"`, `"go.viam.com/rdk/utils"`, and `"github.com/golang/geo/r3"`. The IDE's Go integration auto-fills the import block when you save; if not, run `goimports -w module.go`.

### Build

Save the file and rebuild from the module's repo root:

```bash
make
```

Record the result. A clean build is what we want. If the build fails, paste the specific compile error into Claude Code — it usually catches its own mistakes. Raise your hand if you're stuck.

### What `motion.Move` does

`motion.Move(ctx, MoveReq)` is the motion service's top-level entry point. It plans a path from the arm's current pose to the requested destination, executes that path, and blocks until the move completes (or fails).

The fields on `MoveReq` you used here:

- **`ComponentName`** — the resource the planner is moving. More on this in the next subsection.
- **`Destination`** — a `referenceframe.PoseInFrame`: a 6 degrees of freedom (DoF) pose (position + orientation) plus the name of the frame those coordinates are relative to. For world-coordinate poses, the frame is `"world"`.

#### The framesystem Recap

The Viam framesystem is a tree of named coordinate frames. Every component that has a physical location declares a `frame` block in its config: a `parent` (another frame, ultimately rooted at `world`) and a translation + orientation relative to that parent. The motion service composes the child→parent transforms upward at planning time, so given any leaf frame (your gripper) and a destination expressed in any ancestor frame (here, `world`), the planner can compute the joint angles that put the leaf at the destination.

This is why `move_to_pose` works the way it does. You describe destinations in `world` coordinates — millimetres along the cell's X/Y/Z axes — and the framesystem composes the chain `world → arm-base → arm-flange → gripper-tip` to know where each link of the kinematic chain has to land. The 196 mm +Z translation you declared in Section 10 when you wired the gripper to the arm is one link in that chain; the planner picks it up automatically. The Viam app's **3D scene** tab renders the live framesystem — every named frame is an axis triad attached to its parent.

Other `MoveReq` fields you'll encounter in later sections:

- **`WorldState`** — the obstacle field the planner should avoid. It also carries transforms: geometry that moves with a component, such as the gripper and camera parented to the arm, so attached objects travel with the arm during planning ([transforms reference](https://docs.viam.com/motion-planning/obstacles/attach-detach-geometries/#the-transform-message)).
- **`Constraints`** — orientation lock, straight-line-only, joint limits. A later section introduces the three motion styles (default, orientation-locked, linear).
- **`Extra`** — planner-specific knobs like timeout and algorithm pinning.

What happens inside the motion service when you call `Move`:

1. It looks up `ComponentName` in the framesystem and finds its frame.
2. It composes the framesystem chain `world → arm-base → arm-flange → gripper-tip` to find the transform from the arm's flange to the requested component. The 196 mm +Z stand-off from Section 10 is one link in that chain.
3. It plans a joint-space path so that, when executed, the component's frame origin lands at `Destination`.
4. It hands the path to the arm's executor and returns when execution finishes.

A goal outside the arm's reachable workspace, or one that would collide with anything in `WorldState`, comes back as a planner error before any motion happens. That's the "fail-fast" property of the planner — you'll lean on it in a later section with the verify pattern.

### Why `ComponentName` is the gripper, not the arm

This is where the framesystem matters in practice. When you pass `ComponentName: p.cfg.Gripper` (the gripper's name, `"gripper"`), you're telling the planner: "move so the **gripper's** frame origin lands at `Destination`." The motion service:

1. Looks up the gripper in the framesystem by that name.
2. Sees that the gripper is parented to the arm with a 196 mm +Z translation (the vacuum stand-off from Section 10).
3. Plans the arm's joints so that, after the move, the arm's flange is 196 mm short of `Destination` along the local Z — putting the gripper's frame origin (the vacuum tip) exactly at `Destination`.

If you instead pass `ComponentName: p.cfg.Arm` (the **arm's** name), the planner moves the **arm's end-effector** to `Destination`, leaving the gripper offset from where you wanted it by the 196 mm stand-off. On the motion tab in Part 3 you'd see the **arm** sitting at `Destination` and the **gripper** 196 mm off — that's the symptom to watch for.

## Part 3. Verify the move from the test card (10 min)

Hot-reload the new build onto your cell:

```bash
viam module reload --part-id <PART_ID>
```

In the Viam app, open the **CONTROL** tab and find the `palletizer` service card. The generic-service card exposes a **DoCommand** input — a JSON text box and an Execute button.

Send the verb you just wired:

```json
{
  "move_to_pose": {
    "x": 400,
    "y": 0,
    "z": 400,
    "roll": 0,
    "pitch": 180,
    "yaw": 0
  }
}
```

Click **Execute**. Watch the 3D scene tab — the arm should move so the gripper's vacuum tip lands at the requested pose. 

Record:

- DoCommand response:

- Where did the gripper actually land? The CONTROL-tab arm card's **End Position** reports the *arm's* end-effector pose, not the gripper's — so to read the gripper's pose in the **world** frame, open the **motion** tab, which lists each component's pose (gripper and arm). The **gripper** should be at (400, 0, 400):

Try a second pose with a different orientation to confirm the arm reorients as it moves:

```json
{
  "move_to_pose": {
    "x": 300,
    "y": 200,
    "z": 300,
    "roll": 0,
    "pitch": 180,
    "yaw": 45
  }
}
```

(Same gripper-down orientation, rotated 45° around the down-axis.)

A note on what the response actually tells you: a `{moved: true}` response means the motion service committed a plan, NOT that the arm physically landed at the target. The **motion** tab's gripper pose is the real check — always pair the DoCommand response with the motion-tab readout when the cycle output matters.

## Part 4. Break-on-purpose + harden the method (15 min)

A big part of building reliable cells is finding silent failures before they ship. The two breaks below introduce the most common shapes you'll meet: a value shipped in the wrong type (a string where the method expects a number) silently becoming zero, and the Go method silently zeroing missing fields when no input check rejects them.

### Break 1: ship the wrong type

From the CONTROL-tab DoCommand input, send `move_to_pose` with one coordinate **quoted as a string** instead of a number:

```json
{
  "move_to_pose": {
    "x": "400",
    "y": 0,
    "z": 400,
    "roll": 0,
    "pitch": 180,
    "yaw": 0
  }
}
```

Record:

- Where does the arm go — to `x = 400`, or somewhere else?

- What does the call return — a result, or an error? If an error, what does it say?

The lesson: JSON `"400"` is a string, so the comma-ok assertion `x, _ := args["x"].(float64)` silently fails and leaves `x` at its zero value — the method asks the planner for `(0, 0, 400)` instead of `(400, 0, 400)`. The IK error you got is incidental: this particular zeroed pose happens to be unreachable, but a different coordinate could have zeroed into a perfectly reachable pose and moved the arm to the wrong place with nothing to flag it. This is the first "silent-zero" pattern you'll meet — a wrong-typed field that defaults to zero unnoticed — and Break 2 hardens the method so the same mistake produces a clear, deliberate error instead.

### Break 2: add input validation by hand

Shipping the wrong type is one path to a silent error; a *missing* field is another. JSON typos in the test card hit the same trap — forget a field and the missing value defaults to zero, the arm moves to a half-zero pose, and nothing complains. We'll harden the method by hand so missing fields produce a clear error before the planner ever runs.

Open `module.go`, find the `moveToPose` method from Part 2, and insert a required-field check at the top of the function body — before any of the `args["x"].(float64)` lines:

```go
func (p *palletizer) moveToPose(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
    required := []string{"x", "y", "z", "roll", "pitch", "yaw"}
    var missing []string
    for _, k := range required {
        if _, ok := args[k].(float64); !ok {
            missing = append(missing, k)
        }
    }
    if len(missing) > 0 {
        return nil, fmt.Errorf("move_to_pose: missing or non-numeric fields: %v", missing)
    }

    // (existing method body continues here)
    x, _ := args["x"].(float64)
    // ...
}
```

Three things to notice:

- The check reuses the same `.(float64)` type-assert the method body uses — both "missing" and "wrong type" land in the `!ok` branch. That's important: a request that ships a string for a numeric field (the Break 1 case) gets the same clear error as one that omits the field entirely.
- The error names every missing field in one shot, not just the first one. The test-card operator can fix all the typos before replaying, instead of fixing one at a time.
- Returning the error short-circuits before `motion.Move` runs. The arm never moves, the planner never plans — We want to fail fast and fail early if there is an issue.

Rebuild (`make`) and hot-reload (`viam module reload --part-id <PART_ID>`). Test from the CONTROL-tab DoCommand input — send a request that's missing `pitch`:

```json
{
  "move_to_pose": {
    "x": 400,
    "y": 0,
    "z": 400,
    "roll": 0,
    "yaw": 0
  }
}
```

The response should name `pitch` as missing. Try a few more — drop different fields, send an empty `move_to_pose: {}`, etc. Each case should produce a clear error.

Every verb you add in later sections will want this same shape of input check. The method is the right place for it; the dispatcher only knows the verb name, not what fields each verb requires.

## Done when

You can answer **yes** to all of these:

- `make` from your module's repo builds clean after the `move_to_pose` method edits (Part 2).
- The palletizer's DoCommand `move_to_pose` runs from the test card and the arm visibly traverses to each target in the 3D scene.
- The motion tab's gripper pose confirms the gripper's vacuum tip landed at the requested pose in world (Part 3).
- You reproduced the silent-zero by sending a string-valued coordinate from the test card, and the hardened method then rejected it (Part 4).
- You tested that the hardened method rejects a request missing `pitch` and named the missing field instead of silently moving the arm (Part 4).

If any of these are no, capture the symptom and the LOGS line; raise your hand.

## Takeaway

The shape every later palletizer section reuses is now in place:

1. **Motion service through DoCommand.** Every later cycle state (pickup, grasp, retract, place) is the same shape — build a `PoseInFrame`, call `motion.Move` using the gripper as the component to move, return a result map. The framesystem composes the chain `world → arm-base → arm-flange → gripper-tip` so you describe destinations in world coordinates and the planner finds the joint angles and moves the arm.

2. **Silent-zero is the worst class of bug** — no error fires, the cycle just runs wrong. Part 4's two breaks (a field shipped as the wrong type, a missing required field) are the first two you'll meet. Typed wire shapes in a later section ultimately replace ad-hoc validation; until then, by-hand required-field checks at the top of every verb's method are the right defence.
