# 07 · Decision guide: kagent and Agent Substrate

This chapter helps you decide whether to adopt kagent 1.0 and Agent Substrate, and for
what. It is based only on what was built and observed in this exploration, plus the
upstream documentation where noted. Scores are a judgment; each one cites its evidence.

## Decision in one paragraph

**Adopt kagent for Kubernetes-native agents whose callers speak A2A**, where you want
declarative definitions, a managed Session store, and suspend-and-resume of idle agents.
**Do not adopt the sandbox product for untrusted code isolation yet.** The sandbox
lifecycle works, but the egress restriction it advertises does not hold for IP
destinations in the tested build (chapter 06, F1). Run it only where code is trusted or
where an outer network policy also blocks egress.

## What it is for, and what it is not

| kagent is | kagent is not |
|---|---|
| A control plane that compiles Harness, AgentTemplate, and Agent into runnable Actors | A chat UI product (the UI exists, but it is not the value) |
| An A2A gateway (JSON-RPC and gRPC) with Sessions and task history | A model gateway (you bring one, chapter 03) |
| A way to run your own agent image under a fixed contract (BYO) | An agent framework (BYO lets you bring yours) |
| A sandbox service for scratch environments | A proven security boundary for hostile code (see F1) |

Agent Substrate is the runtime under kagent. It starts Workers, runs Actors inside
gVisor sandboxes, checkpoints idle Actors to snapshots, and restores them on the next
message. You can also use it directly, but this guide did not evaluate that path.

## Measured numbers

These come from the verified single-node runs. Treat them as a baseline, not a benchmark.

| Metric | Observed | Context |
|---|---|---|
| kagent install (CLI, kind, one node) | about 230 s | Fresh VM at 4 vCPU and 15 GiB (the size at install time) |
| Agent first `Ready` after apply | about 1 min | Golden snapshot build; longer on failures |
| Sandbox create to `RUNTIME_STATE_READY` | about 0.8 s | `python` template, warm WorkerPool |
| One A2A `SendMessage`, trivial text | about 5 s | Three invokes in 16.7 s, same Agent family |
| Notebook request end to end (sandbox agent) | 56 s wall | About 24 tool calls, including polls |
| Prompt tokens per trivial turn | about 3,100 to 4,100 | Reported in task usage: kagent adds a large fixed prompt |
| Memory in use | 2.0 GiB of 31 GiB | Sampled on the 8 vCPU VM with four agents deployed |
| Sandbox memory | 1 GiB visible | Matches `config.sandboxes.memory` |
| Sandbox CPUs visible | 2 | Configured 1; not confirmed as a limit (F4) |
| Pods running (all namespaces, Running) | 25 | Single node, after sandbox and agent runs |

The token figure matters for cost. A trivial message costs roughly 3,000 to 4,000 prompt
tokens on the kagent harness, before your own content. Expect that overhead per turn,
and multiply it by the turns in a long session.

## Scorecard

Ratings: **Strong**, **Adequate**, **Weak**, or **Not tested**. Each has its evidence.

| Criterion | kagent | Agent Substrate | Evidence |
|---|---|---|---|
| Declarative agent definition | Strong | n/a | Harness, AgentTemplate, Agent reconciled to Ready (chapters 01, 05) |
| BYO runtime under a fixed contract | Adequate | n/a | Three agents served over A2A; contract rules are strict (chapter 04) |
| Protocol choice for callers | Strong | n/a | JSON-RPC and gRPC on one port, same task shape (chapter 02) |
| Suspend and resume of idle work | Adequate | Strong | Checkpoint and restore observed in atelet logs; behavior of long idle periods not tested |
| Session persistence | Adequate | n/a | Transcript recalled across turns (chapter 01 smoke test) |
| Sandbox lifecycle (create, run, delete) | Strong | Adequate | Sub-second create; full cycle by an agent (chapter 05) |
| Sandbox memory limit | n/a | Adequate | Enforced at 1 GiB (chapter 06, F3) |
| Sandbox CPU limit | n/a | **Not confirmed** | F4 |
| Sandbox egress restriction | n/a | **Weak** | Raw IPs reachable with no policy (F1) |
| Hostname egress restriction | n/a | Adequate | DNS blocks unlisted names (F2) |
| Credential handling | Adequate | Adequate | Gateway injection works; Harness env cannot carry secrets (F9) |
| Default security posture | **Weak** | n/a | `insecure` auth mode by default (F12) |
| Operational footprint | Weak | Weak | 25 running pods on one node; many CRDs; Helm plus Secrets plus pools |
| Upgrade path | Weak | Weak | Docs state 0.x to 1.0 is not in-place; patch upgrades not tested here |
| Documentation accuracy | Adequate | Adequate | Most commands worked as written; the egress claim did not (F1) |
| Debuggability | Adequate | Weak | Controller conditions are good; in-Actor logs are hard to reach (F6) |
| Build friction for BYO | Weak | n/a | Registry over HTTPS, distroless, one-process rule (chapter 04) |

## When to adopt

Adopt kagent when most of these hold:

1. Your agents run on Kubernetes already, and your team operates clusters.
2. Callers will use A2A (or can), and you want a stable address per agent, not per pod.
3. You need many idle agents and cannot keep each one resident. Suspend and resume is the
   point of Substrate.
4. You can accept the trust model for the sandbox you use (see below).
5. You will run an external PostgreSQL with pgvector and an object store, not the dev
   defaults. The docs require this for production; the kind path is for evaluation.

## When not to adopt yet

- **Hostile or unreviewed code in a sandbox,** unless an outer network policy also
  blocks egress. F1 says the gateway does not stop IP connections in this build.
- **A single agent on a single laptop.** The operational cost is higher than the value.
- **Strict per-user isolation of Sessions by default.** The default auth mode is
  `insecure`; you must configure real authentication first.
- **A team that cannot debug a Kubernetes control plane.** Several failures here came
  from registry, image, and startup details that the controller reports only indirectly.

## Risks to plan for

| Risk | Mitigation |
|---|---|
| Sandbox reaches internet by IP (F1) | Outer NetworkPolicy or firewall on Workers; do not rely on EgressPolicy alone |
| Model answers ignore the persona (F6) | Verify persona behavior in-cluster before shipping a persona |
| Agent calls tools outside its binding (F10) | Bind only needed tools; audit tool calls; do not treat the prompt as a control |
| Keys in chat or command history | Rotate after any demo; use Secrets; never inline keys |
| Image pull over HTTPS for local registries | Use a real registry with TLS, or the atelet rewrite flag for kind only |
| Preview feature gates | Harness uses `preview` gates in the Substrate chart; track them on upgrade |

## Questions to answer before you commit

1. Does an isolated sandbox need to block the internet by IP? If yes, F1 must be resolved
   or covered outside kagent.
2. What is your per-turn budget, given roughly 3,000 to 4,000 fixed prompt tokens?
3. Who owns the image contract for BYO agents, and how will you test the one-process rule?
4. What is your upgrade cadence? kagent 1.0 was new at the time of this work.
5. Which identity will callers use, and how will Sessions be shared?

## Recommendation

Pilot kagent with one declarative agent and one BYO agent behind A2A. Measure the three
numbers that matter to you (time to Ready, per-turn token cost, and suspend and resume
latency for your own workload). Keep the sandbox for trusted code until F1 has an answer
from the maintainers, and add a network control outside kagent in the meantime.
