#!/usr/bin/env python3
"""Create private native BuildKit/containerd configuration for this CI job."""
import argparse
import base64
import json
import os
from pathlib import Path
import re
import shutil
import tempfile

# Native registry mappings; image versions remain in their existing upstream inputs.
ROUTES = {
    "docker.io": "docker-hub",
    "gcr.io": "mirror/gcr.io",
    "ghcr.io": "ghcr",
    "quay.io": "quay",
    "registry.k8s.io": "k8s",
}


def configure(image, runner_temp, docker_config, kind):
    registry = image.split("/", 1)[0]
    if registry != "202011926204.dkr.ecr.us-east-1.amazonaws.com":
        raise ValueError("Expected the DRJ CI cache registry")
    authorization = None
    if kind:
        auth = json.loads((docker_config / "config.json").read_text())["auths"][registry]["auth"]
        decoded = base64.b64decode(auth, validate=True)
        if not decoded.startswith(b"AWS:") or len(decoded) <= 4 or not re.fullmatch(r"[A-Za-z0-9+/=]+", auth):
            raise ValueError("Expected Docker's ECR login")
        authorization = "Basic " + auth
    directory = Path(tempfile.mkdtemp(prefix="pvc-image-cache-", dir=runner_temp))
    try:
        buildkit = "\n".join(
            f'[registry.{json.dumps(origin)}]\n  mirrors = [{json.dumps(registry + "/" + prefix)}]\n'
            for origin, prefix in ROUTES.items()
        )
        (directory / "buildkitd.toml").write_text(buildkit)
        (directory / "buildkitd.toml").chmod(0o600)
        hosts = directory / "hosts"
        hosts.mkdir(mode=0o700)
        if kind:
            for origin, prefix in ROUTES.items():
                target = hosts / origin
                target.mkdir(mode=0o700)
                # An explicit server replaces the origin. A host-only mirror would
                # append the public origin as containerd's fallback server.
                config = (
                    f'server = {json.dumps("https://" + registry + "/v2/" + prefix)}\n'
                    'capabilities = ["pull", "resolve"]\noverride_path = true\n'
                    f'[header]\n  authorization = [{json.dumps(authorization)}]\n'
                )
                (target / "hosts.toml").write_text(config)
                (target / "hosts.toml").chmod(0o600)
        return registry, directory
    except BaseException:
        shutil.rmtree(directory)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True)
    parser.add_argument("--kind", action="store_true")
    args = parser.parse_args()
    directory = None
    try:
        registry, directory = configure(
            args.image, Path(os.environ["RUNNER_TEMP"]),
            Path(os.environ.get("DOCKER_CONFIG", str(Path.home() / ".docker"))), args.kind,
        )
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            output.write(f"registry={registry}\nconfig={directory / 'buildkitd.toml'}\n")
        with open(os.environ["GITHUB_ENV"], "a") as environment:
            environment.write(
                f"IMAGE_CACHE_REGISTRY={registry}\nIMAGE_CACHE_CONFIG={directory}\n"
                f"IMAGE_CACHE_DIRECTORY={directory / 'hosts'}\nDOCKER_BUILD=docker buildx build --load\n"
            )
    except Exception:
        if directory is not None:
            shutil.rmtree(directory)
        # Docker auth/config exceptions can contain credential values. Never log them.
        raise SystemExit("Could not prepare the private image cache configuration") from None


if __name__ == "__main__":
    main()
