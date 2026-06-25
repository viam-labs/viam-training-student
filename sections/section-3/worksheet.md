# Section 3: Configure components

Time: 80 min. Pair work; alternate driver and navigator each part.

In Section 2 you stood up a live machine. It doesn't do anything yet. In this section you give it capabilities: you configure three real components from registry modules, exercise each through its test card, edit a configuration directly in JSON, and swap one component's hardware to see that the API stays the same even though the hardware changes completely.

## What you will be able to do after this section

- Find modules in the Viam registry and configure components from them
- Navigate to any component on a running machine and exercise it through its test card
- View and edit a machine's JSON configuration directly
- Describe what happens when a configuration changes
- Explain hardware agnosticism: swap a component's hardware and confirm the API stays the same
- Recognize what a service is and how it depends on the components it uses

## Part 0. Pair setup (1 min)

Decide who drives first. The driver runs the keyboard and mouse. The navigator reads instructions aloud and watches the screen for mistakes. You will switch drivers at each component (Parts 2, 3, 4, 5, and 6). You work on the machine you created in Section 2.

## Part 1. Watch the instructor configure components (10 min, instructor-led)

Your instructor will configure and exercise components on a pre-built machine. Watch for:

- How each component appears in the CONFIGURE tab
- What a test card looks like for the arm, the camera, and the gripper
- The relationship between the form view and the JSON view of a configuration
- What happens in CONFIGURE and LOGS when the instructor changes a configuration value and saves
- Where modules come from when the instructor adds a new component

Take notes. You will be doing all of this yourself in the next hour.

## Part 2. Configure the UR3e arm (15 min)

First driver takes the keyboard.

- In CONFIGURE, browse the registry for arm modules. Find the one for Universal Robots arms published by Viam.
- Add the arm component. Read the README for the chosen model.
- Fill in the required attributes from the README. Use the network address your instructor provides for the UR3e.
- Save.

Watch the LOGS tab during the save.

- What did the logs show?

- Did the arm component come up green in CONFIGURE, or with an error?

Open the arm's test card.

- What information does the test card display about the arm's current state?

- Move the arm a small distance through the test card. What controls do you have access to? What considerations do you need to make before using them?

Record:

- Module name and publisher

- Model

- Required attributes you set

## Part 3. Configure the RealSense D435 camera (10 min)

Switch drivers.

- Find the RealSense module in the registry. Note the model name for the camera.
- Add the camera component. Read the README and fill in the required attributes.
- Save.

Open the camera's test card and start a video feed.

Record:

- Module name and publisher

- Model

- Required attributes you set

## Part 4. Configure the OnRobot RG2 fingered gripper (10 min)

Switch drivers.

- Find the OnRobot gripper module in the registry. Note the model name for the RG2.
- Add the gripper component. Read the README and fill in the required attributes.
- Save.

Open the gripper's test card.

- Use the controls to make the gripper open and grab.

- What did the test card show about the gripper's reported state?

Record:

- Module name and publisher

- Model

- Required attributes you set

## Part 5. Edit JSON directly and observe a reload (10 min)

Switch drivers.

- On the arm's test card, use the appropriate control to rotate the base joint 90 degrees from its current position. Pay attention to the time it takes to rotate through that arc. Move the arm back and forth a few times if necessary.
- Switch the CONFIGURE view from the form view to the JSON view.
- Find the arm's speed attribute, documented in its README. The unit is degrees per second.
- Increase the value. Try doubling it. Save.

Watch the LOGS tab during the save.

- What did the logs show?

- Did the arm component restart? Did any other components restart?

On the arm's test card, rotate the base joint 90 degrees back in the opposite direction. Time this motion.

- Was the second motion visibly faster?

Record:

- Attribute you changed:

- Old value:

- New value:

- Time for the first move (default speed):

- Time for the second move (increased speed):

- What the logs reported:

## Part 6. Swap the gripper hardware (10 min)

Switch drivers.

The gripper on your rig is about to change from a fingered gripper to a vacuum gripper. The hardware is completely different. Watch what happens to the way you control it.

- In CONFIGURE, switch to the JSON view and find the gripper component you configured in Part 4.
- Change the model from the OnRobot RG2 model to the OnRobot VGC10 vacuum gripper model. Keep the same component name. Update the attributes for the VGC10 as documented in its README. Your instructor will handle the physical gripper swap on the rig.
- Save.

Watch the LOGS tab during the save.

- Did the gripper component restart? Did the arm or the camera restart?

Open the gripper's test card.

- What controls does the VGC10 test card show? Are they the same controls you used for the RG2 in Part 4?
- Use Open and Grab. Does the vacuum gripper respond to the same commands the fingered gripper did?

Record:

- Old model:

- New model:

- The gripper test card controls before and after the swap:

- In one sentence: what changed, and what stayed the same?

## Part 7. From components to services, then reflect (6 min)

Everything you configured today is a component: a physical thing the machine controls. Real applications also need capabilities that are not tied to one piece of hardware: planning a path for the arm, finding an object in the camera feed, recording readings over time. In Viam those are services. A service runs on top of your components and consumes their APIs; it does not touch hardware directly. A vision service reads frames from a camera. The motion service plans for an arm using the frame system. The data service records from the components you point it at. The dependency runs one way: the service needs the component, not the reverse. You configure each of these where it first matters later in the course.

Answer in writing:

1. In one or two sentences, what is the relationship between the form view and the JSON view of a configuration?

2. When you saved your change in Part 5, which components restarted? Which did not?

3. In one sentence, what is a registry module, based on what you saw today?

4. In Part 6 you swapped the gripper from a fingered gripper to a vacuum gripper. What did you change to keep controlling it, and what stayed the same? What does that tell you about how Viam represents hardware?

5. A vision service finds objects in a camera feed. Which component would it depend on? If you removed that vision service, would the camera stop working? Explain the direction of the dependency.

## Part 8. Quiz (10 min)

Answer on this worksheet. A lab instructor will check your answers at the end.

1. Name the three components you configured today, the module that provided each, and the publisher of each module.

2. In one sentence each: what is the form view, what is the JSON view, what is the LOGS tab?

3. When you change a configuration value and save, what happens? Use what you saw in Part 5.

4. If a customer's hardware does not have a module in the registry, what are the options?

5. In Part 6 you swapped the fingered gripper for a vacuum gripper, and the gripper test card controls stayed the same. Explain why, in terms of component APIs.

6. You did not configure a service today, but you will soon. In one sentence, what is a service, and how does it relate to the components you configured?

When everyone finishes, we'll discuss as a group.
