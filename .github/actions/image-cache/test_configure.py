"""Run locally with Python 3.11+: private credential and native route checks."""
import base64
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import tomllib
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location("configure", Path(__file__).with_name("configure.py"))
configure = importlib.util.module_from_spec(spec)
spec.loader.exec_module(configure)
REGISTRY = "202011926204.dkr.ecr.us-east-1.amazonaws.com"
IMAGE = REGISTRY + "/docker-hub/moby/buildkit:buildx-stable-1"


class NativeConfigurationTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.docker = self.root / "docker"
        self.docker.mkdir()
        self.auth = base64.b64encode(b"AWS:disposable-test-password").decode()
        (self.docker / "config.json").write_text(json.dumps({"auths": {REGISTRY: {"auth": self.auth}}}))

    def test_kind_has_only_authenticated_pull_servers_and_private_files(self):
        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout):
            registry, directory = configure.configure(IMAGE, self.root, self.docker, True)
        self.assertEqual(stdout.getvalue(), "")
        self.assertEqual(registry, REGISTRY)
        self.assertEqual(directory.stat().st_mode & 0o777, 0o700)
        buildkit = tomllib.loads((directory / "buildkitd.toml").read_text())
        self.assertNotIn(self.auth, (directory / "buildkitd.toml").read_text())
        for origin, mapping in buildkit["registry"].items():
            hosts_file = directory / "hosts" / origin / "hosts.toml"
            native = tomllib.loads(hosts_file.read_text())
            endpoint = mapping["mirrors"][0].split("/", 1)
            self.assertEqual(native["server"], "https://" + endpoint[0] + "/v2/" + endpoint[1])
            self.assertEqual(set(native["capabilities"]), {"pull", "resolve"})
            self.assertTrue(native["override_path"])
            self.assertNotIn("host", native)
            self.assertNotIn("skip_verify", native)
            self.assertEqual(native["header"]["authorization"], ["Basic " + self.auth])
            self.assertEqual(hosts_file.stat().st_mode & 0o777, 0o600)
        self.assertEqual(buildkit["registry"]["docker.io"]["mirrors"], [REGISTRY + "/docker-hub"])
        self.assertEqual(buildkit["registry"]["gcr.io"]["mirrors"], [REGISTRY + "/mirror/gcr.io"])

    def test_build_only_does_not_read_or_copy_credentials(self):
        (self.docker / "config.json").unlink()
        _, directory = configure.configure(IMAGE, self.root, self.docker, False)
        self.assertEqual(list((directory / "hosts").iterdir()), [])

    def test_rejects_another_registry_before_creating_files(self):
        with self.assertRaises(ValueError):
            configure.configure(IMAGE.replace("us-east-1", "us-west-2"), self.root, self.docker, True)
        self.assertEqual(list(self.root.glob("pvc-image-cache-*")), [])

    def test_bad_credentials_never_reach_cli_output_or_disk(self):
        (self.docker / "config.json").write_text("invalid-secret-credential")
        env = dict(os.environ, RUNNER_TEMP=str(self.root), DOCKER_CONFIG=str(self.docker))
        result = subprocess.run(["python3", str(Path(configure.__file__)), "--image", IMAGE, "--kind"],
                                env=env, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("invalid-secret-credential", result.stdout + result.stderr)
        self.assertNotIn("Traceback", result.stderr)
        self.assertEqual(list(self.root.glob("pvc-image-cache-*")), [])

    def test_partial_write_failure_removes_owned_directory(self):
        with mock.patch.object(Path, "write_text", side_effect=OSError("disk unavailable")):
            with self.assertRaises(OSError):
                configure.configure(IMAGE, self.root, self.docker, True)
        self.assertEqual(list(self.root.glob("pvc-image-cache-*")), [])

    def test_failed_github_output_removes_written_credentials(self):
        with mock.patch.dict(os.environ, RUNNER_TEMP=str(self.root), DOCKER_CONFIG=str(self.docker),
                             GITHUB_OUTPUT=str(self.root / "missing" / "output")), \
                mock.patch("sys.argv", ["configure.py", "--image", IMAGE, "--kind"]):
            with self.assertRaises(SystemExit):
                configure.main()
        self.assertEqual(list(self.root.glob("pvc-image-cache-*")), [])


if __name__ == "__main__":
    unittest.main()
