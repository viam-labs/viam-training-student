# Section 11: Framing the Palletizer Module

Time: 85 minutes.

In the previous section you built the cell — arm, gripper, floor, pallet, pick-station — all visible in the 3D scene tab. This section starts the code: you'll scaffold a Viam Go module named `palletizing-module` with a generic service named `palletizer` using the Viam CLI, add an arm and gripper reference, load the module on your cell, and watch `Validate` reject misconfigurations. 

## What you will do in this section

- Generate a Viam Go module from scratch with the Viam CLI
- Read the scaffold's file layout and explain what each file does
- Hot-reload Viam modules onto a viam-server
- Use `Validate` to fail fast on a missing dependency
- Read viam-server logs to find a `Validate` error after a bad save

## Setup check

Before you start, confirm:

- Your cell shows arm, gripper, floor, pallet, and pick-station all green; the 3D scene renders all five.
- VS Code is open, with the Go extension installed and the Go toolchain and Claude Code available in the integrated terminal. It's where you'll view the scaffolded files, watch builds, and run shell commands.
- `go version` reports Go 1.23 or higher. If your Go is older, ask your instructor for the install help.
- `viam login` succeeds — `viam organizations list` shows your org.
- You have an empty workspace directory ready at `~/training/palletizer/`.
- If your cell isn't intact (a component missing or red), you can restore the Section 10 cell from [`sections/section-11/prereq-machine-config.json`](./prereq-machine-config.json) via the Viam app's raw config editor.

## Part 1. Concepts recap (5 min)

In one sentence describe each:

- **Module:**

- **Model:**

- **Difference between a Component and a Service:**

- **`Validate` hook:**

Raise your hand if any of these feel uncertain.

## Part 2. Scaffold the module with the Viam CLI (15 min)

`viam module generate` is the Viam CLI's module scaffolder — it writes a complete, buildable Go module skeleton (entrypoint, service registration, build tooling) so you start from a working module instead of a blank directory. The full CLI module workflow is documented at [docs.viam.com/cli/build-and-deploy-modules](https://docs.viam.com/cli/build-and-deploy-modules/).

From your workspace directory in the IDE terminal, run:

```
viam module generate \
  --generate-type module \
  --name palletizing-module \
  --resource-subtype generic-service \
  --model-name palletizer \
  --language go \
  --public-namespace <NAMESPACE> \
  --visibility public_unlisted
```

What the flags mean:

- **`--name palletizing-module`** — This is the name of the module. Think of a module as a group or library that contains multiple models. For hardware, a module may be a brand like United Robots, or a topic like workcell-components.
- **`--model-name palletizer`** — The Model Name can be thought of as an individual unit inside the library. There can be multiple models inside a module. A physical Toolbox can have a hammer or a wrench. The model name describes the hammer or wrench.
- **`--resource-subtype generic-service`** — Viam comes with several existing types to help with modularity. It also provides generic *component* and generic *service* types as a good catch-all when you're building something that doesn't fit the existing structure. The palletizer orchestrates other resources (arm, gripper, motion) rather than driving a piece of hardware, so a generic **service** is the natural fit. The full list of component and service types is in the [APIs reference](https://docs.viam.com/reference/apis/).
- **`--public-namespace <NAMESPACE>`** — Identifies which org the module belongs to. The flag accepts either your org's namespace (e.g. `testing`) or its org-id UUID; both come from `viam organizations list`. Most commercial developers will have two organizations — a personal org and a company org — so make sure you're passing the one you want this module to live under.
- **`--language go`** — the CLI can scaffold modules in Go or Python; without this flag it prompts for the language interactively.

Record:

- Did you receive any errors? This should complete cleanly. If you have any issues, check the CLI inputs or raise your hand and an instructor can help you review.

**Build the module** — from the folder root, run:

1. `make setup` — runs `go mod tidy` for you, populating `go.mod`'s `require` block.
2. `make` — compiles the binary into `bin/palletizing-module`.

Record the result. We are expecting a clean build with no errors. If the build fails after `make setup` or `make`, raise your hand and an instructor can help.

### Tour the scaffold in the IDE

Open the module's workspace directory in VS Code's file tree. The scaffold produces roughly this layout:

```
palletizing-module/
├── bin/
│   └── palletizing-module # built binary (after `make`)
├── cmd/
│   ├── module/
│   │   └── main.go        # module entrypoint (Each model is added here for Viam to find)
│   └── cli/
│       └── main.go        # standalone test client for running from the CLI
├── DEVELOPER_GUIDE.md     # A provided guide for working with a new Viam Go module, with helpful tips for developing and logging.
├── go.mod                 # Go module + dependency list. This declares what you depend on. You read and edit this.
├── go.sum                 # This lists the expected checksums for the dependencies. It is a built in security check on all required libraries.
├── Makefile               # The instructions on how to build the library.
├── meta.json              # A description of the module, the models, supported platforms, and any supplied Viam apps. Think of this as the machine readable version of the README.md.
├── module.go              # The "meat" of your module. This is where most of the logic lives, including DoCommand and Validate.
├── README.md              # This file is the goto documentation for anyone looking at your module on github. It is good practice to add descriptions of the module and attributes for your models, along with example attributes.
├── <namespace>_palletizing-module_palletizer.md  # An additional markdown file used to describe attributes and do command verbs. This file is used to help others use your components and models.
└── .viam-gen-info         # This is similar to the parameters provided to the CLI when building the module.
```

Every module will follow a similar template. A few files can be renamed, depending on the entry point settings. We are going to be working heavily in a few key files. Let's jump in and look at some key areas:

**`module.go`** — Spend a few minutes scrolling through `module.go` — it's where every later section's DoCommand verbs, dispatch logic, and service state will land. Familiarity with the structure here pays off for future development; every Viam Go module follows this pattern.
- find `resource.RegisterService`: What method is this called in?
- find `resource.NewModel`: What are the 3 parameters?
- What methods return a "not implemented" error?

**`meta.json`** — This file has attributes for the module as a whole. You can create a better description that is visible on the Viam platform, add a URL to the matching github repo, add a script to be run once this module is loaded on a new machine, and specify what architectures this module is capable of running on, etc.
- What is the module visibility?

**`go.mod`** — This is the dependency file for Go. If you need to reference a new library, import it in your go program 
or manually add it to the require list and run `go mod tidy` (or just `make setup`) and Go will download the library and 
its dependencies for you. 
- What is the only required library? (Hint: There is only 1)
- What is the required version of Go?


## Part 3. Tell the module which arm and gripper to use (10 min)

The module we built from Part 2 compiles, but the module doesn't know about any hardware yet — its configuration is empty. The palletizer drives an arm and a gripper, so the module needs a configuration attribute for each, and it should fail fast if either is missing.

We'll edit `module.go` by hand for this one so you get a feel for where the configuration lives and how the `Validate` hook is shaped. Open `module.go` in the IDE.

### Rename the resource struct for readability

The scaffolder named your service struct after the module and model: `palletizingModulePalletizer`. You'll be reading and typing that struct (and its constructor) in every later section, so rename it to `palletizer` now, while the file is still small.

In VS Code:

1. Find the struct definition `type palletizingModulePalletizer struct {`. Click the type name and press **F2** (Rename Symbol). Type `palletizer` and press Enter — this updates the definition and every reference at once, including the method receivers and the `&palletizingModulePalletizer{...}` literal inside the constructor.
2. The scaffold uses `s` for the builder's local variable (the `s := &palletizer{...}` line in `NewPalletizer`) and as each method's receiver. For consistency with the later sections — which all use `p` — rename those: press **F2** on the `s :=` in `NewPalletizer`, and on each method's receiver `s` (there are only a few: `Name`, `DoCommand`, `Status`, `Close`). Each receiver is its own symbol, so that's one F2 per method.

Leave the two constructor **functions** named as generated: `NewPalletizer` (the builder where you'll add code in later sections — it's already nicely named) and the thin `newPalletizingModulePalletizer` wrapper that just forwards to it. Don't rename the `Palletizer` model variable from `resource.NewModel` either — that's a different symbol and must stay.

This is a pure identifier rename: the model triple (`<namespace>:palletizing-module:palletizer`) and the registered API don't change — only the struct name and the `s`→`p` variables move. Rebuild to confirm nothing broke:

```bash
make
```

A clean build means the rename is complete. If it fails, you probably renamed a reference by hand that Rename Symbol would have caught — undo and use **F2** instead.

### Add the config attributes

Find the `Config` struct near the top of the file — the scaffold leaves it empty with a comment showing the expected shape. Replace its body with two string fields, one for the arm component's name and one for the gripper's:

```go
type Config struct {
    Arm     string `json:"arm"`
    Gripper string `json:"gripper"`
}
```

The struct-field JSON tags (`json:"arm"` / `json:"gripper"`) are what map the JSON attributes in the machine config (`"arm": "arm"`, `"gripper": "gripper"`) to the Go fields in this struct.

### Fail fast in Validate

Find the `Validate` method on `*Config`. The scaffold's version returns `nil, nil, nil` — it accepts anything. Update it to reject a config that's missing either attribute, and to declare the arm and gripper as required dependencies:

```go
func (cfg *Config) Validate(path string) ([]string, []string, error) {
    if cfg.Arm == "" {
        return nil, nil, resource.NewConfigValidationFieldRequiredError(path, "arm")
    }
    if cfg.Gripper == "" {
        return nil, nil, resource.NewConfigValidationFieldRequiredError(path, "gripper")
    }
    return []string{cfg.Arm, cfg.Gripper}, nil, nil
}
```

Three things to notice:

- Returning a non-nil error from `Validate` stops the constructor from running. The service appears unhealthy on the CONFIGURE tab and the reason shows up in the LOGS — exactly the "fail fast" behaviour we want.
- `resource.NewConfigValidationFieldRequiredError(path, "arm")` is a helper that produces a consistent error string naming both the path and the missing field. For a module resource the `path` viam-server passes in is the resource's own name, so a missing `arm` reports `Path: "palletizer" Field: "arm"`. You could write `fmt.Errorf(...)` yourself instead; the helper just keeps the wording uniform across modules.
- The first return slice is the list of REQUIRED dependencies — resources the framework resolves and passes into the constructor. Returning `cfg.Arm` and `cfg.Gripper` here makes viam-server wait for the named arm and gripper components are ready before building the palletizer.

### Rebuild

Save the file and rebuild from the module's repo root:

```bash
make
```

Record the result. A clean build is what we want; if `make` fails, try giving the compile error to Claude Code. It is surprisingly good at helping you troubleshoot. Raise your hand if you're stuck.

## Part 4. Load the module on your cell with cloud hot-reload (20 min)

In Part 2 you scaffolded the module locally. Next we need to build and add the module to the Viam Registry. Viam is pretty flexible when it comes to building, and allows you to compile locally or in the cloud. With cloud build you can choose to build for multiple platforms and run several build checks, or build for a single platform for testing purposes.

### 4a. Add the module to the Viam Registry

First let's create a new entry for the module in the Viam Registry.

```bash
viam module create --name palletizing-module --public-namespace <NAMESPACE>
```

If you get `registry item with name ... already exists`, that's fine — it means the name is already registered (from a re-run, or because your org already holds it). Skip ahead to 4b; reload still works.

The module now has a name in the Registry, but no code behind it yet. The way you ship code to the Registry is a **release** — a versioned build of the module, tagged with a semantic version like `1.0.0`, that any machine in any org can deploy by selecting that version. Machines run on different CPU architectures (a Raspberry Pi is `linux/arm64`, most servers are `linux/amd64`, an Apple-silicon laptop is `darwin/arm64`), so a release carries a separate binary for each architecture it supports — the list of architectures you saw in `meta.json`. **Cloud build** produces those binaries for you: `viam module build start --version 1.0.0` compiles the module for every architecture in `meta.json` on Viam's build servers and attaches all of them to that one version, so each machine pulls the binary matching its own architecture and nobody cross-compiles locally.

### 4b. Hot-reload

For day-to-day development against your own machine, going through a release each iteration is slower than we want. The CLI provides a faster inner loop called **hot-reload**: `viam module reload` packages the module source, builds it in the cloud for your machine's architecture only, sends the binary to your machine, and hot-restarts the running module — one command, no release tag, no config edits. We'll use hot-reload for the rest of the palletizer build.

Hot-reload needs to know which machine to deploy to. Record:

- **Part ID** — Open the machine in the Viam app. Click the machine-status button in the header (its label reads **Live**, **Connecting**, or **Offline** depending on the current state) — click **PART ID** to copy it. You'll substitute it for `<PART_ID>` below.

Still in the module's repo root:

```bash
viam module reload --part-id <PART_ID>
```

This:

- Uploads `meta.json` and the module source to Viam's cloud builders.
- Builds the binary for your machine's architecture (no local cross-compile needed).
- Sends the resulting tarball to your machine.
- Adds the module entry to the machine config on the first run, or restarts the running module on subsequent runs.

The first run takes a minute or two for the cloud build. When it finishes, your `palletizer` model is installed on the machine and ready to add to the config via the **Modules** sidebar (see 4c).

For every later code change in this curriculum, just re-run the same command — `viam module reload --part-id <PART_ID>` — and the cloud rebuilds, pushes, and hot-restarts without you touching the machine config.

### 4c. Add the palletizer service

We haven't pushed a full release of the module to the registry yet (`viam module create` only registers the *name* — `viam module reload` ships a working copy to your one machine but not to anyone else's), so the top-level **+ Configuration Block** picker won't find our `palletizer` model. The easiest path right now is to add the service directly from the locally-installed module:

1. In the CONFIGURE tab's left sidebar, scroll to the **Modules** section near the bottom. Find the `palletizing-module` entry that `viam module reload` just installed.
2. Click the module to expand it. You'll see a list of available models — `palletizer` is the one we want.
3. Click **Add** next to `palletizer`. The dialog asks you to name the new service.
4. Name it `palletizer` and click **Add** to confirm.
5. **Save** the config.

You now have a palletizer service block, but it's missing the attributes the module needs to construct. Open the palletizer's card and add:

```json
{
  "arm": "arm",
  "gripper": "gripper"
}
```

With the `arm` and `gripper` attributes set, the dependency wiring is already done — your `Validate` returns those two names as required dependencies, so viam-server brings the arm and gripper up before the palletizer and hands them to the constructor. There's no separate `depends_on` to add. The whole palletizer block is just:

```json
{
  "name": "palletizer",
  "type": "generic",
  "model": "<NAMESPACE>:palletizing-module:palletizer",
  "attributes": {
    "arm": "arm",
    "gripper": "gripper"
  }
}
```

### Save and verify

Save the config. Open the LOGS tab. Record:

- Log line for the module process starting (look for `modmanager` or your module name):

- Log line for the `palletizer` service constructing — look for the `constructing` / `constructed` lines whose resource field is `palletizer`. viam-server logs the messages `Now constructing resource` and `Successfully constructed resource`; the resource name is a separate structured log field, not part of the message text, so a plain text search for `Name:palletizer` won't match:

Switch back to CONFIGURE; the `palletizer` service should appear in the resource graph with a green status. If it shows red or "Unhealthy", switch to LOGS and find the `Validate` error.

## Part 5. Break Validate on purpose (15 min)

A big part of commissioning and development is finding issues quickly. A few of the most common are attribute mismatches and module configuration issues — becoming familiar with these and how to spot them will help immensely. We'll go over a few here, what typically happens when they occur, and how to roll back when something does go wrong.

### Break 1: misspelled attribute name

Open the palletizer's service card. After the `arm` and `gripper` attributes, add an unrecognized attribute to the palletizer, e.g. `"extra_thingy": true`. Save.

Logs observation — does viam-server reject the unknown attribute, ignore it, or something else?

Viam-server's default JSON decoder ignores attributes it does not recognise — a silent forgiveness that helps modules survive forward-compatible config additions, but lets typos slip through. Later sections introduce typed wire shapes for DoCommand verbs where the decoder is strict and unknown keys are hard errors; the trade-off between the two approaches is part of the design choice every Viam-module author makes.

### Break 2: missing required attribute

Delete the `arm` attribute from its attributes pane. Save.

Watch the LOGS. Record:

- The exact error string viam-server emits:

- Where in the UI does the error surface? (CONFIGURE service card status? A separate banner? LOGS only?)

### Restore the good config

At the top of the CONFIGURE tab, click the **History** button (clock-with-arrow icon). Find the most recent version from before you started breaking — the one where the palletizer was green. Click **Restore Version**, then **Save**.

Watch the palletizer come back to green in the resource graph and the LOGS settle.

## Done when

You can answer **yes** to all of these:

- `make` from your module's repo builds clean.
- You have hot-reloaded your module onto your machine and successfully added the palletizer service.
- Your machine's CONFIGURE tab shows the palletizer service with a green status, alongside the cell.
- LOGS shows the module process started and the palletizer constructed cleanly.
- You've watched a `Validate` rejection in the LOGS (Break 2), then restored to green via the History button.

If any of these are no, capture the symptom and the LOGS line; raise your hand.

## Takeaway

The shape every later palletizer section follows is now in place:

1. A Go module hot-reloaded onto the cell via Viam's cloud-build pipeline.
2. `Validate` rejecting bad configs before the constructor ever runs.
3. The History → Restore Version flow as your recovery path when a misconfig breaks something.

Every later section ADDs to this — new DoCommand verbs, new state-machine states, new dependencies — without touching the scaffold. If you ever find yourself rewriting the constructor or the module's core plumbing, stop and let an instructor know something feels off.
