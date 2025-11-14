import subprocess, time, httpx
from fastlite import database


print("Building caddy...")
subprocess.run(['/Users/rensdimmendaal/go/bin/xcaddy', 'build', '--with', 'github.com/AnswerDotAI/caddy-sqlite-router=.'], check=True)

print("Starting backend server on :8001...")
backend_proc = subprocess.Popen(['python', '-m', 'http.server', '8001'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

print("Starting caddy...")
caddy_proc = subprocess.Popen(['sudo','./caddy', 'run', '--config', 'Caddyfile_test'], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
time.sleep(2)

try:
    resp = httpx.get('http://localhost:8001', timeout=5)
    assert resp.status_code == 200, "Python server not started!"
    resp = httpx.get('http://localhost:2019/config/', timeout=5)
    assert resp.status_code == 200, "Caddy not started!"
    resp = httpx.get('https://app1.localhost:9090', timeout=5, verify=False)
    assert resp.status_code == 200, "Test failed!"
    print("Test passed!")
finally:
    caddy_proc.terminate()
    backend_proc.terminate()
    caddy_proc.wait()
    backend_proc.wait()