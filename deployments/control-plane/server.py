#!/usr/bin/env python3
"""Mock control plane for hacp-sidecar demonstration."""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer
import sys
import time


class ControlPlane(BaseHTTPRequestHandler):
    revoked = {
        "tokens": set(),
        "envelopes": set(),
        "keys": set()
    }
    sequence = 0

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"ok\n")
            return

        if self.path == "/snapshot":
            ControlPlane.sequence += 1
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            snapshot = {
                "sequence": ControlPlane.sequence,
                "timestamp": int(time.time()),
                "revoked_tokens": list(ControlPlane.revoked["tokens"]),
                "revoked_envelopes": list(ControlPlane.revoked["envelopes"]),
                "revoked_keys": list(ControlPlane.revoked["keys"])
            }
            self.wfile.write(json.dumps(snapshot).encode())
            return

        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path.startswith("/revoke"):
            parts = self.path.split("/")
            if len(parts) >= 3:
                target_kind = parts[2]
                content_length = int(self.headers.get("Content-Length", 0))
                body = self.rfile.read(content_length).decode() if content_length > 0 else ""

                try:
                    data = json.loads(body)
                    target_id = data.get("target_id", "")
                except Exception:
                    target_id = body.strip()

                if target_kind in ControlPlane.revoked:
                    ControlPlane.revoked[target_kind].add(target_id)
                    ControlPlane.sequence += 1

                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                response = {"status": "revoked", "target_kind": target_kind, "target_id": target_id}
                self.wfile.write(json.dumps(response).encode())
                return

        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        print(f"[control-plane] {args[0]}")


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 5000
    server = HTTPServer(("0.0.0.0", port), ControlPlane)
    print(f"Mock control plane listening on :{port}")
    server.serve_forever()