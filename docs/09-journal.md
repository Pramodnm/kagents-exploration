# 09 · Journal: how the exploration went

A chronological record of the work, including the detours and mistakes. Read it to
learn where time went, and to avoid the same detours.

## Phase 1 · Access and baseline

- Reached a fresh Ubuntu VM over SSH. The first key was rejected for a permissions
  reason on Windows; restricting the file's ACL to one user fixed the client side.
- A second key, and a guessed admin username, were rejected. The VM's admin user
  was the Ubuntu default user. Lesson: confirm the username from the cloud console before debugging keys.
- Baseline at install: Ubuntu 24.04.5 LTS, 4 vCPU, 15 GiB RAM, `/dev/kvm` present. Later the same VM reported 8 vCPU and 31 GiB; the sizing change is not explained in this journal.
  The request had mentioned Ubuntu 26; this VM was 24.04, and the docs do not name 26.

## Phase 2 · Understanding kagent and Agent Substrate

- Read the kagent 1.0 install guide, the core concepts, the architecture pages, the
  Harness and BYO pages, and the Agent Substrate architecture and sandboxing pages.
- Key understanding: kagent 1.0 replaced the 0.x Deployment-based Agent with a
  Harness, AgentTemplate, Agent model, and runs each Session on a Substrate Actor.
- Found that "agentsubstare" in the request refers to Agent Substrate.

## Phase 3 · Install

- Installed Docker, kubectl v1.37.0, kind v0.32.0, Helm 3, and the kagent CLI 1.0.0.
- Created a kind cluster on `kindest/node:v1.37.0` and confirmed the certificate APIs.
- A dry run revealed the installer needs `OPENAI_API_KEY`; the install took about four
  minutes and rolled out cleanly.
- Tailscale was installed on the VM for reaching a model gateway on a workstation.

## Phase 4 · First agent and the model gateway

- Ran the default smoke agent. It answered "2+2" and recalled the previous question,
  which showed the Session transcript persists across turns.
- Verified the OpenAI-compatible gateway over HTTPS through Tailscale Serve. Plain HTTP
  worked; HTTPS on the same port failed until Serve was added.
- Chose the Responses API path. Chat Completions returned 400 for the same model.
- Confirmed tool calling (`finish_reason: tool_calls`), which kagent needs on every turn.

## Phase 5 · BYO agent: the registry rabbit hole (about 45 minutes)

- Built kagent's reference BYO example and pushed it to a local registry on HTTP. The
  golden snapshot failed with `server gave HTTP response to HTTPS client`.
- Tried exposing the registry over Tailscale HTTPS. The kind node could not reach the
  tailnet address. That exposure was reverted, and it was a mistake to leave it up for
  several minutes.
- The fix was in the upstream BYO guide: atelet's `--localhost-registry-replacement` flag,
  which I had missed. Once applied, the reference agent went Ready and answered over gRPC.

Lesson: read the whole tutorial for the path you are on before debugging the platform.

## Phase 6 · Microsoft Agent Framework (Go) and three personas

- Researched the Agent Framework. The Go SDK has an AG-UI provider and an OpenAI
  Responses adapter. The Python SDK was out of scope.
- Built a Go AG-UI server with three personas. Tested each endpoint against the gateway:
  all three streamed replies.
- Wrapped the server in a kagent A2A shim. The first image used a shell entrypoint, which
  failed on the distroless base; a Go launcher replaced it.
- The launcher ran the AG-UI server as a child process and `exec`'d the shim. Every
  invoke failed with `connection refused`. The fix was to run the AG-UI servers in the
  same process, bind the port first, then start A2A. The three agents then completed.
- Built a host client that checks each reply is non-empty. The first client read the
  wrong JSON field and reported empty replies; the reply was in `status.message`.
- Persona test: all three gave the same kind of answer. A direct gateway test showed the
  gateway honors instructions, and a request dump showed the Go provider sends them. The
  cause inside the cluster was not found. This is the open question in chapter 06.

## Phase 7 · Sandbox-managing agent

- Read the sandbox docs. Enabled sandboxes in the release, created a `python` template and
  a `jupyter` template (Jupyter baked in, because sandboxes cannot install at run time).
- Ran Python and a Jupyter notebook in a sandbox from the CLI. Created an agent that
  performs the full lifecycle on a request. It worked: output read back, sandbox deleted.
- Restriction checks: DNS-level egress blocked; memory 1 GiB enforced; CPU not confirmed.
- Found that a raw IP reached the internet from a sandbox with no policy. Reproduced it on
  a second sandbox, and the gateway logged both connections. This contradicts the
  documented default-deny. Recorded as finding F1 and not fixed.

## Phase 8 · Documentation

- Wrote this repository. The first attempt to write chapter 01 used a shell heredoc that
  broke on the backticks and the chapter's own shell blocks, so bash executed chapter text
  as commands. That run included a `kind delete cluster` line from the local shell. The
  VM cluster was checked afterward and was intact, with all agents still Ready. Chapter
  files were then written with the file tool instead, which avoids shell quoting.
- Found that the first draft of the shim `go.mod` referenced a path that exists only on
  the VM. It now points at a documented checkout, and the public source was rebuilt
  against kagent v1.0.0 to confirm the module graph resolves.

## Mistakes to avoid

| Mistake | What it cost | Do this instead |
|---|---|---|
| Secrets pasted into commands and chat | Keys in shell history and chat | Read keys from env vars or Secrets; rotate after demos |
| Plain-HTTP local registry | About 45 minutes | Read the BYO guide for the registry path first |
| Left a registry exposed on the tailnet | Brief unauthenticated exposure | Expose only what you need; remove it at once |
| Background child process in BYO image | Hours of "connection refused" | One process per image |
| Trusted the sandbox egress claim | Had to retract a "restricted" conclusion | Test the property you rely on, from inside the sandbox |
| Shell heredocs for long Markdown | A broken command run, cluster-side impact checked | Write long files with a file tool |

## Where things stood at the end

- kagent 1.0.0 and Agent Substrate 1.0.0 running on one kind node.
- Three BYO agents, one sandbox-managing agent (removed after the run), and the default
  smoke agent.
- Templates `python` and `jupyter` kept for reruns.
- Open: F1 (raw-IP egress), F4 (CPU), F6 (persona in-cluster).
