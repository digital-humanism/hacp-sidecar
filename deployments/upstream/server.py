#!/usr/bin/env python3
"""Mock HTTP/1.1 upstream server for hacp-sidecar tests."""

import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class MockUpstream(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def send_json(
        self,
        status_code: int,
        payload: dict,
    ) -> None:

        body = json.dumps(
            payload,
            separators=(",", ":"),
        ).encode("utf-8")

        self.send_response(
            status_code,
        )

        self.send_header(
            "Content-Type",
            "application/json; charset=utf-8",
        )

        # Required for reliable HTTP/1.1 persistent connections.
        self.send_header(
            "Content-Length",
            str(len(body)),
        )

        self.end_headers()

        self.wfile.write(
            body,
        )

    def do_GET(self) -> None:

        self.send_json(
            200,
            {
                "source": "mock-upstream",
                "method": "GET",
                "path": self.path,
                "message": "Request forwarded through HACP sidecar",
            },
        )

    def do_POST(self) -> None:

        try:
            content_length = int(
                self.headers.get(
                    "Content-Length",
                    "0",
                )
            )
        except ValueError:
            content_length = 0

        if content_length > 0:
            raw_body = self.rfile.read(
                content_length,
            )

            body = raw_body.decode(
                "utf-8",
                errors="replace",
            )
        else:
            body = ""

        self.send_json(
            200,
            {
                "source": "mock-upstream",
                "method": "POST",
                "path": self.path,
                "body_length": len(body),
                "message": "Request forwarded through HACP sidecar",
            },
        )

    def log_message(
        self,
        format,
        *args,
    ) -> None:

        print(
            f"[upstream] "
            f"{self.client_address[0]}:"
            f"{self.client_address[1]} "
            f"{args[0]}"
        )


if __name__ == "__main__":

    port = (
        int(sys.argv[1])
        if len(sys.argv) > 1
        else 8000
    )

    server = ThreadingHTTPServer(
        ("0.0.0.0", port),
        MockUpstream,
    )

    server.daemon_threads = True

    print(
        f"Mock upstream (threaded HTTP/1.1) listening on :{port}"
    )

    try:
        server.serve_forever()

    except KeyboardInterrupt:
        print(
            "\nShutting down mock upstream..."
        )

    finally:
        server.server_close()