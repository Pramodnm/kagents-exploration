# Verified runs

Recorded outputs from the runs that the chapters cite. Hostnames, IPs of the VM, keys,
and identities are removed or replaced with placeholders. Timestamps are from the
original runs.

## Host client, three agents (chapter 04)

```text
agui-assistant  state=TASK_STATE_COMPLETED   ok=true  reply="I'm a concise general assistant."
agui-coder      state=TASK_STATE_COMPLETED   ok=true  reply="I'm Claude, an AI assistant made by Anthropic."
agui-tutor      state=TASK_STATE_COMPLETED   ok=true  reply="I'm Claude, an AI assistant made by Anthropic."
PASS: 3 of 3 agents completed with a reply
client_exit=0
```

## Persona check on a greeting (chapter 04, F6)

```text
== agui-assistant
Hello! What do you need help with?
== agui-coder
Hello.
== agui-tutor
Hello! 👋
```

## Persona check on a coding prompt (chapter 04, F6)

```text
== agui-assistant
print(2 + 3)
== agui-coder
print(2 + 3)
== agui-tutor
print(2 + 3)
```

## Direct gateway persona test (chapter 06, F6)

```text
== with instructions (pirate)
Ahoy, matey! ...
== without instructions
Hello! ...
```

## Request dump, Go provider (chapter 06, F6)

The outgoing Responses body carried the persona as `instructions`:

```text
{"instructions":"You are a pirate. Always talk like a pirate.","store":false,
 "include":["reasoning.encrypted_content"],"input":[{"content":[{"text":"Say hello.",
 "type":"input_text"}],"role":"user","type":"message"}],"model":"<model-id>"}
```

## Sandbox egress, two fresh sandboxes (chapter 06, F1 and F2)

```text
1.1.1.1 REACHABLE
8.8.8.8 REACHABLE
example.com BLOCKED gaierror
```

Gateway log for the same Actor:

```text
proxy::gateway  CONNECT tunnel terminated  target=1.1.1.1:443  atespace=kagent
proxy::gateway  CONNECT tunnel terminated  target=8.8.8.8:443  atespace=kagent
```

## Sandbox memory (chapter 06, F3)

```text
cpus visible: 2
meminfo: MemTotal:        1048576 kB
```

`cpu.max` and `memory.max` cgroup files were not present in the sandbox.

## Sandbox-managing agent, notebook request (chapter 05)

Decoded output of `read_sandbox_outputs` for the notebook run:

```text
1 {"count": 3, "sum": 10}
2 blocked: True gaierror
```

Final lifecycle: `create_sandbox` (template `jupyter`, `ttl_seconds: 900`), notebook
written and executed by `nbconvert` (exit 0), `read_sandbox_outputs`, `delete_sandbox`
returning `RUNTIME_STATE_DELETED`.

## Cluster state at the time of measurement (chapter 07)

```text
pods running (all namespaces): 25
memory: 2.0Gi used of 31Gi
agents in kagent namespace: 4
```

## Sandbox guest and template readiness (chapter 05)

```text
python   docker.io/library/python@sha256:...    Ready=True  Prepared
jupyter  <registry>/sandbox-jupyter@sha256:...  Ready=True  Prepared
```
