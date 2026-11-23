"""Egress check to run inside a kagent sandbox.

Reports whether each destination is reachable from the sandbox. Used to reproduce
finding F1 in docs/06-limitations.md. Run it with:
  kagent sandbox exec <id> -- python3 /data/workspace/check-egress.py
"""
import socket

DESTINATIONS = ["1.1.1.1", "8.8.8.8", "example.com"]

for host in DESTINATIONS:
    try:
        socket.create_connection((host, 443), timeout=5)
        print(host, "REACHABLE")
    except OSError as err:
        print(host, "BLOCKED", type(err).__name__)
