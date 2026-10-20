"""Upload a release archive and verify its Railway deployment without logging secrets."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from urllib.parse import urlsplit


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


def validate_release_event():
    event_name = os.environ.get("GITHUB_EVENT_NAME")

    if event_name == "workflow_dispatch":
        return

    if event_name != "release":
        raise DeploymentError("Production deployment requires a published GitHub Release")
    try:
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
        release = event["release"]
        tag = release["tag_name"]
        valid_tag = re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?", tag)
        if (event["action"] != "published" or release["prerelease"] or release["draft"]
                or not valid_tag or tag != os.environ.get("RELEASE_TAG")):
            raise DeploymentError("Production deployment requires a regular published release tag")
    except (KeyError, TypeError, ValueError, OSError) as error:
        raise DeploymentError("Invalid GitHub release event") from error


def validate_plan(plan, environment):
    """Limit release-time IaC to settings on the existing production service."""
    if (plan.get("kind") != "railway.config.plan" or plan.get("version") != 1
            or plan.get("environmentId") != environment or plan.get("destructive") is not False):
        raise DeploymentError("Unexpected Railway plan target or destructive changes")
    change_set = plan.get("changeSet", {})
    if (change_set.get("declared") != ["service.myjournal-app"]
            or change_set.get("diagnostics") or not isinstance(change_set.get("changes"), list)):
        raise DeploymentError("Unexpected Railway plan resources or diagnostics")
    for change in change_set["changes"]:
        if (change.get("kind") != "resource.update" or change.get("address") != "service.myjournal-app"
                or change.get("field") not in {"source", "build", "deploy"}
                or change.get("severity") != "safe"):
            raise DeploymentError("Railway plan exceeds release settings; inspect before retrying")
        if change["field"] == "source" and change.get("after") is not None:
            raise DeploymentError("GitHub/image deployment sources are forbidden")
        if change["field"] == "deploy":
            deploy = change.get("after", {})
            if (deploy.get("multiRegionConfig") != {"asia-southeast1-eqsg3a": {"numReplicas": 1}}
                    or deploy.get("healthcheckPath") != "/healthz" or deploy.get("healthcheckTimeout") != 300):
                raise DeploymentError("Railway plan must retain one replica and health checks")


def apply_configuration(project, service, environment):
    with tempfile.TemporaryDirectory(prefix="myjournal-railway-plan-") as folder:
        plan_path = Path(folder) / "plan.json"
        cli("config", "plan", "--environment", environment, "--json", "--out", str(plan_path))
        try:
            plan = json.loads(plan_path.read_text())
        except (ValueError, OSError) as error:
            raise DeploymentError("Missing or invalid Railway configuration plan") from error
        validate_plan(plan, environment)
        # Apply the exact validated change set. Never authorize destructive changes.
        cli("config", "apply", "--environment", environment, "--plan", str(plan_path), "--yes", "--json")
        check = cli("config", "plan", "--environment", environment, "--json")
        if check.get("ok") is not True or check.get("changeSet", {}).get("changes") != []:
            raise DeploymentError("Railway configuration did not converge; inspect before retrying")


def deployment_configuration():
    for key in ("APP_BASE_URL", "RAILWAY_TOKEN", "RAILWAY_SERVICE_ID", "RAILWAY_ENVIRONMENT_ID", "RAILWAY_PROJECT_ID"):
        if not os.environ.get(key, "").strip():
            raise DeploymentError(f"Missing {key}")
    origin = os.environ["APP_BASE_URL"]
    try:
        url = urlsplit(origin)
        valid = (url.scheme == "https" and url.hostname and url.port != 0
                 and url.username is None and url.password is None
                 and url.path in ("", "/") and "?" not in origin and "#" not in origin
                 and not any(char.isspace() for char in origin))
    except ValueError as error:
        raise DeploymentError("APP_BASE_URL must be an HTTPS origin") from error
    if not valid:
        raise DeploymentError("APP_BASE_URL must be an HTTPS origin")
    return (origin.rstrip("/"), os.environ["RAILWAY_PROJECT_ID"],
            os.environ["RAILWAY_SERVICE_ID"], os.environ["RAILWAY_ENVIRONMENT_ID"])


def main():
    validate_release_event()
    if len(sys.argv) != 3 or not re.fullmatch(r"[0-9a-f]{40}", sys.argv[2]):
        raise DeploymentError("Expected release archive path and commit SHA")
    source, commit = sys.argv[1:]
    origin, project, service, environment = deployment_configuration()
    # Only archive contents are uploaded: no working-tree .env, credentials or private notes.
    os.chdir(Path(source).resolve())
    apply_configuration(project, service, environment)
    deploy = cli("up", "--project", project, "--service", service, "--environment", environment,
                 "--detach", "--json", "--message", f"Release {commit}")
    if not isinstance(deploy, dict) or not deploy.get("deploymentId"):
        raise DeploymentError("Missing deployment ID; inspect Railway before retrying")
    deployment_id = deploy["deploymentId"]
    wait_for_success(lambda: cli("deployment", "list", "--service", service, "--environment", environment,
                                 "--json", "--limit", "100"), deployment_id, time.monotonic() + 1200)
    for attempt in range(12):
        try:
            with urllib.request.urlopen(origin + "/healthz", timeout=15) as response:
                if response.status == 200 and response.url == origin + "/healthz":
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
