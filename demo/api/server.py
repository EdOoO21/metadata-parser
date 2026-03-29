from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import os
from pathlib import Path


BASE_DIR = Path(__file__).resolve().parent
DEFAULT_SLOT = "1"
DIFF_MODES = {"baseline", "changed"}


def load_case(slot_id: str):
    normalized_slot = slot_id if slot_id in {"1", "2", "3", "4", "5", "6", "7", "8"} else DEFAULT_SLOT
    mode = os.environ.get("DEMO_API_MODE", "").strip()

    if mode in DIFF_MODES:
        case_dir = BASE_DIR / "diff" / normalized_slot / mode
    else:
        case_dir = BASE_DIR / f"test_{normalized_slot}"

    openapi_path = case_dir / "openapi.json"
    routes_path = case_dir / "routes.json"

    if not openapi_path.exists():
        raise FileNotFoundError(f"openapi fixture not found for slot {normalized_slot}")
    if not routes_path.exists():
        raise FileNotFoundError(f"routes fixture not found for slot {normalized_slot}")

    with open(openapi_path, "r", encoding="utf-8") as file:
        openapi = json.load(file)

    with open(routes_path, "r", encoding="utf-8") as file:
        routes = json.load(file)

    return normalized_slot, openapi, routes


SLOT_ID = os.environ.get("DEMO_API_SLOT", DEFAULT_SLOT)
CASE_ID, OPENAPI, ROUTES = load_case(SLOT_ID)


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/openapi.json":
            self.respond(200, OPENAPI)
            return

        payload = ROUTES.get(self.path)
        if payload is not None:
            self.respond(200, payload)
            return

        self.respond(404, {"error": "not found", "slot": SLOT_ID, "case": CASE_ID, "path": self.path})

    def log_message(self, format, *args):
        return

    def respond(self, status, payload):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 8080), Handler)
    server.serve_forever()
