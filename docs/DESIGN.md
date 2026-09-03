# SUNDER

**A C2 framework whose operations are composed, rehearsed, and recited.**

Sunder is a cross-platform command-and-control framework. It speaks its own
language: every command is a *Word* with a poetic meaning behind it, and an
engagement is a *Saga*, an executable and auditable composition.

> **The soul of the project.** The poetry is not decoration; it is the design
> principle. Every command word is chosen so that its metaphor is also the
> technically exact name for the action. The vocabulary is literary and
> archaic English, and the register is consistent to the last word.

---

## 1. Overview

Sunder consists of three parts:

- **Overseer**: the controller (the *hand*). A custom operator console that
  runs on your machine (Linux, macOS, Windows). Domain commands plus a thin
  local bridge to your real shell.
- **Wraith**: the implant (a *Shard* cast from your blade). A memory-resident,
  cross-platform agent written in Rust. Communicates over encrypted channels
  and executes Words on the target, returning Unix-style output regardless of
  the underlying OS.
- **Fragments**: loadable WebAssembly modules pushed to a Wraith at runtime.
  The ecosystem layer: capabilities are modules, not binary logic.

### Architecture

```
+----------------+          Whisper (encrypted)         +----------------+
|   Overseer     | <------------------------------------> |    Wraith      |
|  (Controller)  |   HTTPS / DNS / Ravens (dead drop)   |   (Shard)      |
|    Go console  |                                        |   Rust agent   |
+----------------+                                        +----------------+
        |                                                          |
        |  Local passthrough to                                     |  WASM Fragment
        |  host shell (real ls, tar, ...)                           |  runtime
        v                                                           v
   your machine                                             target system
   (bash/zsh/fish)                                    (Windows/Linux/macOS)

+----------------+
|    Chorus      |  optional mesh: Shards relay for each other,
+----------------+  operations continue with no reachable server
```

### Naming

| Term | Meaning |
|---|---|
| **Sunder** | to split apart; the framework sunders a target from its secrets |
| **Overseer** | the controller; the one who watches and speaks |
| **Wraith** | the implant; a Shard of the blade |
| **Shard** | a deployed implant |
| **Fragment** | a WASM module loaded into a Shard |
| **Whisper** | the encrypted comms layer |
| **Chorus** | the peer mesh of Shards |
| **Raven** | a dead-drop resolver (gist, DNS TXT, mail) |
| **Oracle** | the AI copilot (offline-capable) |
| **Auditor** | the OPSEC conscience |
| **Ledger** | engagement evidence, findings, timeline, reports |

---

## 2. The command language

Commands are **Words**. Categories are coherent metaphors. The structure of an
operation is the structure of a poem:

| Tier | Name | Meaning | What it is technically |
|---|---|---|---|
| **Word** | a single command | the smallest utterance | one task to one Shard |
| **Line** | a step in a plan | one verse | a Word plus condition, target set, and checkpoint |
| **Canto** | an operation | a division of a long poem | a choreographed sequence of Lines across Shards, with branching and rollback at Line boundaries |
| **Saga** | a full engagement | an epic tale | the entire engagement: Cantos, findings, evidence, timeline |

### Design rules for the lexicon

1. The best name is one whose metaphor is also technically exact
   (`neighbors` = ARP table; `eavesdrop` = keylogger; `immolate` = self-destruct).
2. When the metaphor is exact, poetry becomes a mnemonic system: operators
   learn the whole surface through one coherent story.
3. `?` and `help` gloss every Word: the poetic meaning and the action.
4. Every Word has a conventional alias (`ls`, `ps`, `who`, `netstat`, `curl`,
   `kill`, and so on) so operators never have to unlearn muscle memory.
5. Tab completion, aliases, and the Oracle's natural-language mapping keep the
   poetry from ever blocking usability.
6. The vocabulary is literary and archaic English, and the register never breaks.

### Command-line conventions

- `-h` help on any Word; `?` categorized help; `help` flat list with glosses.
- Continuous operations use `-s` start and `-x` stop (keylog, streams, litany).
- Timing uses `-t` time or duration, `-i` interval, `-n` count.
- Output uses `-o` path; results stream to the Ledger by default.
- Destructive Words (`immolate`, `slumber`, `ablution`, `erase`) demand
  confirmation; every Denial Word demands double confirmation plus
  `scope: impact` (§3.12).
- Every Word carries an Auditor risk score (low, medium, high), shown by `?`.

### The console's voice (flavor rules)

The Overseer speaks back in the project's register. The console has a voice
consistent with the lexicon: dry wit where the risk is low, gravity where it
is not. Rules:

- `tone` sets the voice: `verse` (default, flavored), `plain` (no flavor,
  tabular, for real work), `quiet` (errors only). A global `--json` flag
  forces machine-readable output.
- **Flavor never touches data.** Decoration and color appear only on a real
  TTY and flatten automatically when piped. Structured output (`--json`,
  `plain`) is never polluted; personality cannot break scripting.
- **Gravity scales with Auditor risk.** Low-risk Words may be dry. `breath`
  might answer *"It breathes, shallow and patient."* High-risk Words turn
  grave. `immolate`'s confirmation reads like an epitaph, and Denial Words
  are formal and foreboding.
- Every flavored line stays true to its Word's metaphor. No generic quips.

Sample register:

| Moment | Voice |
|---|---|
| `breath` breathing | "It breathes, shallow and patient." / "Alive. It dreams in page faults." |
| `breath` silent | "It holds its breath." |
| `breath` dark | "Silence. The wire is cold." |
| `grasp` success | "You have its ear." |
| `shards` empty | "No shards. The blade is whole, for now." |
| `cast` complete | "A shard leaves the blade." |
| `still` done | "It is still." |
| `litany` stopped | "The litany ends." |
| `recite` open / close | Canto opens and closes with its own stanzas |
| `immolate` confirm | "Once the flame takes it, there is no calling it back." |
| `immolate` done | "It is ash." |
| `cataclysm` before | "The machine will not remember this." |
| `voice` opened | "You have its voice." |
| `murmur` queued | "It will hear you soon." |
| `anchor` set | "Anchored. It will not drift." |
| `drift` done | "It drifts free." |
| `wind` set | "The clockwork is wound." |
| `sow` landed | "A seed takes root." |
| `burrow` open | "A tunnel opens." |
| `doom` set | "Its end is written." |
| `knock` answered / silent | "Someone answers." / "No answer." |

---

## 3. The Word reference

### 3.1 Core: the console itself (Overseer-side)

| Word | Function |
|---|---|
| `?` / `help` | Categorized or flat command reference; glosses the poetry and the action |
| `clear` | Clear the console |
| `echoes` | Console history: past utterances, searchable |
| `annals` | The operation log: a chronological record of every Word and result |
| `marginalia` | Engagement notes: global or per-Shard annotations |
| `folio` | Save and load engagement state; snapshot and restore a Saga mid-engagement |
| `signal` | Whisper channel stats per Shard: latency, last breath, transport in use, retries |
| `probe` | Test a listener or transport from the Overseer side (no Shard involved) |
| `trove` | Browse the local trove: everything lifted off targets |
| `!` | Local passthrough: execute a command on the operator's own machine via the real shell (`!df`, `!tar`) |
| `coven` | Team mode: operators, roles, shared sessions (v3) |
| `ledger` | Engagement data: scope, targets, findings, evidence, timeline; `ledger report` generates the red-team report |
| `keepsake` | Capture evidence (screenshot, file, output) into the Ledger |
| `tone` | Set the console's voice: `verse` (default, flavored), `plain` (no flavor), `quiet` (errors only) |

### 3.2 Casting: payloads and listeners

| Word | Function |
|---|---|
| `cast` | Deploy a Shard: `cast --os linux --arch amd64 --transport https`; beacon config baked in |
| `veil` | Obfuscation options at cast time: strip, pack, encrypt strings, hide the blade |
| `sigil` | Code-signing config for the cast binary (certificate, timestamping) |
| `lighthouse` | Manage listeners: start, stop, status of HTTPS, DNS, and Raven channels |
| `quarry` | Fetch Fragments from the module registry: extract new capabilities |

### 3.3 Scrying: seeing the unseen (recon)

| Word | Function |
|---|---|
| `anatomy` | System info: OS, kernel, hostname, arch, boot time, domain or workgroup, firmware (`--deep`) |
| `pulse` | Process list (`-a` all, `-m` by memory); `pulse watch` for a live top-like view |
| `breath` | Heartbeat check: is the Shard alive |
| `presence` | Logged-in users, sessions, login history |
| `name` | Current user context: user, domain, integrity or uid (`whoami`) |
| `climate` | Environment variables (`env`) |
| `vigil` | Uptime and boot time: how long the machine has kept watch |
| `meridian` | Time, timezone, clock skew |
| `bearings` | Geolocation: public IP geo and locale of the Shard |
| `sinew` | Hardware inventory: CPU, RAM, disks, GPU, model |
| `mind` | Memory usage and available: how much of the machine's mind is free (`free`) |
| `hold` | Disks and mounts or volumes (`-a` all) (`df`) |
| `keel` | Loaded drivers and kernel modules: the structural layer (`lsmod`) |
| `loci` | Network interfaces and addresses (`-4` and `-6`) (`ip`) |
| `paths` | Routing table (`route`) |
| `neighbors` | ARP/NDP neighbor table: who shares its street (`arp`) |
| `threads` | Active connections (`-t` TCP, `-u` UDP, `-p` per-process, `-s` listening) (`netstat`) |
| `gates` | Port scan of the target (`--top` top ports, `-r` range): every gate into the machine |
| `knock` | Probe a single port or address: connectivity plus banner grab (`nc`) |
| `glimpse` | Screenshot (`-o` file, `-s` live stream) |
| `visage` | Webcam capture (`-s` start or stream, `-x` stop): capture the face |
| `echo` | Microphone capture (`-s` record, `-x` stop, `-t` duration) |
| `eavesdrop` | Keylogger (`-s` start, `-x` stop, `-d` dump): listen at the wall |
| `trail` | Recently accessed files (`-n` count, `-t` hours): where it has walked |
| `chronicle` | Browser history (`-b` browser, `-n` count): its written record |
| `clockwork` | Scheduled tasks (`-a` all, `-d` detail): what it has planned |
| `daemons` | Services and daemons (`-a` all, `-d` detail): its silent servants |
| `watchers` | Installed AV and EDR: processes, drivers, services, who is watching |
| `wards` | Firewall rules and state (`-l` list, per-profile) |
| `rank` | Integrity or privilege level (Windows); uid, gid, capabilities (Unix) |

> **Note on `eavesdrop` and the platform walls.** The honest map of what a
> keylogger can hear changes with the display stack. An X11 session exposes
> every keystroke to any client: the XRecord extension was built as a session
> recording facility, and no grab or focus is required. Wayland was designed
> to end that. The compositor owns input and delivers keys only to the
> focused client, so no client API can hear the desktop, and a global
> keylogger is impossible by protocol design. Under XWayland, capture
> reaches only other X11 clients, and only while one of them holds focus.
> The captures that remain on a Wayland desktop are evdev as root (below the
> whole display stack), an input method the user consented to, or an X11
> holdout. `eavesdrop` must detect which world it landed in and report what
> it can actually hear, and say plainly when the wall holds.

### 3.4 Incantations: Execution (speaking things into being)

| Word | Function |
|---|---|
| `utter` | Run a shell command and return Unix-style output (`-t` timeout, `-c` capture stderr, `-e` env) |
| `voice` | Open an interactive shell session on the target (`-t` shell: cmd, powershell, bash, zsh; default per-OS) |
| `voices` | List open voice sessions per Shard: id, shell, duration, status |
| `attend` | Attach to an existing voice session: resume it, watch or take over (coven) |
| `murmur` | Open an async shell conversation: commands queue, output delivered at next check-in |
| `birth` | Spawn a process (`-d` detached, `-u` as user) |
| `still` | Terminate a process (`-f` force, `-n` by name): make it still (`kill`) |
| `rouse` | Start, stop, or restart a service or daemon (`-s`, `-x`, `-r`) |
| `wind` | Create or delete scheduled tasks: wind the clockwork (`-a` add, `-d` delete, `-t` trigger) |
| `litany` | Repeat a Word on an interval until it ends (`-i` interval, `-n` times, `-x` stop): a recited refrain (`repeat`) |
| `portent` | Schedule a one-shot Word for a future time: a sign of what will happen |
| `respite` | One-off dormancy: hold the Shard silent for N seconds before its next check-in (`-s` seconds); the ongoing beacon schedule lives in `cadence` (§3.6) |
| `slumber` | Shut down the target (`-f` force, `-t` delay) |
| `reawaken` | Restart the target (`-f` force, `-t` delay) |
| `beckon` | Open a URL in the target's default browser |
| `dispatch` | Make an HTTP request from the target (`-m` method, `-b` body, `-H` headers, `-o` save) (`curl`) |
| `ascend` | Attempt privilege escalation to admin, SYSTEM, or root (audited vectors) |
| `mask` | Impersonate a token or credential context (Windows token impersonation; `-u` user) |
| `guise` | Run a command as another user (`-u` user, auth source) |
| `muzzle` | Patch AMSI or ETW (Windows) to silence defensive telemetry (`-a` amsi, `-e` etw, `-r` revert) |
| `ablution` | Clear or scrub target traces: event logs, shell history, browser artifacts (`-l`, `-h`, `-t`) |
| `antedate` | Timestomp file timestamps (`-c` created, `-m` modified, `-a` accessed) |

### 3.5 Incantations: Files and Data

| Word | Function |
|---|---|
| `gaze` | List directory (`-a` all, `-l` long, `-r` recursive) (`ls`) |
| `unfold` | Read a file (`-o` offset, `-n` bytes, `-t` text, hex, or base64) (`cat`) |
| `seek` | Find files by name or glob (`-n` name, `-t` type, `-s` size, `-d` since) (`find`) |
| `sift` | Search file contents (`-i` case, `-r` recursive, `-n` line numbers, `-c` count) (`grep`) |
| `lift` | Download file(s) into the local trove (`-r` recursive, `-z` bundle, `-e` encrypt) |
| `bestow` | Upload file(s) to target (`-o` destination, `-m` mode, `-v` verify hash) |
| `bear` | Move a file or directory: carry it hence (`-f` force) (`mv`) |
| `mirror` | Copy a file or directory (`-r` recursive) (`cp`) |
| `seal` | Compute hashes for integrity (`-a` algorithm) (`sha256sum`) |
| `nest` | Create a directory: build a nest (`mkdir`) |
| `spark` | Create an empty file: spark it into existence (`touch`) |
| `erase` / `unmake` | Delete a file or directory (`-r` recursive, `-f` force) (`rm`) |
| `gossip` | Read the clipboard: what the apps whispered to each other |
| `inscribe` | Set the clipboard text |
| `scroll` | Bundle trove contents into an encrypted archive for exfil (`-e` encrypt, `-p` password) |
| `courier` | Exfil a bundle or file through an external channel (HTTP, DNS, mail, Raven) (`-c` channel, `-d` destination) |

### 3.6 Grasp: the fleet and sessions

| Word | Function |
|---|---|
| `shards` | List deployed Shards: id, host, user, os, transport, last breath, notes (`-a` all) |
| `grasp` | Hook into a Shard; subsequent Words target it (`-l` list, id or host) |
| `gather` | Select multiple Shards for a session or operation (`-g` group) |
| `sermon` | Broadcast a Word to all Shards or a group: deliver the sermon |
| `christen` | Rename a Shard: give it a name |
| `brand` | Tag a Shard (`red`, `dc`, `priority`) |
| `cadence` | Set beacon interval and jitter for a Shard (`-s` seconds, `-j` jitter) |
| `doom` | Set or clear the kill date for a Shard: when it must die |
| `patience` | Set the retry and backoff policy: max retries before the Shard goes quiet |
| `fragments` | List, load, or unload Fragments on a Shard (`-l`, `-a` load, `-u` unload) |

### 3.7 Anchors: persistence and lifecycle

| Word | Function |
|---|---|
| `anchor` | List current anchors; `-m` adds one (mechanism: task, wmi, service, startup, launchd, registry); `--disguise` camouflages it with a legit-looking name (Auditor scores higher) |
| `drift` | Remove persistence: `-m` one anchor, bare `drift` releases all; verifies removal (reads back, confirms no trace) |
| `immolate` | Self-destruct as a procedure: drift, then purge per Auditor's ledger, then confess (final trace report to the Overseer), then burn (shred keys, terminate irreversibly). `--force` skips verification. Double confirmation. Queued: a dark Shard immolates at next check-in |

Anchors are first-class: a Shard may hold several, each logged in the
Auditor's footprint ledger (mechanism, name, location) so `drift` and
`immolate` are precise. The tethered prompt state signals the accepted
on-disk footprint.

**The userland covenant.** `anchor` answers one question: whether the implant
returns after a reboot. It never answers the rootkit's question, whether the
OS can see the implant at all, and the design stops at that line. Anchors
are userland (ring 3) mechanisms only: no kernel or boot-start drivers, no
bootloader or firmware hooks, and no self-hiding. Four reasons hold the
line. Removability: `drift` can reliably remove a scheduled task; it cannot
repair a boot chain. Reliability: userland persistence survives OS updates
and Secure Boot, where kernel persistence breaks on exactly those. The
malware line: rootkit behavior is what makes antivirus classify a binary
with no nuance. Accountability: a defender can find an anchor, and that is
the accepted risk of authorized testing. Process injection stays on the same
side of the line: hollowing lives inside another userland process the OS
still manages, and it hides nothing from the OS. Injection is evasion;
subversion is a rootkit. Persistence always costs a disk artifact: RAM is
wiped by reboot, so an anchor's job is pointing the OS back at a file, and
the tethered prompt states exactly that price.

**The anchor adapts; the boot does not.** Linux anchors detect the init
system rather than assume one, checked in order: `/run/systemd/system`
(systemd), `/sbin/openrc` (OpenRC), `/etc/init.d` (SysV), `/etc/sv` (runit
or s6), `/etc/inittab` (busybox). So `anchor -m` on Linux accepts `systemd`,
`openrc`, `sysv`, `runit`, and `inittab` as mechanisms alongside the
Windows and macOS set (task, wmi, service, startup, launchd, registry).
Whatever the init system, the sequence on wake is identical (see the life
and death of a Shard, §5).

### 3.8 Pivoting and Movement

| Word | Function |
|---|---|
| `burrow` | SOCKS5 proxy through a Shard into the target network (`-l` local port, `-x` stop) |
| `passage` | Port forward through a Shard (`-l` local, `-r` remote, `-p` protocol) |
| `sow` | Lateral movement: deploy a Shard to another host via available vectors (SMB, SSH, WinRM, creds) (`-c` creds, `-v` vector): plant seeds |
| `portal` | Enable RDP or remote desktop on target (`-u` user, `-x` disable): open a portal |

### 3.9 Spoils: credentials and data (late phase)

| Word | Function |
|---|---|
| `spoils` | Full harvest: browsers, mail, wifi, ssh keys, DPAPI, tokens (`-b`, `-m`, `-w`, `-k`, `-x` all) |
| `crumbs` | Browser cookies from Chromium or Gecko profiles: the crumbs left behind |
| `vault` | OS credential store: Windows Credential Manager and DPAPI, macOS Keychain, libsecret (`-a` all, `-d` domain) |

> **Note on Spoils.** Credential theft is the single most malware-looking
> feature in this design. It is legal dual-use software, but it is the feature
> set that makes reviewers flinch. It is deferred to a late phase and should
> ship only as a deliberate decision.

### 3.10 Verse: composing operations

| Word | Function |
|---|---|
| `compose` | Write a Canto from Lines: Words plus conditions, target sets, and checkpoints |
| `rehearse` | Auditor dry run: OPSEC pre-score for every Line, footprint estimate, no target contact |
| `recite` | Execute a Canto with checkpointing; it halts at a failed Line with the rollback intact |
| `interlude` | Pause a reciting Canto |
| `cease` | Abort a Canto: clean stop at the next checkpoint |
| `oracle` | The AI copilot: `oracle ask` (natural language to Word), `oracle draft` (brief to Canto), `oracle triage` (Scrying results to findings graph) |

### 3.11 Voice: the interactive shell subsystem

`voice` is where operators live, so it gets subsystem treatment. Words:
`voice`, `voices`, `attend`, `murmur` (§3.4).

- **True PTY, not a dumb pipe**: raw mode, TERM, window-size forwarding
  (SIGWINCH), Ctrl-C and Ctrl-Z, UTF-8 passthrough. On Windows, **ConPTY**
  provides a real pseudo-terminal, not a cmd shim.
- **Streaming channel multiplexed over Whisper**: an interactive session is
  its own channel (not per-task jobs), so it never blocks tasking traffic.
- **Named, resumable sessions**: a voice survives beacon sleep, disconnects,
  and transport switches. `grasp` the Shard, `attend` the session, and state
  and history are intact. Most C2s lose interactive state when the session
  dies.
- **Multiple voices per Shard**, switchable like tmux windows.
- **`murmur` as async mode**: the same ongoing conversation, buffered, with
  output delivered at the next check-in. `utter` (one-shot), `voice` (live),
  and `murmur` (buffered) form the full spectrum.
- **In-session escape hatches**: `!` local passthrough, quick `lift` and
  `bestow` without leaving the session.
- **Full recording to the Ledger**: keystrokes and output with timestamps,
  replayable, evidence-grade for the report.
- **Auto-auth**: sudo and password prompts answered from the creds store where
  available.
- **Multi-operator**: `attend` from another operator's seat via coven, watch
  or take over, role-gated.
- **OPSEC note**: a live interactive session is one of the highest-noise
  things a Shard can do. Auditor scores `voice` high and suggests `murmur` or
  `utter` when the channel is hot.

### 3.12 Denial: availability and resilience testing (gated)

Availability testing is a standard engagement type: failover drills,
DoS-in-scope penetration tests, kiosk-lockdown assessments,
network-partition resilience, and phishing UX checks. These Words map to
those test scenarios and are the highest-impact, highest-noise Words in the
framework. The Auditor refuses them unless the engagement scope explicitly
includes availability testing (`scope: impact`), and every one demands
double confirmation. They land in v4, not v1.

| Word | Function |
|---|---|
| `cataclysm` | Force a system crash as a failover or HA resilience test. Windows BSOD via `NtRaiseHardError` (`-m` custom message); Linux kernel panic via Magic SysRq (`echo c > /proc/sysrq-trigger`, needs root; `-m` via `/dev/kmsg` at KERN_EMERG); FreeBSD via `sysctl debug.kdb.panic=1`; macOS has no userland panic trigger, so it falls back to killing WindowServer |
| `petrify` | Freeze all input (keyboard and mouse) for N seconds (`-t`): kiosk or frozen-session simulation |
| `blackout` | Network partition or outage test (`-t` duration; `-r` rate for degradation or throttle test) |
| `proclaim` | UI notification test: dialog box or toast (info, warning, error); phishing and UX assessment tooling |

### Console prompt

- Default: `{user}@sunder:~$`
- Tethered (at least one anchor set): `{user}@tethered:~$`. The change is a
  warning, not decoration: you have accepted an on-disk footprint.
- Untethered: `{user}@sunder:~$`

---

## 4. The Verse layer (the differentiator)

Every other C2 is a pile of commands. Sunder's core invention: **an engagement
is a first-class, composed, executable document.**

- **`compose`**: operations are written as Cantos: Lines of Words with
  conditions, target sets, and checkpoints.
- **`rehearse`**: a Canto can be dry-run before it ever touches a target.
  The Auditor scores every Line for OPSEC risk and footprint and shows the
  bill before recitation. A pre-audited, client-approvable operation plan is
  also the strongest answer to the authorization question: the composition
  *is* the authorization document.
- **`recite`**: checkpointed execution. A failed Line halts at the verse
  boundary with the Ledger intact. Rollback, not incident. `interlude` pauses
  mid-recitation; `cease` aborts cleanly at the next checkpoint.
- **Reproducibility**: quarterly engagements are the same Saga, re-recited
  against updated scope. Engagements are replayable, reviewable, and
  diff-able.
- **Oracle integration**: the copilot drafts Cantos from a plain-English
  brief; the operator approves Line by Line. Poetry never blocks usability;
  it is what the machine speaks.

Words to Lines to Cantos to Saga. The poetry is the automation architecture.

---

## 5. Systems

### Whisper (comms layer)

- Noise protocol with key rotation and forward secrecy: one crypto skeleton
  that works identically over HTTPS, DNS, or raw sockets.
- AES-256-GCM sessions.
- **Egress auto-probe**: the Wraith tests which egress paths work (443
  blocked? try 53, try a Raven) and self-switches. Operators pick a policy;
  the implant adapts.

### The life and death of a Shard

A Shard is either breathing or it is not, and the Overseer must never guess
which. The Wraith beacons and is never polled, so the controller can never
ask a dead Shard anything. Presence is therefore registry truth with age,
tracked through a state ladder:

| State | Condition | The console shows |
|---|---|---|
| breathing | last breath within grace (cadence times a jitter-aware multiplier) | "It breathes, shallow and patient." |
| silent | one or two beats missed, still inside grace | "It holds its breath." |
| dark | grace expired | "Silence. The wire is cold." |
| ash | immolated, or its doom confirmed | "It is ash." |

A `breath` on a dark Shard does not hang and does not error. It reports the
registry truth with the age of the silence: how long since the last breath
and how many beats were missed. A reaper on the Overseer walks the ladder
for every Shard, breathing to silent to dark, and narrates the crossing at
the console so an operator notices a lost Shard without polling. `shards`
and the Whisper registry endpoint carry the state column, so the console,
the verse, and the API always agree.

Offline is not death, and the difference is the anchor:

- **Sleep.** The process suspends with the machine and resumes on wake,
  breathing again. The Overseer sees a silent gap, then beacons resume.
- **Shutdown, unanchored.** The Wraith dies with the machine and is not
  there when it returns. The operator casts again. That is the honest life
  of an unanchored implant.
- **Shutdown, anchored.** The process dies, but the anchor survives on
  disk. Reboot relaunches it and the Shard returns from the dark. An
  anchored Wraith cannot die, only wait.
- **Doom on boot.** If the kill date passed while the machine was off, the
  anchored Wraith burns its own anchors and immolates before its first
  beacon. A dead order cannot resurrect.

Return from the dark depends on stable identity. `cast` embeds a shard id,
alongside the controller URL and cadence, into the binary, so every respawn
presents the same face. When a known id re-registers after going dark, the
Overseer reconciles: it logs the return and refreshes the session instead of
registering a stranger.

**The boot sequence.** An anchored Wraith never breathes during boot
initialization, on any OS. Anchors are userland, and nothing userland runs
at the kernel phase: on Windows, services fire only after the Service
Control Manager starts, and logon tasks wait for a session; on Linux, units
fire once the init system resolves their dependencies. Breathing also
requires the network, and no init system can be trusted to order it,
especially SysV, runit, and busybox inits, which have no dependency system
at all. The sequence is therefore universal: spawn when the anchor fires,
probe for network readiness, retry with the backoff `patience` sets, check
`doom`, and only then breathe. The first breath lands seconds to a minute
after boot, never at init, and deliberately not instant: the loudest moment
of a boot is the worst time to beacon. The anchor adapts to the init
system; the boot behavior never does (see §3.7).

**The cost of tenacity.** Anchors trade footprint for survival, and the
ranking is the same on every OS. On Windows, an auto-start service as
SYSTEM or an equivalent boot-time scheduled task is the tenacious choice:
both run with no user logged in. On Linux it is a systemd unit as root with
restart on failure; on macOS, a LaunchDaemon. WMI event subscriptions
survive reboot and stay nearly invisible to admins, but endpoint defense
watches them. Per-logon mechanisms, run keys, logon tasks, and
LaunchAgents, are weaker and relaunch the Wraith per session, so the Wraith
checks for an existing instance before a second one breathes. Anchoring
several mechanisms buys redundancy at multiplied footprint, which is
exactly why the Auditor's ledger counts every anchor: what cannot be
counted cannot be cleaned.

### Shards and Fragments (the WASM module runtime)

- The Wraith embeds a small WASM runtime (wasmtime or wasmi) and executes
  operator-written modules, called **Fragments**, at runtime.
- Any language that compiles to WASM works (Rust is first-class); one module
  runs on Windows, Linux, and macOS.
- Memory-safe by construction: no BOF-style crashes taking down the implant.
- Heavy logic never ships in the binary; the Wraith stays a generic comms
  host, with little for AV to statically fingerprint.
- The module registry (`quarry`) becomes Sunder's community ecosystem.

### Chorus (serverless mesh)

- Shards relay for each other in a peer mesh; any node carries Words to any
  other. If the Overseer goes dark, the mesh keeps operating and queues
  results. A C2 that works with no reachable server at all.

### Ravens (dead-drop resolvers)

- Commands are fetched from public services (gists, pastebin, DNS TXT, mail).
  You *send a raven*; it returns with commands. No persistent connection to
  find.

### Oracle (the AI copilot, offline-capable)

- **Natural language to Word or Canto**: knows your fleet, findings, and
  goals.
- **Auto-triage**: recon results land in a structured knowledge graph; Oracle
  connects the dots (users, hosts, creds, services, suggested paths).
- **Offline-capable**: ships with a local-model mode (for example llama.cpp)
  so the Overseer works air-gapped. Engagement data must never leave the box
  by default. A C2 that asks a cloud API about its own targets is an OPSEC
  disaster.

### Auditor (the OPSEC conscience)

- Every Word gets a risk score (signature exposure, persistence visibility,
  noise) and a footprint cost.
- The Wraith reports what it touched (files, registry, processes, network);
  Auditor maintains a per-engagement **footprint ledger**.
- `immolate` becomes burn: Auditor verifies trace cleanup against its ledger.
- A framework that polices its own operators is the strongest
  professionalism signal a C2 can send.

### Ledger (engagement management and reporting)

- Every Saga has scope, targets, findings, evidence (screenshots and files),
  and a timeline as first-class data.
- One-command red-team report generation (markdown or PDF) from the Ledger.

---

## 6. Console philosophy

The Overseer is a **domain console** (Sliver-style), not a terminal emulator.
Explicitly:

- **No BusyBox reimplementation.** We do not reimplement `ls`, `grep`, `find`,
  `top`, `ssh`, `scp`, `tar`, or `zip`. Every one of those exists, better, on
  the operator's own machine. Rebuilding them is months of work for zero
  advantage.
- **Local passthrough.** A small set of commands delegate to the real host
  shell (`!df`, `!tar`, and so on) so Unix operations work inside the console
  without reimplementation.
- **Remote Unix-style shims.** The feature worth keeping: the Wraith
  normalizes its output to Unix style regardless of the underlying OS. The
  operator knows `ls` and `ps`, not `dir` and `tasklist`, even when the target
  is Windows. This is a thin normalization layer on the **Wraith** side.
- **Curated conveniences.** A few bridges, such as piping remote output into a
  local filter, give the second-shell feel without the cost.

The lesson of the BusyBox idea is simple: the consistent Unix feel belongs in
the Wraith's output shims, not in an Overseer-side userspace clone.

---

## 7. Language choices

### Go Overseer

- Runs on the operator's machine: size and runtime footprint are irrelevant.
- Development velocity: the TUI and concurrency story (prompts, gRPC, cobra)
  is excellent.
- The part you write fast and iterate on forever.

### Rust Wraith

- **No runtime, no GC.** Go's GC is a *tell*: periodic stop-the-world,
  heap growth, goroutine scheduling are detectable behavior in an idle
  process. Rust is deterministic and silent.
- **Binary size.** Stripped `opt-level="z"` plus `panic=abort` lands in the
  hundreds of KB versus Go's multi-MB floor. Size is a heuristic.
- **Injection compatibility.** Injected into another process's address space,
  a GC runtime's assumptions break and the runtime panics. Rust code with no
  runtime runs clean inside a foreign process. Hollowing and reflective
  loading are exactly where Go implants die and Rust ones do not.
- **No Go fingerprint.** `.gopclntab`, buildinfo, and string table layout
  make Go binaries instantly recognizable. Rust binaries lack the tell.
- **Raw syscall work.** Controlled `unsafe` for NT API and direct syscalls
  when evasion demands bypassing hooked APIs.
- **WASM synergy.** The Fragment runtime embeds tiny in Rust; Rust to WASM is
  seamless for authoring Fragments.
- **Memory safety** is the free bonus: reliability, not philosophy. A
  segfault mid-engagement is an OPSEC incident.

### The asymmetry is the design

Server-side: optimize for development velocity (Go).
Implant-side: optimize for footprint, runtime silence, and surviving
injection (Rust).

---

## 8. Competitive positioning

| | Sliver | Havoc | Mythic | Cobalt Strike | **Sunder** |
|---|---|---|---|---|---|
| Engagement as executable document | ✗ | ✗ | ✗ | partial | **✓ Canto/Saga** |
| Pre-execution OPSEC rehearsal | ✗ | ✗ | ✗ | ✗ | **✓ rehearse** |
| Cross-platform runtime modules | ✗ | ✗ | ✗ | Win-only BOFs | **✓ WASM Fragments** |
| Works with no server (mesh) | ✗ | ✗ | ✗ | ✗ | **✓ Chorus** |
| AI-assisted ops (offline) | ✗ | ✗ | ✗ | ✗ | **✓ Oracle** |
| Built-in report from engagement | ✗ | ✗ | partial | ✗ | **✓ Ledger** |

The wedge: **Sunder is the C2 whose operations are composed, rehearsed, and
recited.** A framework that speaks its own language. No one else occupies
that space.

---

## 9. Known tradeoffs

- **WASM host-API tax.** Fragments cannot touch the OS directly; every
  capability must be exposed by the Wraith's host API. The Wraith therefore
  ships a real API surface, and host-ABI versioning is a maintenance burden.
- **Native-only territory.** Raw syscalls, NT API, process injection, and
  unhooking cannot live inside the sandbox; they live in the host or in
  native modules. Nothing-to-fingerprint is true for logic, not the base
  runtime.
- **Performance.** Interpreted runtimes are slow; JIT adds size. Heavy work
  (decrypting large browser DBs, bulk exfil) may justify native paths.
- **A runtime is a footprint.** Embedding a WASM runtime has its own strings
  and layout; an analyst could fingerprint it.
- **Two toolchains** (implant plus Fragment SDK) mean more plumbing than a
  single-language design.
- **Poetic naming is a training cost** unless the gloss system, aliases, and
  Oracle carry the load. The design rules in §2 exist to make sure they do.

---

## 10. Legal, licensing, and scope

- **License: AGPLv3.** Strong copyleft, OSI-approved, and fully
  redistributable, so it qualifies for the BlackArch repository. No one can
  host a proprietary Sunder service without releasing modifications.
- **Authorized use.** The README states the tool is for authorized penetration
  testing and security research only, on systems you own or have permission to
  test (mirroring BlackArch's own disclaimer). Sunder is a real tool under a
  real license.
- **Gated: the Denial category.** Availability and resilience testing is a
  real engagement type (failover drills, DoS-in-scope pentests). BSOD-class
  Words exist as test scenarios under the Denial category (§3.12), refused by
  the Auditor unless `scope: impact` is declared, double-confirmed, and
  scheduled for v4. Anything that reads as disruption for its own sake is out
  of scope.
- **Cut: full Unix shell reimplementation** (see Console philosophy).
- **Deferred: Spoils (credential stealers)**, a late-phase, deliberate
  decision.
- **Deferred: fileless execution (process hollowing)**, a late phase and
  native-only; the v1 Shard is a plain binary, which is legitimate. Sliver
  ships real binaries too.

---

## 11. Roadmap

### v1: the spine (weeks, BlackArch-submittable)
- Go Overseer console and Rust Wraith workspace, AGPLv3.
- Whisper comms: Noise over HTTPS, AES-256-GCM sessions, job loop.
- Built-in Words: `utter`, `gaze`, `unfold`, `lift`, `bestow`, `pulse`,
  `anatomy`, `breath`, `shards`, `grasp`.
- Presence registry: the state ladder (breathing, silent, dark, ash), the
  reaper, and return reconciliation for known shard ids.
- Remote Unix-style output shims on the Wraith.
- Auditor skeleton (Word risk scores) plus Ledger basics.
- Local passthrough bridge in the console.

### v2: the ecosystem
- WASM Fragment runtime in the Wraith, the `fragments` SDK, and the module
  registry (`quarry`).
- More Scrying Words as Fragments; file Words (`seek`, `sift`, `seal`).
- `voice` live PTY (ConPTY on Windows) with resumable sessions; `murmur`
  async mode for slow channels.

### v3: the intelligence
- Chorus serverless mesh.
- Coven team mode: operators, roles, shared sessions.
- Oracle copilot (local-model first).
- Ledger reporting (`ledger report`) and the Verse layer (`compose`,
  `rehearse`, `recite`).
- Pivoting: `burrow`, `passage`, `sow`, `portal`.

### v4: the long tail
- Ravens dead-drop resolvers.
- Persistence (`anchor` and `drift`) and injection modules.
- `ascend`, `muzzle`, `ablution`, `antedate` evasion Words.
- Denial Words (`cataclysm`, `petrify`, `blackout`, `proclaim`), gated
  behind `scope: impact`.
- Deliberate decision on Spoils.

---

## 12. BlackArch packaging

- Submit a PKGBUILD to `BlackArch/blackarch` (fork, add
  `sunder/PKGBUILD`, add `sunder` to `lists/to-release`, open a PR).
- AGPLv3 satisfies the repository's redistribution requirement.
- Test with `blackarch-devtools` (`ba-dev`) before submitting.

---

*Status: concept/design v0.9 (planning phase). The command reference in §3 is
the canonical surface: **117 Words** across 12 subsections. Words migrate into
the roadmap as they are implemented.*
