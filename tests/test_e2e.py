import pytest
import subprocess
import shutil
import os
import re
import time
import logging

def debug(s):
    # Helper function to debug outputs
    print("------")
    print(s)
    print("------")

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# @pytest.fixture(scope="session")
# def distributed_setup():
#     try:
#         subprocess.run(["go", "build", "-o", "bin/server", "./server"], check=True)
#         subprocess.run(["go", "build", "-o", "bin/client", "./client"], check=True)
#         subprocess.run(["go", "build", "-o", "bin/worker", "./worker"], check=True)
#         logging.info("[+] Created binaries successfully")
#     except subprocess.CalledProcessError as e:
#         raise RuntimeError(f"[-] Error creating binaries: {e}") from e

@pytest.fixture(scope="session", autouse=True)
def server_and_worker():

    try:
        process = subprocess.run(list("docker compose up -d --scale worker=3".split(" ")))
        time.sleep(3)
    except Exception as e:
        raise RuntimeError(f"[-] Error running docker compose: {e}") from e

    yield process

    # Teardown — nuke the db.
    try:
        subprocess.run(list("docker compose down -v".split(" ")))
    except Exception as e:
        raise RuntimeError(f"[-] Error in docker compose: {e}") from e



def extract_job_id(stderr: str) -> str:
    """Parse the job ID from the '[+] Job Accepted by the server <id>' log line."""
    match = re.search(r"Job Accepted by the server\s+(\d+)", stderr)
    if not match:
        raise ValueError(
            f"Could not find job ID in client output. stderr was:\n{stderr}"
        )
    return match.group(1)


def test_submit_one_off_job():
    out = subprocess.run(
        ["./bin/client", "submit", "-i", "alpine", "-c",
         "echo Hello from test_submit_one_off_job", "-s", "-1"],
        capture_output=True, text=True, check=True,
    )
    time.sleep(2)
    assert out.returncode == 0
    assert "[+] Job Accepted by the server" in out.stderr

def test_check_status():
    out = subprocess.run(
        ["./bin/client", "submit", "-i", "alpine", "-c",
         "echo Hello from test_check_status", "-s", "-1"],
        capture_output=True, text=True, check=True,
    )
    time.sleep(5)

    job_id = extract_job_id(out.stderr)

    out = subprocess.run(
        ["./bin/client", "status", "-j", str(job_id)],
        capture_output=True, text=True, check=True,
    )
    time.sleep(2)

    assert out.returncode == 0
    assert "Hello from test_check_status" in out.stderr

def test_failed_submit_job():
    out = subprocess.run(
        ["./bin/client", "submit", "-i", "this-image-does-not-exist", "-c",
         "echo Hello from test_failed_job_submit", "-s", "-1"],
        capture_output=True, text=True, check=True,
    )
    time.sleep(5)

    job_id = extract_job_id(out.stderr)

    out = subprocess.run(
        ["./bin/client", "status", "-j", str(job_id)],
        capture_output=True, text=True, check=True,
    )
    time.sleep(1)

    assert out.returncode == 0
    assert "FAILED" in out.stderr

def test_cron_job():
    out = subprocess.run(
        ["./bin/client", "submit", "-i", "alpine", "-c",
         "echo Hello from test_cron_job", "-s", "2"],
        capture_output=True, text=True, check=True,
    )
    time.sleep(7)

    job_id = extract_job_id(out.stderr)

    out = subprocess.run(
        ["./bin/client", "status", "-j", str(job_id)],
        capture_output=True, text=True, check=True,
    )
    time.sleep(1)

    matches = re.findall("Hello from test_cron_job", out.stderr)

    assert out.returncode == 0
    assert len(matches) >= 2
