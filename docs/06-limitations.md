# 06 · Limitations and findings

This chapter lists what the platform does and does not do, with the evidence for each
claim. Each finding is labeled **verified**, **contradicted by docs**, or **not
confirmed**. Where a claim could not be settled, it says so.

## Summary

| ID | Area | Finding | Status |
|---|---|---|---|
| F1 | Sandbox egress | A sandbox with no egress policy reached public IPs directly | **Contradicts docs** |
| F2 | Sandbox egress | Hostnames are blocked (DNS fails) | Verified |
| F3 | Sandbox memory | Memory cap is enforced (1 GiB visible) | Verified |
| F4 | Sandbox CPU | CPU limit not visible; sandbox reports 2 CPUs | **Not confirmed** |
| F5 | Sandbox isolation | Host state path not visible from sandbox | Verified (one path only) |
| F6 | BYO personas | Persona instructions do not change in-cluster replies | **Not resolved** |
| F7 | BYO startup | Two-process BYO image races and fails under Substrate | Verified; fixed by one process |
| F8 | Image registry | Substrate pulls over HTTPS; HTTP local registry fails | Verified; fixed by atelet flag |
| F9 | Harness env | Secrets cannot be passed through Harness `env` | Verified (documented) |
| F10 | Agent tools | Agent called a tool outside its binding and once passed null arguments | Observed once |
| F11 | Controller | No AG-UI route for an Agent; only A2A (JSON-RPC and gRPC) | Verified |
| F12 | Auth | Controller runs in `insecure` mode by default | Verified (docs) |

## F1 · Raw-IP egress is open (the important one)

The kagent networking documentation describes a default-deny policy:

> The policy is default-deny. An Actor can attempt any request, and the gateway denies the
> ones that no rule allows.

A standalone sandbox with an empty `egressPolicyRefs` (no EgressPolicy, no derived
destinations) should therefore be unable to open a connection.

### Evidence

Two separate fresh sandboxes from the `python` template ran this check:

```python
import socket
for host in ["1.1.1.1", "8.8.8.8", "example.com"]:
    try:
        socket.create_connection((host, 443), timeout=5)
        print(host, "REACHABLE")
    except OSError as e:
        print(host, "BLOCKED", type(e).__name__)
```

Result, both runs:

```text
1.1.1.1 REACHABLE
8.8.8.8 REACHABLE
example.com BLOCKED gaierror
```

The gateway log for the same Actor recorded both IP connections:

```text
proxy::gateway  CONNECT tunnel terminated  target=1.1.1.1:443   atespace=kagent
proxy::gateway  CONNECT tunnel terminated  target=8.8.8.8:443   atespace=kagent
```

The hostname was blocked only because DNS did not resolve. The IP connections were
accepted by the gateway.

### Why this is not yet a verdict

The documentation describes rules by hostname. A raw IP has no hostname to match, so the
docs do not say what should happen. The two possibilities are a gateway that fails open
for IP destinations, or a documented gap. Both would mean that **a sandbox is not isolated
from the internet by IP**.

### How to reproduce

```bash
kagent sandbox create python --request-id f1-$(date +%s) --ttl 300s -o json
kagent sandbox upload <sandbox-id> ./check.py /data/workspace/check.py
kagent sandbox exec <sandbox-id> -- python3 /data/workspace/check.py
kubectl logs -n ate-system deploy/atenet-egress --since=5m | grep CONNECT
```

Use `examples/06-verify/check-egress.py` for `check.py`.

### What to do

Treat sandbox egress as **not restricted for IP destinations** until the maintainers
confirm the intended behavior. Do not rely on an empty `egressPolicyRefs` to keep
untrusted code offline.

## F2 · Hostname egress is blocked

`example.com` fails with `gaierror` inside the sandbox, and the agent's notebook cell
reported `blocked: True gaierror`. This matches the documented behavior for names.

## F3 · Memory limit is enforced

Inside the sandbox, `/proc/meminfo` reports `MemTotal: 1048576 kB`, which is 1 GiB. This
matches `config.sandboxes.memory: 1Gi`.

## F4 · CPU limit is not confirmed

The Helm value is `cpu: "1"`, but the sandbox reports `os.cpu_count() == 2`, and the
cgroup file `/sys/fs/cgroup/cpu.max` does not exist in the gVisor sandbox.

A CPU limit might still apply through a mechanism the sandbox cannot see. The check
that would settle it, a fixed CPU-bound workload timed against a known baseline, was not
run. Do not cite the CPU limit as enforced.

## F5 · Host state is not visible from the sandbox

`/var/lib/ate` (Substrate's on-node state) does not exist inside the sandbox. This tests one
path. It is not a full escape test.

## F6 · Persona instructions do not change in-cluster replies (open)

The three BYO agents differ only by instructions. What is known:

| Check | Result |
|---|---|
| Go provider sends `instructions` on the wire | Yes (request dump, chapter 04) |
| Gateway honors `instructions` on a direct call | Yes (a pirate instruction changed the reply) |
| Same instruction, inside the Actor | Replies are unchanged across all three personas |

The cause is not isolated. The likeliest candidates, none tested:

1. The Actor's request differs from the direct call (for example, message content or a
   system message order).
2. The shim's `Name` field or run options drop the instructions in a path the request dump
   did not cover.

**Next step to close this:** log the outgoing Responses request from inside the Actor.

## F7 · Two processes in a BYO image fail under Substrate

Running an AG-UI server as a child process and then `exec`ing the shim produced
`connection refused` on the server's port, on every invoke. The fix was one process
that binds its listener before serving A2A.

Lesson: treat a BYO image as one process you own. Substrate checkpoints the whole Actor
as a unit, so background children are fragile.

## F8 · Substrate pulls images over HTTPS

A local registry on plain HTTP fails with `server gave HTTP response to HTTPS client`.
The documented fix for a kind setup is the atelet flag
`--localhost-registry-replacement=kind-registry:5000`, with `localhost` references in the
Harness. A registry exposed over Tailscale HTTPS did not work, because the kind node
could not reach the tailnet address (the exposure was reverted).

## F9 · Secrets cannot go through Harness env

Harness `env` accepts only literal values. The `credentialRef` option is deferred in the
v1alpha3 API. Keys must go through a ModelConfig or EgressPolicy, which the gateway
injects.

## F10 · Agents call tools outside their binding

In the sandbox-agent run, the model called a `bash` tool that the AgentTemplate did not
bind, and once called `create_sandbox` with null arguments. Neither left an orphan sandbox
that could be found. The lesson: a prompt is not a control. Bind only the tools you need,
and treat tool calls as untrusted input to your own audit.

## F11 · No AG-UI route on the controller

The controller serves JSON-RPC and gRPC A2A for each Agent. It has no `/agents/.../agui`
route. A client that speaks AG-UI can reach an agent only from inside the Actor.

## F12 · Default auth mode is insecure

A default install runs the controller in `insecure` mode: any caller that reaches port
8083 can invoke any Agent. Keep the port inside the cluster, and read the identity docs
before exposing it.

## Checks not run

- Measured CPU enforcement (F4).
- Persona behavior from inside the Actor (F6).
- Ubuntu 26.04 (the VM ran 24.04.5).
- Reboot behavior of the VM, kind cluster, Tailscale serve, and the omniroute process.
- The omniroute request log for the model that answered (the response body reported a
  routed model name, but the log was not read).
- Multi-user authorization on sessions, and any load test.

## What "limited" means here

Each limitation above affects the decision in chapter 07. None of them is a reason to
avoid the platform by itself. Each is a reason to test the specific property you rely on
before you depend on it.
