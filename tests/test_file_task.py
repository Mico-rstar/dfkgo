"""
Upload + Task integration tests for dfkgo.
Requires a running dfkgo server at localhost:8888.
Uses real OSS upload via STS credentials to test full flow.
"""

import time
import unittest
import uuid

import oss2
import requests

BASE_URL = "http://localhost:8888"

# Small test content for OSS upload
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


class TestFileTask(unittest.TestCase):
    """Integration tests for Upload and Task modules."""

    @classmethod
    def setUpClass(cls):
        cls.session = requests.Session()
        cls.unique_email = f"file_test_{uuid.uuid4().hex[:12]}@test.com"
        cls.password = "TestPass123"
        cls.file_id = None
        cls.task_id = None

        # Register
        resp = cls.session.post(
            f"{BASE_URL}/api/auth/register",
            json={"email": cls.unique_email, "password": cls.password},
        )
        assert resp.status_code == 200, f"Register failed: {resp.text}"
        body = resp.json()
        assert body["code"] == 0, f"Register code != 0: {body}"

        # Login
        resp = cls.session.post(
            f"{BASE_URL}/api/auth/login",
            json={"email": cls.unique_email, "password": cls.password},
        )
        assert resp.status_code == 200, f"Login failed: {resp.text}"
        body = resp.json()
        assert body["code"] == 0, f"Login code != 0: {body}"
        cls.token = body["data"]["token"]
        cls.session.headers.update({"Authorization": f"Bearer {cls.token}"})

    # ── Upload: Init ──────────────────────────────────────────

    def test_01_upload_init_success(self):
        """Normal upload init with video/mp4."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "test_video.mp4",
                "fileSize": 10485760,
                "mimeType": "video/mp4",
                "md5": "9e107d9d372bb6826bd81d3542a419d6",
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        data = body["data"]
        self.assertIn("fileId", data)
        self.assertTrue(data["fileId"].startswith("file_"))
        self.assertFalse(data["hit"])
        self.assertIn("sts", data)
        sts = data["sts"]
        self.assertIn("accessKeyId", sts)
        self.assertIn("bucket", sts)
        self.assertIn("objectKey", sts)
        # Save for later tests
        self.__class__.file_id = data["fileId"]
        self.__class__._sts_data = sts

    def test_02_upload_init_unsupported_mime(self):
        """Unsupported mimeType should return 401301."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "doc.pdf",
                "fileSize": 1024,
                "mimeType": "application/pdf",
                "md5": "d41d8cd98f00b204e9800998ecf8427e",
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401301, f"Expected code=401301, got {body}")

    def test_03_upload_init_file_too_large(self):
        """File exceeding 500MB should return 401302."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "huge.mp4",
                "fileSize": 600000000,
                "mimeType": "video/mp4",
                "md5": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401302, f"Expected code=401302, got {body}")

    # ── Upload: Real OSS upload + Callback ────────────────────

    def test_04_upload_oss_and_callback_success(self):
        """Upload real file to OSS via STS, then callback succeeds."""
        self.assertIsNotNone(self.file_id, "file_id not set from test_01")
        sts = self.__class__._sts_data

        # Real OSS upload
        uploaded = oss_upload_via_sts(sts)
        self.assertTrue(uploaded, "Failed to upload to OSS via STS")

        # Callback should succeed now (HeadObject finds the file)
        resp = self.session.post(
            f"{BASE_URL}/api/upload/callback",
            json={"fileId": self.file_id},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Callback failed: {body}")
        self.assertIn("fileId", body["data"])
        self.assertIn("ossUrl", body["data"])

    def test_05_upload_callback_without_oss_file(self):
        """Callback for file not on OSS should return 401303."""
        # Init a new file but don't upload to OSS
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "no_upload.jpg",
                "fileSize": 2048,
                "mimeType": "image/jpeg",
                "md5": uuid.uuid4().hex,
            },
        )
        body = resp.json()
        self.assertEqual(body["code"], 0)
        no_upload_file_id = body["data"]["fileId"]

        resp = self.session.post(
            f"{BASE_URL}/api/upload/callback",
            json={"fileId": no_upload_file_id},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401303, f"Expected 401303, got {body}")

    def test_06_upload_init_same_md5_hit(self):
        """Same MD5 after completed upload should return hit=true (instant upload)."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "test_video_dup.mp4",
                "fileSize": 10485760,
                "mimeType": "video/mp4",
                "md5": "9e107d9d372bb6826bd81d3542a419d6",
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        self.assertTrue(body["data"]["hit"], "Expected hit=true for same MD5")
        self.assertEqual(body["data"]["fileId"], self.file_id, "Should return same fileId")

    def test_07_upload_callback_nonexistent_fileid(self):
        """Callback with non-existent fileId should return 401304."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/callback",
            json={"fileId": "file_00000000000000000000000000000000"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401304, f"Expected code=401304, got {body}")

    # ── Task: Create with completed file ──────────────────────

    def test_08_task_create_success(self):
        """Create task with completed file (from test_04)."""
        self.assertIsNotNone(self.file_id, "file_id not set")
        resp = self.session.post(
            f"{BASE_URL}/api/tasks",
            json={"fileId": self.file_id, "modality": "video"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Task create failed: {body}")
        data = body["data"]
        self.assertIn("taskId", data)
        self.assertTrue(data["taskId"].startswith("task_"))
        self.assertEqual(data["status"], "pending")
        self.__class__.task_id = data["taskId"]

    def test_09_task_create_missing_fields(self):
        """Missing fileId should return non-zero code."""
        resp = self.session.post(
            f"{BASE_URL}/api/tasks",
            json={"modality": "video"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")

    def test_10_task_create_invalid_modality(self):
        """Invalid modality should return non-zero code."""
        resp = self.session.post(
            f"{BASE_URL}/api/tasks",
            json={"fileId": self.file_id or "file_fake", "modality": "pdf"},
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")

    def test_11_task_create_nonexistent_fileid(self):
        """Non-existent fileId should return non-zero code."""
        resp = self.session.post(
            f"{BASE_URL}/api/tasks",
            json={
                "fileId": "file_00000000000000000000000000000000",
                "modality": "video",
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertNotEqual(body["code"], 0, f"Expected non-zero code, got {body}")

    # ── Task: Status, Result, Cancel ──────────────────────────

    def test_12_task_query_status(self):
        """Query task status."""
        self.assertIsNotNone(self.task_id, "task_id not set from test_08")
        resp = self.session.get(f"{BASE_URL}/api/tasks/{self.task_id}")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Task status query failed: {body}")
        data = body["data"]
        self.assertEqual(data["taskId"], self.task_id)
        self.assertIn("status", data)
        self.assertIn("modality", data)
        self.assertIn("createdAt", data)

    def test_13_task_wait_and_check_result(self):
        """Wait for worker to process task (ModelServer unavailable → failed)."""
        self.assertIsNotNone(self.task_id, "task_id not set")
        # Wait for worker to pick up and fail (ModelServer not running)
        time.sleep(5)

        resp = self.session.get(f"{BASE_URL}/api/tasks/{self.task_id}")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0)
        status = body["data"]["status"]
        self.assertIn(status, ["pending", "processing", "completed", "failed"])

        # Query result
        resp = self.session.get(f"{BASE_URL}/api/tasks/{self.task_id}/result")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        if status == "failed":
            self.assertEqual(body["code"], 0, f"Result query failed: {body}")
            self.assertIn("errorMessage", body["data"])
        elif status == "completed":
            self.assertEqual(body["code"], 0)
            self.assertIn("result", body["data"])

    def test_14_task_cancel_flow(self):
        """Create a new task and immediately cancel it."""
        # Init + upload + callback a new file
        new_md5 = uuid.uuid4().hex
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "cancel_test.png",
                "fileSize": 1024,
                "mimeType": "image/png",
                "md5": new_md5,
            },
        )
        body = resp.json()
        self.assertEqual(body["code"], 0)
        cancel_file_id = body["data"]["fileId"]
        sts = body["data"]["sts"]

        # Real OSS upload
        uploaded = oss_upload_via_sts(sts)
        self.assertTrue(uploaded, "Failed to upload to OSS")

        # Callback
        resp = self.session.post(
            f"{BASE_URL}/api/upload/callback",
            json={"fileId": cancel_file_id},
        )
        self.assertEqual(resp.json()["code"], 0, f"Callback failed: {resp.json()}")

        # Create task
        resp = self.session.post(
            f"{BASE_URL}/api/tasks",
            json={"fileId": cancel_file_id, "modality": "image"},
        )
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Task create failed: {body}")
        cancel_task_id = body["data"]["taskId"]

        # Cancel immediately (should be pending)
        resp = self.session.post(f"{BASE_URL}/api/tasks/{cancel_task_id}/cancel")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Cancel failed: {body}")

        # Cancel again → should fail with 401403
        resp = self.session.post(f"{BASE_URL}/api/tasks/{cancel_task_id}/cancel")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401403, f"Expected 401403, got {body}")

    def test_15_query_nonexistent_task(self):
        """Query non-existent taskId should return 401402."""
        fake_task_id = "task_00000000000000000000000000000000"
        resp = self.session.get(f"{BASE_URL}/api/tasks/{fake_task_id}")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401402, f"Expected code=401402, got {body}")

    def test_16_cancel_nonexistent_task(self):
        """Cancel non-existent task should return 401402."""
        fake_task_id = "task_00000000000000000000000000000000"
        resp = self.session.post(f"{BASE_URL}/api/tasks/{fake_task_id}/cancel")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401402, f"Expected code=401402, got {body}")

    def test_17_result_nonexistent_task(self):
        """Result query for non-existent task should return 401402."""
        fake_task_id = "task_00000000000000000000000000000000"
        resp = self.session.get(f"{BASE_URL}/api/tasks/{fake_task_id}/result")
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 401402, f"Expected code=401402, got {body}")

    # ── Upload: Additional MIME types ─────────────────────────

    def test_18_upload_init_image_jpeg(self):
        """Upload init with image/jpeg should succeed."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "photo.jpg",
                "fileSize": 2048000,
                "mimeType": "image/jpeg",
                "md5": uuid.uuid4().hex,
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")
        self.assertFalse(body["data"]["hit"])

    def test_19_upload_init_audio_mpeg(self):
        """Upload init with audio/mpeg should succeed."""
        resp = self.session.post(
            f"{BASE_URL}/api/upload/init",
            json={
                "fileName": "audio.mp3",
                "fileSize": 5000000,
                "mimeType": "audio/mpeg",
                "md5": uuid.uuid4().hex,
            },
        )
        self.assertEqual(resp.status_code, 200)
        body = resp.json()
        self.assertEqual(body["code"], 0, f"Expected code=0, got {body}")


if __name__ == "__main__":
    unittest.main(verbosity=2)
