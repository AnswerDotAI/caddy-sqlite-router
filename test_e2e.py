import subprocess, time, httpx
from fastlite import database
from pathlib import Path
from dataclasses import dataclass

@dataclass
class Route: domain:str; host:str; port:int

print("Creating test.db...")
dbp = Path('test.db')
if dbp.exists(): dbp.unlink()
db = database(dbp)
routes = db.create(Route, pk='domain')
routes.insert(dict(domain='app1', host='localhost', port=8001))
routes.insert(dict(domain='app2', host='localhost', port=8002))

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
    resp = httpx.get('http://app1.localhost:9090', timeout=5)
    assert resp.status_code == 200, "Test failed!"
finally:
    # caddy_proc.terminate()
    # backend_proc.terminate()
    # caddy_proc.wait()
    # backend_proc.wait()
    # dbp.unlink()
    pass
