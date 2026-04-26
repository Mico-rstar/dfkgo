"""
History + E2E integration tests for dfkgo.
Runs against a live server at localhost:8888.
Uses real OSS upload via STS credentials to create tasks for full testing.
"""

import time
import unittest
import uuid

import oss2
import requests

BASE_URL = "http://localhost:8888"
TEST_FILE_CONTENT = b"fake file content for integration test"


def oss_upload_via_sts(sts_data, content=TEST_FILE_CONTENT):
    """Upload content to OSS using STS credentials from upload/init response."""
    auth = oss2.StsAuth(
        sts_data["accessKeyId"],
        sts_data["accessKeySecret"],
        sts_data["securityToken"],
    )
    endpoint = f"https://{sts_data['endpoint']}"
    bucket = oss2.Bucket(auth, endpoint, sts_data["bucket"])
    result = bucket.put_object(sts_data["objectKey"], content)
    return result.status == 200


def register_and_login(session, email=None, password="Test@12345"):
    """Register a new user and login, return (email, token)."""
    if email is None:
        uid = uuid.uuid4().hex[:12]
        email = f"hist_test_{uid}@test.com"

    r = session.post(f"{BASE_URL}/api/auth/register", json={
        "email": email,
        "password": password,
    })
    data = r.json()
    assert data["code"] == 0, f"Register failed: {data}"

    r = session.post(f"{BASE_URL}/api/auth/login", json={
        "email": email,
        "password": password,
    })
    data = r.json()
    assert data["code"] == 0, f"Login failed: {data}"
    token = data["data"]["token"]
    session.headers.update({"Authorization": f"Bearer {token}"})
    return email, token


def create_task_full_flow(session, file_name, file_size, mime_type, modality):
    """Upload init → real OSS upload → callback → create task. Returns taskId."""
    md5 = uuid.uuid4().hex

    # Upload init
    r = session.post(f"{BASE_URL}/api/upload/init", json={
        "fileName": file_name,
        "fileSize": file_size,
        "mimeType": mime_type,
        "md5": md5,
    })
    data = r.json()
    assert data["code"] == 0, f"Upload init failed: {data}"
    file_id = data["data"]["fileId"]
    sts = data["data"]["sts"]

    # Real OSS upload
    uploaded = oss_upload_via_sts(sts)
    assert uploaded, "Failed to upload to OSS via STS"

    # Callback
    r = session.post(f"{BASE_URL}/api/upload/callback", json={"fileId": file_id})
    data = r.json()
    assert data["code"] == 0, f"Upload callback failed: {data}"

    # Create task
    r = session.post(f"{BASE_URL}/api/tasks", json={
        "fileId": file_id,
        "modality": modality,
    })
    data = r.json()
    assert data["code"] == 0, f"Create task failed: {data}"
    return data["data"]["taskId"]


class TestHistoryWithoutTasks(unittest.TestCase):
    """History API tests that work without pre-existing tasks."""

    session: requests.Session

    @classmethod
    def setUpClass(cls):
        cls.session = requests.Session()
        register_and_login(cls.session)

    def test_01_history_list_empty_user(self):
        """GET /api/history for a fresh user returns empty items."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 10})
        data = r.json()
        self.assertEqual(data["code"], 0, f"Unexpected: {data}")
        self.assertIn("items", data["data"])
        self.assertIn("pagination", data["data"])
        self.assertEqual(len(data["data"]["items"]), 0)
        self.assertEqual(data["data"]["pagination"]["total"], 0)

    def test_02_history_list_defaults(self):
        """GET /api/history without params uses defaults."""
        r = self.session.get(f"{BASE_URL}/api/history")
        data = r.json()
        self.assertEqual(data["code"], 0)
        self.assertIn("items", data["data"])
        self.assertIn("pagination", data["data"])

    def test_03_history_list_out_of_range(self):
        """GET /api/history?page=100&limit=10 returns empty items."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 100, "limit": 10})
        data = r.json()
        self.assertEqual(data["code"], 0)
        self.assertEqual(len(data["data"]["items"]), 0)

    def test_04_history_stats_empty(self):
        """GET /api/history/stats for fresh user returns zeros."""
        r = self.session.get(f"{BASE_URL}/api/history/stats")
        data = r.json()
        self.assertEqual(data["code"], 0, f"Stats failed: {data}")
        stats = data["data"]
        self.assertIn("total", stats)
        self.assertIsInstance(stats["total"], int)
        self.assertEqual(stats["total"], 0)
        self.assertIn("byModality", stats)
        self.assertIsInstance(stats["byModality"], dict)
        self.assertIn("byCategory", stats)
        self.assertIsInstance(stats["byCategory"], dict)

    def test_05_history_delete_nonexistent(self):
        """DELETE /api/history/{taskId} with nonexistent task should return error."""
        r = self.session.delete(f"{BASE_URL}/api/history/task_00000000000000000000000000000000")
        data = r.json()
        self.assertEqual(data["code"], 401402, f"Expected 401402 for nonexistent task, got {data}")

    def test_06_history_batch_delete_empty(self):
        """POST /api/history/batch-delete with empty array."""
        r = self.session.post(f"{BASE_URL}/api/history/batch-delete", json={
            "taskIds": [],
        })
        data = r.json()
        self.assertIn("code", data)


class TestHistoryWithTasks(unittest.TestCase):
    """History API tests that require task data.
    Creates real tasks via upload init → OSS upload → callback → create task.
    """

    session: requests.Session
    task_ids: list

    @classmethod
    def setUpClass(cls):
        cls.session = requests.Session()
        register_and_login(cls.session)
        cls.task_ids = []

        # Create 5 tasks with different modalities via real OSS upload
        test_files = [
            ("test1.jpg", 1048576, "image/jpeg", "image"),
            ("test2.mp4", 2097152, "video/mp4", "video"),
            ("test3.wav", 524288, "audio/wav", "audio"),
            ("test4.png", 819200, "image/png", "image"),
            ("test5.mp4", 1572864, "video/mp4", "video"),
        ]

        for file_name, file_size, mime_type, modality in test_files:
            task_id = create_task_full_flow(
                cls.session, file_name, file_size, mime_type, modality
            )
            cls.task_ids.append(task_id)

        # Wait for workers to process (ModelServer unavailable → tasks become failed)
        time.sleep(6)

    def test_07_history_list_basic(self):
        """GET /api/history?page=1&limit=10 returns items and pagination."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 10})
        data = r.json()
        self.assertEqual(data["code"], 0, f"Unexpected: {data}")
        self.assertIn("items", data["data"])
        self.assertIn("pagination", data["data"])
        self.assertGreaterEqual(data["data"]["pagination"]["total"], 5)
        self.assertTrue(len(data["data"]["items"]) > 0)
        # Verify item structure
        item = data["data"]["items"][0]
        self.assertIn("taskId", item)
        self.assertIn("fileName", item)
        self.assertIn("modality", item)
        self.assertIn("status", item)
        self.assertIn("createdAt", item)

    def test_08_history_list_pagination(self):
        """GET /api/history?page=1&limit=2 returns at most 2 items."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 2})
        data = r.json()
        self.assertEqual(data["code"], 0)
        self.assertLessEqual(len(data["data"]["items"]), 2)
        self.assertGreaterEqual(data["data"]["pagination"]["pages"], 2)

    def test_09_history_list_page2(self):
        """GET /api/history page 2 returns remaining items."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 2, "limit": 2})
        data = r.json()
        self.assertEqual(data["code"], 0)
        self.assertTrue(len(data["data"]["items"]) > 0)

    def test_10_history_delete_single(self):
        """DELETE /api/history/{taskId} soft-deletes one task."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 100})
        total_before = r.json()["data"]["pagination"]["total"]

        target = self.task_ids[0]
        r = self.session.delete(f"{BASE_URL}/api/history/{target}")
        data = r.json()
        self.assertEqual(data["code"], 0, f"Delete failed: {data}")

        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 100})
        total_after = r.json()["data"]["pagination"]["total"]
        self.assertEqual(total_after, total_before - 1)

    def test_11_history_batch_delete(self):
        """POST /api/history/batch-delete removes multiple tasks."""
        targets = self.task_ids[1:3]
        r = self.session.post(f"{BASE_URL}/api/history/batch-delete", json={
            "taskIds": targets,
        })
        data = r.json()
        self.assertEqual(data["code"], 0, f"Batch delete failed: {data}")
        self.assertEqual(data["data"]["deletedCount"], 2)

    def test_12_history_after_deletes(self):
        """After deleting 3 tasks, list should show 2 remaining."""
        r = self.session.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 100})
        data = r.json()
        self.assertEqual(data["code"], 0)
        self.assertEqual(data["data"]["pagination"]["total"], 2)

    def test_13_history_stats(self):
        """GET /api/history/stats returns totals and breakdowns."""
        r = self.session.get(f"{BASE_URL}/api/history/stats")
        data = r.json()
        self.assertEqual(data["code"], 0, f"Stats failed: {data}")
        stats = data["data"]
        self.assertIn("total", stats)
        self.assertIsInstance(stats["total"], int)
        self.assertIn("byModality", stats)
        self.assertIsInstance(stats["byModality"], dict)
        self.assertIn("byCategory", stats)
        self.assertIsInstance(stats["byCategory"], dict)


class TestE2EFullFlow(unittest.TestCase):
    """End-to-end full flow test with real OSS upload."""

    def test_14_e2e_full_flow(self):
        """Register → login → upload(OSS) → callback → task → wait → result → history → stats."""
        s = requests.Session()
        uid = uuid.uuid4().hex[:12]
        email = f"e2e_{uid}@test.com"
        password = "E2ePass@999"

        # 1. Register
        r = s.post(f"{BASE_URL}/api/auth/register", json={
            "email": email, "password": password,
        })
        self.assertEqual(r.json()["code"], 0, f"Register: {r.json()}")

        # 2. Login
        r = s.post(f"{BASE_URL}/api/auth/login", json={
            "email": email, "password": password,
        })
        resp = r.json()
        self.assertEqual(resp["code"], 0, f"Login: {resp}")
        token = resp["data"]["token"]
        s.headers.update({"Authorization": f"Bearer {token}"})

        # 3. Upload init (image)
        r = s.post(f"{BASE_URL}/api/upload/init", json={
            "fileName": "e2e_image.jpg",
            "fileSize": 2048,
            "mimeType": "image/jpeg",
            "md5": uuid.uuid4().hex,
        })
        resp = r.json()
        self.assertEqual(resp["code"], 0, f"Upload init: {resp}")
        file_id_img = resp["data"]["fileId"]
        sts_img = resp["data"]["sts"]
        self.assertTrue(file_id_img.startswith("file_"))

        # 4. Real OSS upload
        uploaded = oss_upload_via_sts(sts_img)
        self.assertTrue(uploaded, "OSS upload failed")

        # 5. Upload callback
        r = s.post(f"{BASE_URL}/api/upload/callback", json={"fileId": file_id_img})
        resp = r.json()
        self.assertEqual(resp["code"], 0, f"Callback: {resp}")

        # 6. Create image task
        r = s.post(f"{BASE_URL}/api/tasks", json={
            "fileId": file_id_img, "modality": "image",
        })
        resp = r.json()
        self.assertEqual(resp["code"], 0, f"Create task: {resp}")
        task_id_img = resp["data"]["taskId"]
        self.assertEqual(resp["data"]["status"], "pending")

        # 7. Upload + create video task
        r = s.post(f"{BASE_URL}/api/upload/init", json={
            "fileName": "e2e_video.mp4",
            "fileSize": 4096,
            "mimeType": "video/mp4",
            "md5": uuid.uuid4().hex,
        })
        resp = r.json()
        self.assertEqual(resp["code"], 0)
        file_id_vid = resp["data"]["fileId"]
        sts_vid = resp["data"]["sts"]

        uploaded = oss_upload_via_sts(sts_vid)
        self.assertTrue(uploaded)

        r = s.post(f"{BASE_URL}/api/upload/callback", json={"fileId": file_id_vid})
        self.assertEqual(r.json()["code"], 0)

        r = s.post(f"{BASE_URL}/api/tasks", json={
            "fileId": file_id_vid, "modality": "video",
        })
        resp = r.json()
        self.assertEqual(resp["code"], 0)
        task_id_vid = resp["data"]["taskId"]

        # 8. Wait for worker to process (ModelServer unavailable → failed)
        time.sleep(6)

        # 9. Check task status
        r = s.get(f"{BASE_URL}/api/tasks/{task_id_img}")
        resp = r.json()
        self.assertEqual(resp["code"], 0, f"Task status: {resp}")
        status = resp["data"]["status"]
        self.assertIn(status, ["failed", "completed", "processing", "pending"])

        # 10. Check result (if failed, should have errorMessage)
        if status == "failed":
            r = s.get(f"{BASE_URL}/api/tasks/{task_id_img}/result")
            resp = r.json()
            self.assertEqual(resp["code"], 0, f"Task result: {resp}")
            self.assertIn("errorMessage", resp["data"])
            self.assertTrue(len(resp["data"]["errorMessage"]) > 0)

        # 11. History list - should have 2 tasks
        r = s.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 10})
        resp = r.json()
        self.assertEqual(resp["code"], 0)
        total = resp["data"]["pagination"]["total"]
        self.assertEqual(total, 2)
        # Verify items contain expected fields
        items = resp["data"]["items"]
        self.assertEqual(len(items), 2)
        for item in items:
            self.assertIn("taskId", item)
            self.assertIn("fileName", item)
            self.assertIn("modality", item)
            self.assertIn("fileSize", item)

        # 12. Delete one task
        r = s.delete(f"{BASE_URL}/api/history/{task_id_img}")
        self.assertEqual(r.json()["code"], 0)

        # 13. Verify total decreased
        r = s.get(f"{BASE_URL}/api/history", params={"page": 1, "limit": 10})
        resp = r.json()
        self.assertEqual(resp["data"]["pagination"]["total"], 1)

        # 14. Stats
        r = s.get(f"{BASE_URL}/api/history/stats")
        resp = r.json()
        self.assertEqual(resp["code"], 0)
        stats = resp["data"]
        self.assertIn("total", stats)
        self.assertIn("byModality", stats)
        self.assertIn("byCategory", stats)


if __name__ == "__main__":
    unittest.main(verbosity=2)
