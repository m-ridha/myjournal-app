"""Offline deployment lifecycle regressions."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("deploy_railway", Path(__file__).with_name("deploy-railway.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DeploymentTests(unittest.TestCase):
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
