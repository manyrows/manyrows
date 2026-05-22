import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer

from manyrows import ManyRowsServer, ManyRowsServerError


class _Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):  # silence test output
        pass

    def _handle(self):
        srv = self.server
        srv.last_method = self.command
        srv.last_path = self.path
        srv.last_api_key = self.headers.get("X-API-Key")
        length = int(self.headers.get("Content-Length") or 0)
        srv.last_body = self.rfile.read(length) if length else b""

        status, body = srv.responder(self)
        self.send_response(status)
        if body is None:
            self.end_headers()
            return
        payload = json.dumps(body).encode()
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(payload)

    do_GET = _handle
    do_POST = _handle
    do_PUT = _handle
    do_PATCH = _handle
    do_DELETE = _handle


class ServerClientTest(unittest.TestCase):
    def _client(self, responder):
        srv = HTTPServer(("127.0.0.1", 0), _Handler)
        srv.responder = responder
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        # Cleanups run LIFO: shutdown() stops serving, then server_close()
        # releases the listening socket (avoids ResourceWarning).
        self.addCleanup(srv.server_close)
        self.addCleanup(srv.shutdown)
        port = srv.server_address[1]
        client = ManyRowsServer(
            base_url=f"http://127.0.0.1:{port}/",  # trailing slash trimmed
            workspace="acme",
            app_id="app-1",
            api_key="mr_abc_secret",
        )
        return client, srv

    def test_check_permission(self):
        client, srv = self._client(
            lambda h: (200, {"allowed": True, "permission": "posts:read", "accountId": "u1"})
        )
        res = client.check_permission("u1", "posts:read")
        self.assertTrue(res.allowed)
        self.assertEqual(res.account_id, "u1")  # camelCase -> snake_case
        self.assertEqual(srv.last_method, "GET")
        self.assertIn("/x/acme/api/v1/apps/app-1/check-permission", srv.last_path)
        self.assertIn("accountId=u1", srv.last_path)
        self.assertEqual(srv.last_api_key, "mr_abc_secret")

    def test_create_user_posts_body(self):
        client, srv = self._client(
            lambda h: (201, {"user": {"id": "u2", "email": "a@b.com"}, "created": True, "roles": ["editor"]})
        )
        res = client.create_user(email="a@b.com", email_verified=True, roles=["editor"])
        self.assertTrue(res.created)
        self.assertEqual(res.user.id, "u2")
        self.assertEqual(srv.last_method, "POST")
        sent = json.loads(srv.last_body)
        self.assertEqual(sent, {"email": "a@b.com", "emailVerified": True, "roles": ["editor"]})

    def test_non_2xx_raises_typed_error(self):
        client, _ = self._client(lambda h: (404, {"error": "error.notFound", "message": "Not found"}))
        with self.assertRaises(ManyRowsServerError) as ctx:
            client.get_user("missing")
        err = ctx.exception
        self.assertEqual(err.status, 404)
        self.assertEqual(err.code, "error.notFound")
        self.assertEqual(err.message, "Not found")

    def test_delete_field_value_no_content(self):
        seen = {}

        def responder(h):
            seen["method"] = h.command
            seen["path"] = h.path
            return (204, None)

        client, _ = self._client(responder)
        self.assertIsNone(client.delete_user_field_value("f1", "u1"))
        self.assertEqual(seen["method"], "DELETE")
        self.assertTrue(seen["path"].endswith("/user-fields/f1/users/u1"))

    def test_list_users_omits_empty_params(self):
        client, srv = self._client(
            lambda h: (200, {"members": [], "total": 0, "page": 0, "pageSize": 50})
        )
        client.list_users(search="ali")
        self.assertTrue(srv.last_path.endswith("/users?search=ali"))

    def test_constructor_validation(self):
        with self.assertRaises(ValueError):
            ManyRowsServer(base_url="", workspace="a", app_id="b", api_key="c")
        with self.assertRaises(ValueError):
            ManyRowsServer(base_url="x", workspace="a", app_id="b", api_key="")


if __name__ == "__main__":
    unittest.main()
