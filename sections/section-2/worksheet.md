# Section 2: Set up a Viam machine

Time: 45 min. Pair work; alternate driver and navigator each part.

This is your first hands-on with Viam. By the end you will have a host computer running viam-server, linked to a machine in the Viam app and showing live. The machine won't do anything yet: configuring components on it is Section 3.

## What you will be able to do after this section

- Install viam-server on a host computer and link it to a machine in the Viam app
- View logs
- Place a machine within the organization, location, and machine hierarchy

## Part 0. Pair setup (1 min)

Decide who drives first. The driver runs the keyboard and mouse. The navigator reads instructions aloud and watches the screen for mistakes. Parts 2 and 3 you do together; switch drivers for Part 4.

## Part 1. Watch the instructor stand up a machine (8 min, instructor-led)

Your instructor will stand up a machine from nothing. Watch for:

- Installing viam-server on the host computer
- Creating a machine in the Viam app and choosing its organization and location
- Connecting the machine and watching the status indicator turn green
- The machine page: the CONFIGURE tab (empty for now) and the LOGS tab
- What the LOGS tab shows right after viam-server starts

Take notes. You will be doing all of this yourself next.

## Part 2. Install viam-server on the host (10 min)

- SSH into the host computer using the credentials your instructor provides.
- Follow the install steps for viam-server on Linux from the Viam docs.
- Start the viam-server service.

Verify:

- The viam-server process is running on the host.
- The process logs are visible from the host.

Record:

- Command you used to install viam-server:

- Command to verify the service is running:

## Part 3. Create a machine in the Viam app and connect it (10 min)

- Open the Viam app. Use the organization and location your instructor specifies.
- Create a new machine. Choose a name.
- Follow the setup instructions in the Viam app to connect your machine.

Verify:

- Your machine appears live (green status indicator) in the Viam app.

Note the name you gave your machine. You'll need to return to this machine repeatedly throughout the training.

Record:

- Organization and location your machine lives in:

- Machine name:

## Part 4. Explore the machine (5 min)

Switch drivers.

- Open the LOGS tab. Read the entries from when viam-server started and connected.
- Open the CONFIGURE tab. It has no components yet; that is expected. This is where you will add components in Section 3.

Answer:

- What did the LOGS tab show right after the machine came up?

- How would you tell, from the Viam app alone, that your machine is live and reachable?

## Part 5. Closing reflection (3 min)

Answer in writing:

1. In one or two sentences, what is the relationship between the machine in the Viam app and the viam-server process on the host?

2. What does the green status indicator tell you?

3. Where does your machine sit in the organization and location hierarchy?

## Part 6. Quiz (8 min)

Answer on this worksheet. A lab instructor will check your answers at the end.

1. What had to be true for your machine to show green in the Viam app? Name at least two things.

2. What is the relationship between a machine in the Viam app and the viam-server process on the host?

3. What did the LOGS tab show right after viam-server started?

4. In one sentence, what was the Viam app's role in everything you did today?

When everyone finishes, we'll discuss as a group.
