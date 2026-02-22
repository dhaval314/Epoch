import pytest
import subprocess
import shutil
import os
import time

@pytest.fixture(scope="session")
def distributed_setup():
    try:
        subprocess.run(["go","build","-o","bin/server","./server"])
        subprocess.run(["go","build","-o","bin/client","./client"])
        subprocess.run(["go","build","-o","bin/worker","./worker"])
        print("[+] Created binaries successfully")
    except Exception as e:
        print(f"[-] Error creating binaries: {e}")
        exit()

@pytest.fixture(autouse=True)
def server_and_worker(distributed_setup):
    # Deleting the database
    try:
        if os.path.exists("badger"):
            shutil.rmtree("badger")
        print("[+] Deleted database to start fresh")
    except Exception as e:
        print(f"[-] Error deleting database: {e}")
        exit()

    # Starting up the server
    try:
        process = subprocess.Popen(["./bin/server"])
        print("[+] Started server successfully")
        time.sleep(1)

        # Starting the workers

        worker_list = []
        for i in range(1,4):
            try:
                p = subprocess.Popen(["./bin/worker", f"--worker-id={i}"])
                worker_list.append(p)
                time.sleep(1)
                print(f"[+] Started worker {i} successfully")
            except Exception as e:
                print(f"[-] Error starting worker {i}: {e}")
                exit()
        
        yield process

        # Terminate the workers
        for w in worker_list:
            w.terminate()
            w.wait()
            time.sleep(1)
        
        # Terminate the server
        process.terminate()
        process.wait()
        time.sleep(1)

    except Exception as e:
        print(f"[-] Error starting server: {e}")
        exit()


def test_submit_one_off_job():
    # Submit a job using the client
    
    out = subprocess.run(["./bin/client","submit", "-i", "alpine", "-c", "echo Hello from test_submit_job", "-s", "-1"],
                            capture_output=True,
                            text=True,
                            check=True)
    time.sleep(7)
    assert out.returncode == 0
    

    
   
    

    
