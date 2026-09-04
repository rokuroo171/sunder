# SUNDER

A C2 framework whose operations are composed, rehearsed, and recited.

Sunder is a cross-platform command-and-control framework that speaks its own
language. Every command is a Word with a poetic meaning behind it, and an
engagement is a Saga, an executable and auditable composition.

> The poetry is not decoration. Every command word is chosen so that its
> metaphor is also the technically exact name for the action. The vocabulary
> is literary and archaic English, and it is consistent to the last word.

The full design lives in [docs/DESIGN.md](docs/DESIGN.md).

## Components

| Name | Role | Language |
|---|---|---|
| **Overseer** | the controller, the hand | Go |
| **Wraith** | the implant, a Shard of the blade | Rust |
| **Fragments** | loadable WASM modules pushed to a Shard at runtime | any WASM target |

## Current state

The **Whisper v1 handshake over HTTPS** works end to end:

1. The Wraith connects to the Overseer over TLS
2. Both sides exchange ephemeral X25519 keys
3. A session key is derived with HKDF-SHA256
4. All further payloads are sealed with AES-256-GCM
5. The Wraith registers as a Shard and proves the loop with a `breath` word

On top of that wire, v0.2.0 adds the **Overseer console**: run the Overseer
on a terminal and it drops you into `user@sunder:~$` with `shards`, `grasp`,
`breath`, and `tone`, while Shard registrations and heartbeats narrate
behind the prompt. v0.3.0 turns the wire into a loop: every beat carries
the Shard's answer back and the next queued Word forward, so a grasped
Shard can be spoken to with `utter`, `gaze`, `unfold`, `pulse`, and
`anatomy`. The full design calls for the Noise protocol, dead drops,
mesh relays, and the WASM fragment runtime; those land in later milestones.

## Quick start

```sh
# build and run the Overseer (requires Go 1.24+)
make overseer
make run        # on a tty this drops you into the console

# in another shell, beacon with the Wraith (requires a Rust toolchain)
cd wraith && cargo run -- https://localhost:8443 --loop --interval 5
```

Inside the console, watch a Shard land and speak to it:

```
rokuroo@sunder:~$ shards
rokuroo@sunder:~$ grasp 71d9d655
You have its ear.
rokuroo@sunder:~$ breath
It breathes, shallow and patient.
rokuroo@sunder:~$ utter echo hello
hello
```

Run `overseer -headless` to serve without the console, or `overseer -console`
to force the console when stdin is not a tty. The Overseer serves a
self-signed certificate generated at startup. The Wraith accepts any
certificate in this scaffold build, dev only, for a local handshake.
Certificate pinning and proper trust belong to a later milestone.

## Authorized use

Sunder is intended for authorized penetration testing, red team engagements,
and security research only. Use it only on systems you own or have explicit
written permission to test. Unauthorized access to computer systems is a crime
in most jurisdictions. The author of this software is not responsible for
misuse.

## Command surface

The complete lexicon, 117 Words across twelve categories, is catalogued in
[docs/DESIGN.md](docs/DESIGN.md#3-the-word-reference). A few of its voices:

| Word | Meaning | Action |
|---|---|---|
| `cast` | deploy a Shard | ship an implant |
| `breath` | heartbeat check | is the Shard alive |
| `gaze` | list a directory | `ls` |
| `immolate` | burn the Shard to ash | self-destruct |

## License

Sunder is released under the GNU Affero General Public License v3.0. See
[LICENSE](LICENSE).
