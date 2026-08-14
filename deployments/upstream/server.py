#!/usr/bin/env python3
"""Mock upstream server for hacp-sidecar demonstration."""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer
import sys


class MockUpstream(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        response = {
            "source": "mock-upstream",
            "method": "GET",
            "path": self.path,
            "message": "Request forwarded through HACP sidecar"
        }
        self.wfile.write(json.dumps(response).encode())

    def do_POST(self):
        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length).decode() if content_length > 0 else ""

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        response = {
            "source": "mock-upstream",
            "method": "POST",
            "path": self.path,
            "body_length": len(body),
            "message": "Request forwarded through HACP sidecar"
        }
        self.wfile.write(json.dumps(response).encode())

    def log_message(self, format, *args):
        print(f"[upstream] {args[0]}")


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
    server = HTTPServer(("0.0.0.0", port), MockUpstream)
    print(f"Mock upstream listening on :{port}")
    server.serve_forever()