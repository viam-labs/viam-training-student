# Section 6: Modules

Time: 100 min. Pair work; alternate driver and navigator each part, in the browser IDE.

In Section 5 you named two CLI commands without using them: `viam module generate` and `viam module reload`. This section is where they earn their keep. A module is how you add capability to Viam that the core software doesn't ship; here you build one. You'll meet the browser IDE, direct Claude to write the Go, scaffold a module, give it a `Validate` hook and a dependency, load it on your machine, and run the edit-reload-read-logs loop that is the rest of the course's development rhythm.

You will not write Go from a blank file. Claude does the typing; your job is to direct it in Viam's vocabulary, read its output critically, and verify each step through the app, the logs, and `viam machines part run`, the loop you started in Section 5.

## What you will be able to do after this section

- Say what a module is and why modules exist
- Read a `meta.json`, pin a module version, and explain what `latest` means
- Set up the browser IDE and prime Claude for Viam Go work
- Scaffold a module with `viam module generate` and direct Claude to implement a small generic component
- Use the `Validate` hook and `depends_on` so a module fails fast on misconfiguration
- Configure a local module and run the edit-reload-logs loop, choosing between `viam module reload` and `reload-local`

## Setup check

- Your `course` profile from Section 5 works: `viam --profile=course machines list` returns your machine.
- You can open the browser IDE your instructor assigned. It ships with the Go toolchain, Claude Code, viam-server, and the reference repos pre-cloned (`rdk`, `viam-python-sdk`, `Palletizing-Module`, `robotiq-epick`), plus a starter `CLAUDE.md`.
- Your machine shows green in the Viam app.

## Part 0. Pair setup (1 min)

Decide who drives first. Switch drivers at each part. You build one module together; both of you should drive at least one of Parts 4 through 7.

## Part 1. Concepts recap (5 min)

In one sentence each:

- **Module:**

- **Model (vs module):**

- **Driver vs control logic:** (hint: the UR, RealSense, and OnRobot models you configured in Section 3 are drivers; the thing you build today orchestrates other resources)

- **Why modules exist (what would you do without them?):**

Raise your hand if any of these feel uncertain.

## Part 2. Watch the instructor build a module with Claude (15 min, instructor-led)

Your instructor builds a small module live, in the browser IDE, by directing Claude. Watch for:

- How the instructor primes Claude: pointing at the pre-cloned reference repos, the course `CLAUDE.md`, and the running machine's resource list
- The Viam vocabulary in the prompts (generic component, `Validate`, `depends_on`, the test card) and how it changes what Claude writes
- At least one moment where Claude gets something wrong (a hallucinated API, a wrong `meta.json` shape) and the instructor course-corrects
- The verification loop: build, reload, read the logs with `viam machines part logs --tail`, exercise it with `viam machines part run`

The point is not a polished result. It is the verify-then-correct rhythm you will use yourself in Part 5.

## Part 3. Open the IDE and prime Claude (10 min)

- Open the browser IDE. Confirm `go version` reports Go 1.23 or higher and `viam version` works.
- Open Claude Code in the IDE. Give it the context it needs before you ask for anything:
  - Point it at the pre-cloned reference repos and the course `CLAUDE.md`.
  - Tell it you are building a Viam module in Go, and hand it your machine's resource list (from `viam machines part run` or the app) so it knows what is on the machine.
- Confirm the framing with a question, not a code request: ask Claude to explain, in a few sentences, the difference between a driver and a control-logic generic component, and which one you are about to build. Read the answer. If it is wrong about Viam, that is your signal to give it more context before you ask for code.

Record:

- One sentence: what did you point Claude at, and how did you confirm it understood the task?

## Part 4. Read a registry module and its meta.json (10 min)

Before you build one, read one.

- In the Viam app's registry browser (or `viam module` on the CLI), open one of the modules you configured in Section 3 (for example the OnRobot or RealSense module).
- Find and read its `meta.json`. Identify: the module ID, the models it provides, and the supported platforms.
- Look at its versions. Note the difference between pinning a specific version and tracking `latest`.

Answer:

- What models does the module provide, and what is its module ID?

- If you pin `latest`, what happens to your machine when the publisher ships a new version? When would you pin a specific version instead?

## Part 5. Scaffold and build your module with Claude (20 min)

You will build a small generic component that depends on one resource your machine already exposes (your instructor names which; the example below uses the camera) and exposes two or three DoCommands that combine a read with an action.

- Scaffold the module with `viam module generate` (let Claude run it, or run it yourself and hand Claude the result). Note the layout it produces: the entrypoint, `meta.json`, `go.mod`, and the file where the model's logic lives.
- Direct Claude to implement the component. A prompt in your own words, naming what you want:
  - the component depends on the camera (declare it with `depends_on` so the module waits for the camera before it builds);
  - a `Validate` hook that rejects the config if the dependency is missing, so the module fails fast;
  - two or three DoCommands, for example a `status` that returns "ready" plus the dependency names, and one that reads from the camera and returns something about the frame.
- Read Claude's output before you build. Skim the `meta.json` (does the model triple look right?), the entrypoint (is the model registered?), and the `Validate` and DoCommand code. If something looks like a hallucinated API, say so and have Claude fix it.
- Build it. If the build fails, paste the specific compile error back to Claude; it is good at fixing its own output.

Record:

- The model triple Claude registered (`namespace:module:model`):

- One thing you had to course-correct from Claude's first attempt (if nothing, write "clean first pass"):

## Part 6. Load it locally and iterate (15 min)

- Configure your module as a local module on your machine, then push your build with `viam module reload-local` (local build, no cloud round-trip). Watch it install in `viam machines part logs --tail`.
- Add the component to your machine's config with the dependency wired in. Confirm it comes up green.
- Exercise it: call each DoCommand with `viam machines part run` (the test card from Section 5). Confirm the read-plus-action DoCommand returns what you expect.
- Now iterate: have Claude make one small change (add a field to a DoCommand response, say), then `viam module reload-local` again and re-run. This edit-reload-read loop is the rhythm for every later module section.

Answer:

- What is the difference between `viam module reload` (cloud build) and `viam module reload-local` (local build)? Which did you use, and why is it the right one for the browser IDE?

Record:

- The `viam machines part run` call you used to exercise a DoCommand, and its response:

## Part 7. Break Validate on purpose (10 min)

- In your machine config, remove the dependency attribute the module requires (the camera name), or mistype it. Save.
- Watch the logs. The component should go unhealthy and `Validate` should reject the config with a clear error naming the missing field, before the constructor ever runs.
- Restore the config (the History button, as in Sections 2 and 3) and watch it come back to green.

Answer:

- What exact error did `Validate` produce, and where did it surface (the component card, a banner, the logs)?

- Why is failing in `Validate` better than letting the component build and crash later?

## Part 8. Course-correction log and closing reflection (5 min)

- Write down the one or two things you had to correct from Claude's output today. These feed the class playbook for building Viam modules with Claude.

Then answer in writing:

1. In one or two sentences, what is a module, and what does `meta.json` describe?

2. What does `depends_on` do, and what does `Validate` add on top of it?

3. You built Go today without writing it from scratch. What was your job, and how did you know Claude's output was right?

## Part 9. Quiz (10 min)

Answer on this worksheet. A lab instructor will check your answers at the end.

1. What is a module, and why do modules exist (what do they let you do without changing Viam's core)?

2. Name three things a `meta.json` describes.

3. What is the difference between `viam module reload` and `viam module reload-local`, and when would you pick each?

4. What does the `Validate` hook do, and why is failing there better than failing in the constructor or at runtime?

5. You directed Claude to write the Go. Name one kind of mistake Claude makes on Viam work, and how you caught it.

When everyone finishes, we'll discuss as a group.
