import pytest
import subprocess

@pytest.fixture
def distributed_setup():
    subprocess.run(["go","build","-o","bin/server","./server"])
    subprocess.run(["go","build","-o","bin/client","client/client.py"])
    subprocess.run(["go","build","-o","bin/worker","client/worker.py"])

