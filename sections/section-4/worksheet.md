# Section 4: SDKs

Time: 80 min. Pair work; alternate driver and navigator each part.

In Sections 2 and 3 you built and exercised a machine entirely through the Viam app: you configured components and drove them through their test cards. In this section you do the same things from code. You connect to your machine from Python in JupyterLab, list its resources, read a camera frame, actuate a component, and call the vision service, all through the same APIs the test cards use. By the end you will have seen that the app, the test cards, and your Python script are three ways of calling one set of APIs over one connection.

## What you will be able to do after this section

- Name the Viam SDKs and recognize the CLI and the app as sibling interfaces to the same APIs
- Connect to your machine from Python with an API key
- List the resources on your machine and match them to what the app shows
- Find a component or service method in the Python SDK reference and call it
- Read a camera frame and display it in a notebook
- Actuate a component and call the vision service from code
- Recognize that the test cards, the app, and your script all reach the machine over the same gRPC and WebRTC connection

## Part 0. Pair setup (1 min)

Decide who drives first. The driver runs the keyboard. The navigator reads instructions aloud, watches the screen, and keeps the Python SDK reference open in a browser tab. You will switch drivers at each part. You work against the machine you configured in Sections 2 and 3: the one with the UR3e arm, the RealSense D435 camera, and the gripper.

## Part 1. Watch the instructor connect from code (8 min, instructor-led)

Your instructor will connect to a machine from a Python notebook on the projector. Watch for:

- Where the connection code comes from: the machine's CONNECT tab in the Viam app, the SDK code sample page, Python
- The two credentials in the snippet: the API key and the API key ID
- The machine address
- What `machine.resource_names` prints, and how that list matches the components and services in the CONFIGURE tab
- How the instructor finds a method in the Python SDK reference and calls it

Take notes. You will do all of this yourself next.

## Part 2. Connect to your machine and list resources (15 min)

First driver takes the keyboard. Open JupyterLab and start a new notebook.

- In the Viam app, open your machine, go to the CONNECT tab, open the SDK code sample page, and choose Python. Copy the connection snippet. It already contains your machine's address and an API key your instructor provisioned.
- Paste it into a notebook cell and run it. The snippet connects and prints `machine.resource_names`.

Read the printed resource list.

- How many resources are listed?

- Find the arm, the camera, and the gripper in the list. What is the exact name of each, as your script sees it?

- Are there any resources in the list that you did not configure yourself? Look for built-in services.

Record:

- Your machine address (host only, not the API key):

- The number of resources returned:

- The names of the arm, camera, and gripper:

## Part 3. Find a method in the reference and read a camera frame (15 min)

Switch drivers. Keep your connection from Part 2 open. You will now read one image from the camera.

- Open the Python SDK reference. Find the camera client and the method that returns the camera's current image or images.
- In your notebook, get a handle to your camera by its name, call that method, and display an image inline in the notebook. If the method returns more than one image, display the first.

- What is the name of the method you called?

- What type did it return? Read the reference, or print the type.

Record:

- The camera method you used:

- One line: how did you find it in the reference?

## Part 4. Actuate a component and call the vision service (15 min)

Switch drivers. Two short tasks against the same connection.

First, actuate a component:

- Get a handle to the gripper by name. Find the method that opens it and the method that grabs. Call open, then grab, and watch the gripper on the rig.

Then call the vision service:

- Your machine has a vision service configured with a pretrained box detector. Find it in your resource list from Part 2.
- Get a handle to the detector. Find the method that returns detections from your camera. Call it and print the detections.

- How many detections came back?

- For the highest-confidence detection, what is the class name and the confidence?

Record:

- The gripper methods you called:

- The vision method you called:

- The highest-confidence detection (class and confidence):

## Part 5. Find the connection in browser dev tools (8 min)

Switch drivers. Your script reached the machine over the same transport the app uses: gRPC for the API calls, WebRTC for the connection itself. You will now see that connection in the app.

- In a browser, open your machine in the Viam app. Open the browser's developer tools and find the network or WebRTC view. Your instructor will point to the right panel for your browser.
- Find the active connection to your machine.

- What does the developer tools view tell you about the connection?

- In one sentence: your Python script and the app both reach the machine. What do they have in common underneath?

## Part 6. Close cleanly, and one more client (5 min)

Switch drivers.

- Look at the connection snippet from Part 2. It wraps the connection in `async with await connect() as machine:`. In one sentence, what does the `async with` do when the block ends?

- The client you have used all section is a RobotClient: it talks to one machine. There is a second client, ViamClient, that talks to the Viam cloud and platform (data, fleet, the organization). You will use ViamClient in later sections. For each task below, write R (RobotClient) or V (ViamClient):

  - Read the current image from your camera: \_\_\_
  - List every machine in your organization: \_\_\_
  - Move the arm to a pose: \_\_\_
  - Download captured data from the cloud: \_\_\_

## Part 7. Closing reflection (4 min)

Answer in writing:

1. In one or two sentences: what is the relationship between a test card and your Python script?

2. You connected with two credentials. Name them, and say in one line what each is for.

3. In one sentence, what is the difference between RobotClient and ViamClient?

4. Name two of the Viam SDKs other than Python.

## Part 8. Quiz (10 min)

Answer on this worksheet. A lab instructor will check your answers at the end.

1. List the Viam SDKs. Name one other interface that calls the same APIs.

2. What two credentials does the Python connection snippet need, and where in the app did you get them?

3. You called `machine.resource_names` and saw resources you never configured. Where did those come from?

4. Your script, the app, and the test cards all reach the machine. In one or two sentences, what transport do they share, and what does WebRTC do for that connection?

5. You read a camera frame in Part 3 and called the vision service in Part 4 without writing either method from scratch. How did you know what to call?

When everyone finishes, we'll discuss as a group.
