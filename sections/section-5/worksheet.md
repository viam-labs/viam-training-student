# Section 5: The Viam CLI

Time: 85 min. Pair work; sit with your Section 2-4 partner. The CLI runs on your own laptop, so each of you installs it and logs in yourself, then pair up to compare output and help each other debug.

In Sections 2 and 3 you configured a machine through the Viam app, and in Section 4 you drove it from Python. The CLI is the third way in: the same operations and the same APIs, from your terminal. In this section you install the CLI, authenticate, set up a reusable profile the rest of the course depends on, and run the app's operations (status, logs, configuration, and the test card) from the command line. By the end the `viam` command alone is enough to inspect and drive your machine.

## What you will be able to do after this section

- Install the Viam CLI and authenticate interactively (`viam login`) and with an API key (`viam login api-key`)
- Look up the organization, location, machine, and part IDs the rest of the course relies on
- Set organization and location defaults and save credentials in a named profile
- View machine status and stream logs from the terminal
- Edit a resource and drive a component with `viam machines part run`, the test card from the command line
- Open a remote shell, copy a file, and tunnel a port, and recognize the `ViamShellDanger` fragment requirement

## Setup check

- You can reach your machine in the Viam app and it shows green.
- You have a terminal on your laptop (macOS, Linux, or Windows with WSL).
- Your instructor has told you the organization and location your machine lives in.

## Part 0. Pair setup (1 min)

Sit with your partner. Unlike earlier sections, you each work on your own laptop for the install and login, because the CLI lives on your machine, not the rig. Read each step aloud to each other and compare output as you go; debug together when one of you hits something the other didn't.

## Part 1. Watch the instructor drive a machine from the terminal (10 min, instructor-led)

Your instructor works from a clean terminal with no Viam app open. Watch for:

- Installing the CLI and running `viam login`
- `viam whoami`: which credential is active right now
- Looking up the organization, location, machine, and part IDs
- Setting org and location defaults, then saving a profile
- `viam machines status`, `viam machines part logs --tail`, and one `viam machines part run` call (the test card, from the terminal)

The app stays closed until the very end, when the instructor opens it as a side-by-side sanity check. The point: the CLI alone is enough.

## Part 2. Install, log in, and make a profile (15 min)

- Install the Viam CLI for your OS from the Viam docs. Confirm it with `viam version`.
- Run `viam login`. A browser opens; approve the request. Back in the terminal, run `viam whoami` and confirm it names you.
- Look up the IDs you will need all course:
  - `viam organizations list` and find the organization your instructor named.
  - `viam locations list` and find your location.
- Set defaults so you stop passing IDs on every command: set your default organization and your default location (look up the exact `viam defaults` subcommands with `viam defaults --help`).
- Create a machine-scoped API key for your machine with `viam machines api-key create` (run it with `--help` first to see the flags), and save it in a profile named `course` with `viam profiles add`.
- Confirm the profile works: `viam --profile=course machines list` should return your machine.

Verify:

- `viam whoami` names you, and `viam --profile=course machines list` lists your machine.

Record:

- Organization ID:

- Location ID:

- Machine name and part ID (from `viam machines list` and `viam machines part list`):

- The command you used to create the API key (without the key value):

## Part 3. Do your Section 2-3 app work from the terminal (20 min)

Everything you did in the app, you can do here. Use `--profile=course` (or your defaults) for these.

- `viam machines status` for your machine. How does it compare to the green indicator in the app?
- Stream logs with `viam machines part logs --tail`. In the app, save a small config change (or have your partner do it) and watch the restart appear in the stream. Stop the stream with Ctrl+C.
- Restart a part with `viam machines part restart` and watch the logs.
- Edit a resource attribute from the terminal with `viam resource update` (point it at inline JSON or a file). Change the arm's speed the way you did in Section 3's JSON view, and confirm the change in the app or the logs.
- Drive a component with `viam machines part run`, the test card from the terminal. Call a method on the camera, then on at least one other component. Do it twice: once with the short `--component` form and a short method name, and once with the full protobuf method path.

Answer:

- What did `viam machines part run` return for the camera, and how does that compare to the camera's test card in the app?

- What is the difference between the short `--component` form and the full protobuf method path? When would each be easier?

Record:

- The two `viam machines part run` commands you used (component form and protobuf path):

## Part 4. Shell, copy, and tunnel (15 min)

These three reach onto the machine's host. They need the `ViamShellDanger` fragment on the machine; your instructor has added it before this section.

- Open a remote shell on your machine's host with `viam machines part shell`. Run a harmless command (for example `ls` or `hostname`), then exit.
- Copy a file in and back out with `viam machines part cp`.
- Tunnel a port from the host to your laptop with `viam machines part tunnel`.

Answer:

- What did the shell service require to work, and why do you think it is gated that way?

- Name one thing each of shell, cp, and tunnel is good for during a real deployment.

## Part 5. Write a reusable CLI snippet (10 min)

Each of you writes a short shell snippet that:

1. Lists your machines.
2. Calls one method on one machine with `viam machines part run`.
3. Exits cleanly.

Commit it to your course repo. You will reuse it as the starting point for CLI scripting in later sections.

Record:

- The path where you committed your snippet:

## Part 6. Closing reflection (4 min)

Answer in writing:

1. In one or two sentences, what is the relationship between the Viam app, the SDKs, and the CLI?

2. What is the difference between `viam login` and `viam login api-key`, and when is each appropriate?

3. What does a profile hold, and why does the course want you to create one now?

## Part 7. Quiz (10 min)

Answer on this worksheet. A lab instructor will check your answers at the end.

1. Name the three interfaces to a Viam machine. What do they share underneath?

2. You ran `viam whoami`. What question does it answer, and why does it matter when you work across more than one organization?

3. What four IDs did you look up, and which command finds each?

4. `viam machines part run` is described as "the test card from the terminal." Explain why, and give the two ways to name the method you want to call.

5. `viam machines part shell` did not work until something was configured on the machine. What was it, and what is the risk that gating protects against?

When everyone finishes, we'll discuss as a group.
