"""Upload a release archive and verify its Railway deployment without logging secrets."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request

ORIGIN = "https://myjournal-ink.up.railway.app"


class DeploymentError(Exception):
    """Safe CI diagnostic."""


def cli(*args):
    try:
        result = subprocess.run(["railway", *args], capture_output=True, text=True, timeout=120, check=True)
        return json.loads(result.stdout)
    except (subprocess.SubprocessError, ValueError, OSError) as error:
        raise DeploymentError("Railway CLI failed; inspect the dashboard before retrying") from error


def wait_for_success(fetch, deployment_id, deadline, sleep=time.sleep, now=time.monotonic):
    pending = {"INITIALIZING", "BUILDING", "DEPLOYING", "WAITING", "QUEUED"}
    while now() < deadline:
        deployments = fetch()
        if not isinstance(deployments, list):
            raise DeploymentError("Invalid Railway deployment list")
        match = next((item for item in deployments if item.get("id") == deployment_id), None)
        if match:
            status = match.get("status")
            if status == "SUCCESS":
                return
            if status not in pending:
                raise DeploymentError("Railway deployment failed or has unsupported status")
        sleep(10)
    raise DeploymentError("Railway deployment timed out; it may still be running")


def main():
    if len(sys.argv) != 3 or not re.fullmatch(r"[0-9a-f]{40}", sys.argv[2]):
        raise DeploymentError("Expected release archive path and commit SHA")
    source, commit = sys.argv[1:]
    for key in ("RAILWAY_TOKEN", "RAILWAY_SERVICE_ID", "RAILWAY_ENVIRONMENT_ID", "RAILWAY_PROJECT_ID"):
        if not os.environ.get(key):
            raise DeploymentError(f"Missing {key}")
    service = os.environ["RAILWAY_SERVICE_ID"]
    environment = os.environ["RAILWAY_ENVIRONMENT_ID"]
    project = os.environ["RAILWAY_PROJECT_ID"]
    # Only archive contents are uploaded: no working-tree .env, credentials or private notes.
    os.chdir(Path(source).resolve())
    deploy = cli("up", "--project", project, "--service", service, "--environment", environment,
                 "--detach", "--json", "--message", f"Release {commit}")
    if not isinstance(deploy, dict) or not deploy.get("deploymentId"):
        raise DeploymentError("Missing deployment ID; inspect Railway before retrying")
    deployment_id = deploy["deploymentId"]
    wait_for_success(lambda: cli("deployment", "list", "--service", service, "--environment", environment,
                                 "--json", "--limit", "100"), deployment_id, time.monotonic() + 1200)
    for attempt in range(12):
        try:
            with urllib.request.urlopen(ORIGIN + "/healthz", timeout=15) as response:
                if response.status == 200 and response.url == ORIGIN + "/healthz":
                    print(f"Railway deployment {deployment_id} succeeded for uploaded release {commit}; health passed")
                    return
        except urllib.error.URLError:
            pass
        time.sleep(5)
    raise DeploymentError("Deployment succeeded but public health verification failed")


if __name__ == "__main__":
    try:
        main()
    except DeploymentError as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
