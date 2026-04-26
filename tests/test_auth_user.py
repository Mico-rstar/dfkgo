"""
Auth + User integration tests for dfkgo.
Requires a running dfkgo server at localhost:8888.
"""

import unittest
import uuid

import requests

BASE_URL = "http://localhost:8888"


class TestAuthUser(unittest.TestCase):
    """Integration tests for Auth and User modules."""

    @classmethod
    def setUpClass(cls):
        cls.session = requests.Session()
        cls.unique_email = f"auth_test_{uuid.uuid4().hex[:12]}@test.com"
        cls.password = "TestPass123"
        cls.token = None

    # ── Auth: Register ──────────────────────────────────────────

    def test_01_register_success(self):
        """Normal registration with unique email."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/register",
            json={"email": self.unique_email, "password": self.password},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")

    def test_02_register_duplicate_email(self):
        """Duplicate email should return 401009."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/register",
            json={"email": self.unique_email, "password": self.password},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401009, f"Expected code=401009, got {body}")

    def test_03_register_invalid_email(self):
        """Invalid email format should fail."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/register",
            json={"email": "notanemail", "password": self.password},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")

    def test_04_register_short_password(self):
        """Short password should fail."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/register",
            json={"email": f"short_pwd_{uuid.uuid4().hex[:8]}@test.com", "password": "123"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")

    # ── Auth: Login ─────────────────────────────────────────────

    def test_05_login_success(self):
        """Normal login returns token."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/login",
            json={"email": self.unique_email, "password": self.password},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        data = body["data"]
        self.assertIn("token", data)
        self.assertTrue(len(data["token"]) > 0, "Token should not be empty")
        self.assertIn("expiresAt", data)
        self.assertGreater(data["expiresAt"], 0)
        # Save token for subsequent tests
        TestAuthUser.token = data["token"]

    def test_06_login_wrong_password(self):
        """Wrong password should return 401005."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/login",
            json={"email": self.unique_email, "password": "WrongPass999"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401005, f"Expected code=401005, got {body}")

    def test_07_login_nonexistent_user(self):
        """Non-existent user should return 40104."""
        resp = self.session.post(
            f"{BASE_URL}/api/auth/login",
            json={"email": f"nouser_{uuid.uuid4().hex[:8]}@test.com", "password": self.password},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401004, f"Expected code=401004, got {body}")

    # ── User: Profile ──────────────────────────────────────────

    def _auth_headers(self):
        self.assertIsNotNone(self.token, "Token not set; login test must run first")
        return {"Authorization": f"Bearer {self.token}"}

    def test_08_get_profile(self):
        """Get profile returns registered email."""
        resp = self.session.get(
            f"{BASE_URL}/api/user/get-profile",
            headers=self._auth_headers(),
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        data = body["data"]
        self.assertEqual(data["email"], self.unique_email)

    def test_09_update_profile_nickname(self):
        """Update nickname and verify."""
        new_nickname = f"tester_{uuid.uuid4().hex[:6]}"
        resp = self.session.put(
            f"{BASE_URL}/api/user/update-profile",
            headers=self._auth_headers(),
            json={"nickname": new_nickname},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")

        # Verify the change
        resp2 = self.session.get(
            f"{BASE_URL}/api/user/get-profile",
            headers=self._auth_headers(),
        )
        self.assertEqual(resp2.status_code, 200)
        body2 = resp2.json()
        self.assertEqual(body2["code"], 0)
        self.assertEqual(body2["data"]["nickname"], new_nickname)

    # ── User: Avatar ───────────────────────────────────────────

    def test_10_avatar_sts_init(self):
        """Avatar STS init returns credentials."""
        resp = self.session.post(
            f"{BASE_URL}/api/user/avatar-upload/init",
            headers=self._auth_headers(),
            json={"mimeType": "image/jpeg", "fileSize": 102400},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        data = body["data"]
        self.assertIn("accessKeyId", data)
        self.assertIn("objectKey", data)
        self.assertIn("bucket", data)

    def test_11_avatar_callback_no_file(self):
        """Avatar callback with fake objectKey should fail (no OSS file)."""
        resp = self.session.post(
            f"{BASE_URL}/api/user/avatar-upload/callback",
            headers=self._auth_headers(),
            json={"objectKey": "avatars/fake/avatar.jpg"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401202, f"Expected code=401202, got {body}")

    def test_12_fetch_avatar_empty(self):
        """New user should have empty avatar URL."""
        resp = self.session.get(
            f"{BASE_URL}/api/user/fetch-avatar",
            headers=self._auth_headers(),
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        self.assertEqual(body["data"]["avatarUrl"], "")

    # ── Auth: No Token ─────────────────────────────────────────

    def test_13_no_token_access(self):
        """Accessing protected endpoint without token should fail."""
        resp = self.session.get(f"{BASE_URL}/api/user/get-profile")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")


if __name__ == "__main__":
    unittest.main(verbosity=2)
