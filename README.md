# kagent + Agent Substrate: Field Notes

A hands-on exploration of [kagent](https://kagent.dev) 1.0 and
[Agent Substrate](https://ate.dev), run end to end on a single Azure Ubuntu VM.
It covers installation, the architecture, bring-your-own (BYO) agents built with the
Microsoft Agent Framework (Go SDK), a sandbox-managing agent, and what the platform
does and does not enforce today.

Everything here was run and checked on a live cluster. Where a claim was **not**
verified, or where behavior contradicted the documentation, the document says so.

## Who this is for

- Developers who want to run agents on Kubernetes and need to know what kagent does for them.
- Architects evaluating kagent and Agent Substrate for sandboxed or stateful agents.
- Anyone who wants to reproduce these results, step by step.

## What you will learn

| Topic | Document |
|---|---|
| Install kagent 1.0 and Agent Substrate on a single VM | [docs/01-setup.md](docs/01-setup.md) |
| How the pieces fit: Harness, AgentTemplate, Agent, Session, Actor, Worker | [docs/02-architecture.md](docs/02-architecture.md) |
| Run a model through an OpenAI-compatible gateway (omniroute) | [docs/03-models.md](docs/03-models.md) |
| Build a BYO agent: three Agent Framework agents behind kagent's A2A contract | [docs/04-byo-agents.md](docs/04-byo-agents.md) |
| A sandbox-managing agent that runs a Jupyter notebook on request | [docs/05-sandbox-agent.md](docs/05-sandbox-agent.md) |
| What is enforced, what is not, and the evidence | [docs/06-limitations.md](docs/06-limitations.md) |
| Decision guide: kagent and Agent Substrate | [docs/07-decision-guide.md](docs/07-decision-guide.md) |
| Common agent use cases and how this stack fits them | [docs/08-use-cases.md](docs/08-use-cases.md) |
| Full chronology of the exploration | [docs/09-journal.md](docs/09-journal.md) |

## Repository layout

```
docs/                      Guide chapters, read in order (01 to 09)
examples/01-setup/         Smoke-test Agent manifest
examples/02-byo-agui-shim/ BYO image: shim, Dockerfile, harness.yaml, host client
examples/03-models/        ModelConfig for an OpenAI-compatible gateway
examples/03-sandbox-agent/ Sandbox agent manifest and Jupyter image
examples/06-verify/        Egress check script (finding F1)
scripts/verify.sh          Health check and sandbox round trip
diagrams/                  Mermaid sources
evidence/verified-runs.md  Recorded outputs, secrets removed
```

## Quick verification

After setup (chapter 01), these commands confirm the platform is healthy:

```bash
kubectl get pods -n kagent
kubectl get pods -n ate-system
kubectl get agent -n kagent
```

## Status of this work

| Area | State |
|---|---|
| kagent 1.0.0 install (CLI path) on kind | Verified |
| Model routing through an OpenAI-compatible gateway (Responses API) | Verified |
| BYO agent with kagent A2A contract (Go) | Verified |
| Three Agent Framework agents behind one BYO image | Verified over A2A |
| Sandbox create / run / delete by an agent | Verified |
| Raw-IP egress restriction | **Not enforced** (see chapter 06) |
| CPU cap inside a sandbox | Not confirmed |
| Persona behavior of the three agents | Not confirmed (see chapter 04) |

## Safety notes before you run anything

- Never commit model API keys. The examples read keys from environment variables or
  Kubernetes Secrets. Do not paste them into manifests.
- The examples use placeholder hostnames such as `llm-gateway.example.internal`.
  Replace them with your own values.
- Port 8083 (the kagent controller) runs in `insecure` auth mode by default. Keep it
  inside the cluster or behind port-forwarding.

## License

Documentation: CC BY 4.0. Code examples: MIT. Upstream projects (kagent, Agent
Substrate, Microsoft Agent Framework) keep their own licenses.
