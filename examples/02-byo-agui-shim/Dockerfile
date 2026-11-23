# Single-process BYO image: the shim runs the AG-UI servers in-process.
# Distroless has NO shell. Do not use a shell entrypoint here
# (see docs/04-byo-agents.md, Failure 2).
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY shim /app/shim
ENTRYPOINT ["/app/shim"]
