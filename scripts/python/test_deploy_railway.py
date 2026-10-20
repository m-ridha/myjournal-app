"""Offline deployment lifecycle regressions."""
import importlib.util
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("deploy_railway", Path(__file__).with_name("deploy-railway.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


PROJECT_ID = "test-project"
ENVIRONMENT_ID = "test-production-environment"
SERVICE_ID = "test-service"
ORIGIN = "https://journal.example"
CONFIGURATION = {"APP_BASE_URL": ORIGIN, "RAILWAY_TOKEN": "synthetic-token",
                 "RAILWAY_PROJECT_ID": PROJECT_ID, "RAILWAY_SERVICE_ID": SERVICE_ID,
                 "RAILWAY_ENVIRONMENT_ID": ENVIRONMENT_ID}


class DeploymentTests(unittest.TestCase):
    def plan(self):
        return {"kind": "railway.config.plan", "version": 1, "environmentId": ENVIRONMENT_ID,
                "destructive": False, "changeSet": {"declared": ["service.myjournal-app"],
                "diagnostics": [], "changes": []}}

    def test_plan_rejects_unexpected_changes(self):
        changes = [
            {"kind": "resource.create", "address": "service.myjournal-app", "field": "build", "severity": "safe"},
            {"kind": "resource.delete", "address": "service.other", "severity": "destructive"},
            {"kind": "resource.update", "address": "service.other", "field": "build", "severity": "safe"},
            {"kind": "resource.update", "address": "service.myjournal-app", "field": "variables", "severity": "safe"},
            {"kind": "resource.update", "address": "service.myjournal-app", "field": "source", "severity": "safe", "after": {"repo": "owner/repo"}},
            {"kind": "resource.update", "address": "service.myjournal-app", "field": "deploy", "severity": "safe", "after": {"numReplicas": 2}},
        ]
        for change in changes:
            with self.subTest(change=change), self.assertRaises(module.DeploymentError):
                plan = self.plan()
                plan["changeSet"]["changes"] = [change]
                module.validate_plan(plan, ENVIRONMENT_ID)

    def test_plan_target_and_diagnostics(self):
        module.validate_plan(self.plan(), ENVIRONMENT_ID)
        for update in [{"environmentId": "other"}, {"destructive": True}, {"kind": "unknown"},
                       {"changeSet": {"declared": ["service.other"], "changes": []}},
                       {"changeSet": {"declared": ["service.myjournal-app"], "changes": [], "diagnostics": ["error"]}}]:
            with self.subTest(update=update), self.assertRaises(module.DeploymentError):
                module.validate_plan(self.plan() | update, ENVIRONMENT_ID)

    def test_release_event_gate(self):
        event = {"action": "published", "release": {"tag_name": "v0.2.0", "draft": False, "prerelease": False}}
        cases = [("release", event, True), ("push", event, False), ("workflow_dispatch", event, True)]
        for field, value in [("draft", True), ("prerelease", True), ("tag_name", "main")]:
            changed = copy.deepcopy(event)
            changed["release"][field] = value
            cases.append(("release", changed, False))
        changed = copy.deepcopy(event)
        changed["action"] = "edited"
        cases.append(("release", changed, False))
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "event.json"
            for name, payload, allowed in cases:
                path.write_text(json.dumps(payload))
                with self.subTest(name=name, payload=payload), patch.dict(os.environ, {
                    "GITHUB_EVENT_NAME": name, "GITHUB_EVENT_PATH": str(path), "RELEASE_TAG": "v0.2.0",
                }):
                    if allowed:
                        module.validate_release_event()
                    else:
                        with self.assertRaises(module.DeploymentError):
                            module.validate_release_event()

    def test_configuration_applies_the_validated_plan(self):
        calls = []
        def cli(*args):
            calls.append(args)
            if "--out" in args:
                Path(args[args.index("--out") + 1]).write_text(json.dumps(self.plan()))
            return {"ok": True, "changeSet": {"changes": []}}
        with patch.object(module, "cli", side_effect=cli):
            module.apply_configuration(PROJECT_ID, SERVICE_ID, ENVIRONMENT_ID)
        self.assertEqual([call[:2] for call in calls], [("link", "--project"), ("config", "plan"),
                                                      ("config", "apply"), ("config", "plan")])
        self.assertEqual(calls[1][calls[1].index("--out") + 1], calls[2][calls[2].index("--plan") + 1])
        self.assertIn("--yes", calls[2])
        self.assertNotIn("--confirm-destructive", calls[2])

    def test_invalid_configuration_stops_before_apply(self):
        def cli(*args):
            if "--out" in args:
                plan = self.plan() | {"destructive": True}
                Path(args[args.index("--out") + 1]).write_text(json.dumps(plan))
            return {}
        with patch.object(module, "cli", side_effect=cli) as mocked:
            with self.assertRaises(module.DeploymentError):
                module.apply_configuration(PROJECT_ID, SERVICE_ID, ENVIRONMENT_ID)
        self.assertEqual(mocked.call_count, 2)

    def test_configuration_must_converge(self):
        def cli(*args):
            if "--out" in args:
                Path(args[args.index("--out") + 1]).write_text(json.dumps(self.plan()))
            return {"ok": False, "changeSet": {"changes": []}}
        with patch.object(module, "cli", side_effect=cli):
            with self.assertRaisesRegex(module.DeploymentError, "did not converge"):
                module.apply_configuration(PROJECT_ID, SERVICE_ID, ENVIRONMENT_ID)

    def test_configuration_failure_prevents_source_upload(self):
        env = CONFIGURATION
        with patch.dict(os.environ, env), patch.object(module, "validate_release_event"), \
                patch.object(module.sys, "argv", ["deploy-railway.py", "/synthetic-archive", "a" * 40]), \
                patch.object(module.os, "chdir"), \
                patch.object(module, "apply_configuration", side_effect=module.DeploymentError("apply failed")), \
                patch.object(module, "cli") as cli:
            with self.assertRaisesRegex(module.DeploymentError, "apply failed"):
                module.main()
        cli.assert_not_called()

    def test_missing_configuration_prevents_configuration_and_upload(self):
        for key in CONFIGURATION:
            with self.subTest(key=key), patch.dict(os.environ, CONFIGURATION | {key: ""}), \
                    patch.object(module, "validate_release_event"), \
                    patch.object(module.sys, "argv", ["deploy-railway.py", "/synthetic-archive", "a" * 40]), \
                    patch.object(module, "apply_configuration") as apply, patch.object(module, "cli") as cli:
                with self.assertRaisesRegex(module.DeploymentError, "Missing " + key):
                    module.main()
                apply.assert_not_called()
                cli.assert_not_called()

    def test_deployment_configuration_uses_environment(self):
        with patch.dict(os.environ, CONFIGURATION | {"APP_BASE_URL": ORIGIN + "/"}):
            self.assertEqual(module.deployment_configuration(), (ORIGIN, PROJECT_ID, SERVICE_ID, ENVIRONMENT_ID))

    def test_invalid_health_origin(self):
        for origin in ["", "http://journal.example", "https://journal.example/path", "https://user:password@journal.example",
                       "https://journal.example?query=value", "https://journal.example#fragment", "https://journal.example?", "https://journal.example#", "https://", "https://journal.example:invalid",
                       "https://journal.example:0", " https://journal.example", "https://journal.example/\n"]:
            with self.subTest(origin=origin), patch.dict(os.environ, CONFIGURATION | {"APP_BASE_URL": origin}):
                with self.assertRaises(module.DeploymentError):
                    module.deployment_configuration()

    def test_release_upload_and_health_follow_configuration(self):
        calls = []
        def cli(*args):
            calls.append(args)
            if args[0] == "up":
                return {"deploymentId": "release-deployment"}
            return [{"id": "release-deployment", "status": "SUCCESS"}]
        env = CONFIGURATION
        with patch.dict(os.environ, env), patch.object(module, "validate_release_event"), \
                patch.object(module.sys, "argv", ["deploy-railway.py", "/synthetic-archive", "a" * 40]), \
                patch.object(module.os, "chdir") as chdir, \
                patch.object(module, "apply_configuration", side_effect=lambda *args: calls.append(("apply",))), \
                patch.object(module, "cli", side_effect=cli), \
                patch.object(module.urllib.request, "urlopen") as health, patch("builtins.print") as output:
            response = health.return_value.__enter__.return_value
            response.status = 200
            response.url = ORIGIN + "/healthz"
            module.main()
        chdir.assert_called_once_with(Path("/synthetic-archive"))
        self.assertEqual(calls[0], ("apply",))
        self.assertEqual(calls[1], ("up", "--project", PROJECT_ID, "--service", SERVICE_ID,
                                  "--environment", ENVIRONMENT_ID, "--detach", "--json", "--message", "Release " + "a" * 40))
        self.assertEqual(calls[2][0:2], ("deployment", "list"))
        health.assert_called_once_with(ORIGIN + "/healthz", timeout=15)
        self.assertIn("release-deployment", output.call_args.args[0])

    def test_go_configuration_graph(self):
        # Evaluate offline using the same entry point and context as the CLI.
        root = Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory() as folder:
            target = Path(folder)
            for name in ("go.mod", "go.sum", "railway.go"):
                shutil.copy(root / ".railway" / name, target / name)
            (target / "eval.go").write_text('''package main
import ("encoding/json"; "os"; "github.com/railwayapp/railway-go-sdk")
func main() {
 var ctx railway.Context
 if err := json.Unmarshal([]byte(os.Getenv("TEST_RAILWAY_CONTEXT")), &ctx); err != nil { panic(err) }
 if err := json.NewEncoder(os.Stdout).Encode(Railway(ctx)); err != nil { panic(err) }
}
''')
            context = {"ProjectID": PROJECT_ID, "EnvironmentID": ENVIRONMENT_ID, "Environment": "production"}
            def evaluate(ctx, overrides=None):
                return subprocess.run(["go", "run", "railway.go", "eval.go"], cwd=target,
                                      env=os.environ | CONFIGURATION | (overrides or {}) | {"TEST_RAILWAY_CONTEXT": json.dumps(ctx)},
                                      capture_output=True, text=True, timeout=120)
            result = evaluate(context)
            self.assertEqual(result.returncode, 0, result.stderr)
            graph = json.loads(result.stdout)
            self.assertEqual(graph["variables"], {"managed": False})
            self.assertEqual(len(graph["resources"]), 1)
            service = graph["resources"][0]
            self.assertEqual(service["name"], "myjournal-app")
            self.assertNotIn("source", service)
            self.assertNotIn("variables", service)
            self.assertEqual(service["build"]["builder"], "DOCKERFILE")
            self.assertEqual(service["deploy"]["multiRegionConfig"], {"asia-southeast1-eqsg3a": {"numReplicas": 1}})
            self.assertEqual(service["deploy"]["healthcheckPath"], "/healthz")
            self.assertEqual(service["deploy"]["healthcheckTimeout"], 300)
            self.assertEqual(service["deploy"]["restartPolicyMaxRetries"], 3)
            for key in context:
                with self.subTest(key=key):
                    self.assertNotEqual(evaluate(context | {key: "other"}).returncode, 0)

            for key in ("RAILWAY_PROJECT_ID", "RAILWAY_ENVIRONMENT_ID"):
                with self.subTest(missing=key):
                    self.assertNotEqual(evaluate(context, {key: ""}).returncode, 0)

    def test_waits_for_requested_deployment(self):
        replies = iter([[{"id": "other", "status": "SUCCESS"}],
                        [{"id": "target", "status": "DEPLOYING"}],
                        [{"id": "target", "status": "SUCCESS"}]])
        sleeps = []
        module.wait_for_success(lambda: next(replies), "target", 10, sleeps.append, lambda: 0)
        self.assertEqual(sleeps, [10, 10])

    def test_failure_is_not_success(self):
        for status in ("FAILED", "CRASHED", "REMOVED", "unknown"):
            with self.subTest(status=status), self.assertRaises(module.DeploymentError):
                module.wait_for_success(lambda: [{"id": "target", "status": status}], "target", 10, now=lambda: 0)

    def test_timeout(self):
        with self.assertRaisesRegex(module.DeploymentError, "timed out"):
            module.wait_for_success(lambda: [], "target", 0, now=lambda: 0)


if __name__ == "__main__":
    unittest.main()
