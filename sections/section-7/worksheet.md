# Section 7: Frame system, orientation, and end-effector frames

Time: 75 min. Individual work, in your own work cell simulation.

Through Section 6 you worked on a single machine and exercised one resource at a time. From here the course is spatial: where things are relative to each other, and how the motion service reasons about that geometry. This section is the foundation for everything in the motion stack. You build a frame tree for the work cell, point the gripper where you want it with an orientation vector, offset the end-effector frame to the gripper's real tool-tip, and verify every change by eye in the 3D scene tab.

There is no code here. The work is configuration and reading the 3D scene. The one idea that trips up everyone, including people with robotics backgrounds, is how Viam describes orientation. Spend your time there.

## What you will be able to do after this section

- Define the world frame and parent-child frame relationships, and explain why the tree runs one way
- Configure translation and orientation between a parent and child frame
- Read and write an `ov_degrees` orientation vector: set where the tool points, then twist it with `theta`
- Offset an arm's end-effector (TCP) frame so motion calls plan to the actual tool tip
- Compose a multi-component frame tree: gripper on the arm, camera on the wrist
- Use the 3D scene tab as your feedback loop, and recognize a broken composition on sight
- Recognize geometries (box, sphere, capsule) as the collision volumes the planner reasons about, and know they get configured in Section 9
- Tell a static frame (in config) apart from a dynamic obstacle (covered in Section 9)

## Setup check

- You can open your work cell simulation and its machine shows green in the Viam app.
- The machine page has a 3D scene tab, and it renders the simulated arm.
- You can reach the CONFIGURE tab and switch between the form view and the JSON view, the way you did in Section 3.

## Part 1. Concepts framing (5 min)

In one sentence each:

- **World frame:**

- **Parent-child relationship:** (which way does the tree run, and what does it mean that it runs that way?)

- **Translation vs. orientation:** (what does each part of a frame describe?)

- **Static frame vs. dynamic obstacle:** (hint: one lives in config, one is passed per motion call; full coverage is Section 9)

- **Geometry (collision volume):** (what is a box, sphere, or capsule attached to a frame for? you configure these in Section 9; here just name what they are)

Raise your hand if any of these feel uncertain before you start configuring.

## Part 2. Watch the instructor build a frame tree (10 min, instructor-led)

Your instructor builds a frame tree live in the work cell sim, with the 3D scene tab open the whole time. Watch for:

- How a child frame is attached to a parent, and what the 3D scene shows the instant the parent changes
- The orientation vector: the instructor points the gripper straight down by setting **where it points**, then twists it with `theta`, and names the difference out loud
- The end-effector frame: where the arm's default tool tip is, and what moving the offset does to the rendered tool point
- At least one deliberate mistake (a wrong parent or a wrong pointing vector) and what the broken composition looks like before it gets fixed

The point is the read-it-in-the-scene habit. You will use it in every part that follows.

## Part 3. Build the basic frame tree (15 min)

Build the work cell's frame tree from the world frame down. Keep the 3D scene tab open and check after each step.

- Confirm the **world** frame exists; it is the root, and every other frame traces back to it.
- The simulated **arm** is parented to **world**. Set its translation so the arm base sits where the cell geometry expects it.
- Parent the **gripper** to the arm's **end-effector frame** (not the arm's base, and not world). Leave its offset at zero for now; you fix the offset in Part 5.
- Parent the **camera** to the arm's **wrist** so it moves with the arm.

Record:

- The parent of each frame you configured:
  - arm →
  - gripper →
  - camera →

- One sentence: what happened in the 3D scene the moment you parented the gripper to the end-effector frame?

## Part 4. Point things with `ov_degrees` (15 min)

This is the part to slow down on. An `ov_degrees` orientation vector has two jobs, and they are separate:

- `x, y, z` is **where the frame points**: the direction the tool's local Z axis aims, as a point on the unit sphere. The default `z: 1` points straight up.
- `th` (theta) is the **roll**: an in-line twist around the direction you are already pointing. Changing `th` does not change where the tool points, only how it is rotated about that line.

So the move is always: set the pointing direction first, then twist with `th` if you need to.

Hit each of these three targets by editing the orientation vector and reading the result in the 3D scene tab. Find the values yourself; do not look them up.

1. **Point the gripper straight down** (toward the table).
2. **Face the gripper toward the pickup stand** (pointing roughly horizontal, toward the stand).
3. **Tilt the camera 30 degrees forward** from straight down.

Record, for each target, the orientation vector you landed on:

| Target | x | y | z | th |
| ------ | - | - | - | -- |
| 1. Gripper straight down | | | | |
| 2. Gripper toward pickup stand | | | | |
| 3. Camera 30° forward | | | | |

Answer:

- For target 1, change **only** `th` and watch the 3D scene. What moved, and what did not? Say in one sentence what `th` controls.

- In your own words, how is this different from describing orientation as "an axis to rotate around and an angle to rotate by"?

## Part 5. Offset the end-effector frame to the real TCP (10 min)

Every arm ships with a default end-effector frame at its last link. The motion service plans **that frame** to whatever pose you ask for. If the frame sits at the wrist flange but your real grip point is at the gripper's fingertips, every motion lands short by the length of the gripper.

- With the gripper config in front of you, find the offset from the arm's last link to the gripper's actual tool-tip (the TCP). Your instructor gives the gripper's reach.
- Set the end-effector frame's translation (and orientation, if the tool is angled) to that offset.
- In the 3D scene tab, confirm the rendered tool point now sits at the fingertips, not the flange.

Answer:

- What offset did you set, and along which axis is most of it?

- The motion service plans the end-effector frame to the target pose. If you left the offset at zero, where would the gripper actually go relative to where you asked, and by how much?

## Part 6. Break a frame on purpose (10 min)

You will recognize these mistakes later only if you make them once on purpose now.

- **Wrong parent:** re-parent the camera from the wrist to **world**. Move the arm (through the arm's test card) and watch the 3D scene. Does the camera follow the arm anymore? Put it back.
- **Wrong pointing vector:** take your "gripper straight down" frame and flip the pointing direction (point it up, or sideways). Look at the 3D scene. Describe what a wrong ov looks like before you fix it.

Record:

- What did re-parenting the camera to world change about how it behaves when the arm moves?

- What did the broken orientation look like in the 3D scene, and what tells you at a glance that an ov is wrong?

## Part 7. Reflection and quiz (10 min)

Answer in writing.

1. Why does the frame tree run one way (parent to child)? What would it mean for the gripper to be the parent of the arm?

2. An `ov_degrees` value has `x, y, z` and `th`. What does each part do, and which one do you set first when you want the tool to point a specific way?

3. The motion service plans the end-effector frame to the target pose. Why does the TCP offset have to be right before any motion call will land where you intend?

4. Name one frame that is **static** (belongs in the cell config) and one thing that would be a **dynamic obstacle** (passed per motion call, Section 9). What is the difference?

5. You configured the camera parented to the wrist. Why parent it there instead of to world? When would parenting a frame to world be the right choice instead?

When everyone finishes, we will discuss as a group.
