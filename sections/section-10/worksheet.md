# Section 10: Building the Virtual Work Cell

Time: 85 minutes.

Sections 1-9 built your understanding of the Viam platform: machines, components, services, the motion service, the framesystem, simulation, WorldState. This section starts a new arc where you build a palletizer module using the Viam platform.

Before we start developing the palletizer module, we need to import the components that make up the workcell. This section does just that. We will import a simulated arm with a gripper attached, a floor for the planner to avoid, and a pallet and pick-station so the arm has a clear job — pick a box from one, place it on the other. You'll build all of it in the Viam app's CONFIGURE tab and watch it appear in the 3D scene as you go.

## What you will do in this section

- Use the Viam app's CONFIGURE tab to add components from the registry
- Place components in world coordinates with the frame editor
- Distinguish a `simulated` module from real hardware components, like those used in earlier sections
- Import a fragment into a machine configuration and update the fragment's attributes
- Use the History button to roll back a misconfig

## Setup check

- You can reach your assigned machine in the Viam app.
- The machine status is green (viam-server is up and reachable).

## Part 1. Concepts recap (5 min)

In one sentence describe each:

- **Component (vs Service):**

- **`frame` block (parent, translation, orientation):**

- **`world` frame:**

- **Why a planner needs a `geometry` block on a component:**

## Part 2. Add the simulated arm (10 min)

Viam offers simulated and fake components to act as testing modules to help you develop without hardware. Today we are going to be using the simulated arm component. This component has the same methods and moves the same way a real arm would move. The simulated arm model comes with a few attributes to help you configure the arm's type and speed. We will use this component to build the palletizer module virtually, and in later sections swap the simulated model for a real arm driver.

In the Viam app, open the CONFIGURE tab. Click **+ Configuration Block**. In the registry picker, search for `simulated` and pick the **arm/simulated**. Click **add to machine** and name the component **arm**.

Simulated Arm Attributes:

```json
{
  "arm-model": "ur5e",
  "simulate-time": true,
  "speed": 0.3
}
```


Now we have to tell Viam how the arm is attached to the frame system. For components that have a statically mounted base, such as an arm, fixed sensor, or a conveyor, we can add a frame attribute directly to the component. A frame typically consists of:

- `Parent Frame`: The name of the frame this component is attaching to
- `Translation`: The distance in mm along X,Y,Z this component is from the parent frame
- `Orientation`: The rotation of the component, after the component has been translated. See [Supported Orientation Types](https://docs.viam.com/motion-planning/reference/orientation-vectors/?source=searchResultItem#supported-orientation-formats) for more details about Orientation Vectors, Euler angles, Axis Angles, and Quaternions.

Note that we are mounting the arm to the origin (X:0, Y:0, Z:0) with `world` as the parent:

**Frame:**

```json
{
  "orientation": {
    "type": "ov_degrees",
    "value": {
      "th": 0,
      "x": 0,
      "y": 0,
      "z": 1
    }
  },
  "parent": "world",
  "translation": {
    "x": 0,
    "y": 0,
    "z": 0
  }
}
```

Save. Now let's check the 3D Scene:

- Did the arm appear in the 3D scene tab at the origin?
- How would you change the arm's position or initial Yaw angle?
- What other arm types does the simulated arm support?

## Part 3. Add the gripper (10 min)

The simulated arm is not very useful on its own — we also need a gripper. For this
workcell we are going to be using a Robotiq EPick vacuum gripper. There is a software
simulation of this gripper that models everything the real cup does (engage suction,
release, report whether a seal has formed), so we can build and test the cycle entirely
in sim before swapping to real hardware later in the course.

**+ Configuration Block**. Search `simulated epick`, pick
**robotiq-epick/simulated-epick-vacuum-gripper**. Click **add to machine**
and name the component `gripper`. The first time you add this model, Viam adds the
`robotiq-epick` registry module to your modules list automatically.

The gripper is not attached to `world` like the arm. Instead the gripper is attached to the `arm`. Notice the change for the `parent` attribute. We need to offset the gripper's frame **196 mm out from the arm flange (TCP)** — translation `(0, 0, 196)`. That puts the gripper's mesh in the right place relative to the arm and pins the gripper-frame at the vacuum tip, which is where every later motion-service call expects the EE to be.

**Frame:**
```json
{
  "parent": "arm",
  "translation": {"x": 0, "y": 0, "z": 196},
  "orientation": {
    "type": "ov_degrees",
    "value": {"x": 0, "y": 0, "z": 1, "th": 0}
  }
}
```

Save. Now let's check the 3D Scene:

- In the 3D scene, does the gripper render on the arm's end-effector?
- If you move the arm, would the gripper move with it?
- How would you rotate the gripper 90 degrees?

## Part 4. Add the Workcell Fragment (20 min)

Up to this point, we have been adding components one at a time. Often we find groups of components are used together, such as an arm, gripper, and camera combination or any number of other groupings. You can turn these into a reusable bundle called a fragment. A fragment can be a complete machine setup or just a few components and services used to help speed up commissioning. We are going to be importing the `example-palletizing-cell` fragment that contains a floor, pallet, pick-station, and a new component called a world state store.

First let's add the fragment:
**+ Configuration Block**. Search `example palletizing cell`, pick the `example-palletizing-cell` **Fragment**. Click **add to machine**. Do not forget to click **save** once added.

You should see 4 new items in the configuration view. Click on each item and view the included attributes and frames. These were defined in the fragment, but they are still configurable after import.

**Pallet:** This is a workcell component provided by Viam. You can specify the dimensions as attributes and the pallet will resize in the 3D scene. Along with the configurable geometry, it includes a few helper `DoCommand` actions that will be used later in the course.

**Attributes:** — Notice the pallet's dimensions are predefined:
```json
{
  "width_mm": 500,
  "length_mm": 350,
  "thickness_mm": 100
}
```

**Frame:**
```json
{
  "parent": "world",
  "translation": {"x": 200, "y": 500, "z": 100},
  "orientation": {
    "type": "ov_degrees",
    "value": {"x": 0, "y": 0, "z": 1, "th": 0}
  }
}
```

**Pick Station:** Similar to the pallet, this is an included workcell component provided by Viam. You can specify attributes about the pick-station, as well as the items being picked up. It includes a few helper methods that will be used later in the course. For now, it is important to know these can be predefined when creating a fragment, and can be overwritten at any time.

**Attributes:**
```json
{
  "lowest_point_height_mm": 200,
  "box_origin_offset_mm": {"x": 200, "y": 200},
  "box_theta_deg": 0,
  "pick_home_z_offset_mm": 120
}
```

**Frame:**
```json
{
  "parent": "world",
  "translation": {"x": 400, "y": -300, "z": 200},
  "orientation": {
    "type": "ov_degrees",
    "value": {"x": 0, "y": 0, "z": 1, "th": 0}
  }
}
```

**Floor:** This is a generic component that represents an obstacle. Notice the geometry attached to the world, which follows the same conventions as the gripper geometry. We must define this geometry to tell the motion planner about the ground. It is good practice to define the obstacles early as a safety step when creating a workcell. Similar to the gripper geometry, we define the box height as 10 mm and shift the translation down -5 mm.

```json
{
  "parent": "world",
  "translation": {"x": 0, "y": 0, "z": -5},
  "geometry": {
    "type": "box",
    "x": 2000,
    "y": 2000,
    "z": 10
  }
}
```

**World State Store:** This service is used to update visuals, such as the dynamic pallet graphics, and helps manage the `world state`. The world state keeps track of collision geometries for the motion service and is used when you have more than just static geometries. We will be using this to move simulated boxes during the palletizer motion.

**Attributes:**
```json
{
  "pick_station_names": ["pick-station"],
  "pallet_names": ["pallet"],
  "tick_interval_secs": 1.0
}
```

**example-palletizing-cell Fragment:** The fragment itself is also configurable. Here we can see the description, the resources that it provides, the version or tag to pin to, and any variables related to the fragment. In this fragment, no variables are needed, but you can create variables for anything machine dependent, like IP addresses, serial numbers, configuration file paths, etc.

**Viewing the Fragment:** After clicking on the example-palletizing-cell fragment, click on **View Fragment** in the top right hand corner. This will take you to the fragment configuration. This is the source for all machines using the fragment. Any changes you make here can automatically be applied to any machines using the fragment. The versions tab will show you which versions are available and how many machines are using each version.

**The 3D Scene**

Now that you have been introduced to the fragment and the supplied components, let's open the 3D scene tab and you should see:

- The arm at the origin.
- The gripper rendered on the end-effector (the EPick model carries its own mesh).
- The floor as a thin box below the arm.
- The pallet on the +Y side.
- The pick-station on the -Y side.

**Questions**:
- Can you see the arm, gripper, floor, pallet, and pick-station? (Note that the workcell-scene will not have a visual, as it is helping with the other visuals.)
- Could you make a ceiling or wall or keep out zone by following the floor as an example?
- When would you use a fragment? Are there any components that cannot be added to a fragment?

If something is missing, try refreshing the page. If still missing, check the logs or raise your hand and someone can assist.


## Part 5. Exercise the arm with the test cards (10 min)

Now that the cell is in place, let's test the simulated arm before we start writing code against it. In the Viam app, open the **CONTROL** tab; you'll see a test card for each configured component. Find the `arm` card. If your computer is able, open up the **3D SCENE** tab on the right side of the monitor and the **CONTROL** tab on the left.

### Read where the arm is

The card shows the arm's current **End Position** (for the UR5e, this is called the `arm flange`, and is displayed in world coordinates) and the current **Joint Positions** (six joint angles). Both update live.

Mark down:
- Current end position (XYZ + orientation):
- Current joint positions:

### Move to an end position

Use the **Move to position** form. Let's try:

- `x: 400`, `y: 0`, `z: 400`
- Orientation: `o_x: 0`, `o_y: 0`, `o_z: -1`, `theta: 0` (gripper pointing down)

Click **Execute**. Watch the 3D scene — the arm animates over a few seconds (if the arm moves too slowly, you can adjust the speed using the simulated arm's `speed` attribute from Part 2).

Send it to a second pose with the same orientation:

- `x: 300`, `y: 200`, `z: 300`

### Move to joint positions

The **Move to joint positions** form takes six angles and lets you set the joint positions directly. Try sending the arm to all zeros, then take a look in the 3D scene. This is a good time to get familiar with the working area of the robot arm. Change J0 to `90`, then `180`, then `270` (one at a time, keeping the other five joints at `0`). From here it should be apparent what the working area of the real robot looks like.

For safety, it is good to limit the range of motion to shrink the working area to only what is needed. Note down what you think a good upper and lower limit for J0 would need to be to reach both the pallet and the pick-station — we will use this later in a safety-limit step. Try out the other joints as well to get a feel for how each one moves the arm.

Record:

- A good upper / lower J0 limit to reach both the pallet and the pick-station. Does a J0 range of +60° to +300° look good?

### Stop a move

Start a move and click **Stop** mid-flight. The arm should freeze where it is.

- What error message / notification is printed? We will talk more about contexts in a future section.

### Reach for something unreachable

Send `x: 2000, y: 0, z: 400`. The motion service rejects it because the target is outside the arm's reachable workspace. The motion service should report an error in the response panel with some information about what went wrong. Switch to the **LOGS** tab and you should see a similar error from `plan_manager.go`.

- What error message / notification is printed?
- What about in the Logs tab?

## Part 6. Roll back a misconfig (5 min)

The machine configuration has built in version control to help you revert when something has gone wrong, or if you want to experiment with different components and services. Under the **CONFIGURE** tab, next to the `Save` button, is the history button (clock-with-arrow icon). This is your safety net. We are going to experiment with a few attributes and use the roll back feature.

Make two changes:

- In the pallet's **attributes** pane, change `width_mm` to `300` and `length_mm` to `200`. The pallet shrinks in the 3D scene as soon as you save.
- Update the pick-station's **frame** from the CONFIGURE tab — change its `translation` to a new position (any XYZ) somewhere away from where you placed it and Save.
- Now view your changes in the **3D scene**. You may need to refresh for the settings to take effect.

Once you are ready to "Undo", click **History** at the top of **CONFIGURE**, find the version from just before your two changes, click **Restore Version**, then **Save**. The pallet returns to 500×350 mm and the pick-station snaps back to `(400, -300, 200)`.


## Done when

You can answer **yes** to all of these:

- The CONFIGURE tab shows arm, gripper, floor, pallet, pick-station — all green.
- The 3D scene tab shows the arm, gripper, floor, pallet, and pick-station.
- You drove the arm to a couple of reachable poses from the CONTROL tab's test cards and watched the motion animate in the 3D scene.
- You used the History button to restore a deliberate break.

If any of these are no, capture the symptom and raise your hand.

## Takeaway

You have a cell. It's not doing anything yet, but the physical environment the palletizer will operate in is fully described:

1. An arm that can plan and move.
2. A gripper attached to the arm with a known stand-off.
3. A floor the planner will avoid.
4. A pallet positioned where boxes will go.
5. A pick-station positioned where boxes will arrive.

In the next section you will frame the palletizer module using the Viam Command Line Interface and add it to this cell!
