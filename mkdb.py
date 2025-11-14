from fastlite import database
from pathlib import Path
from dataclasses import dataclass

@dataclass
class Route: domain:str; host:str; port:int

for p in Path('./').glob("test.db*"): p.unlink(missing_ok=True)
db = database('test.db')
routes = db.create(Route, pk='domain')
routes.insert(dict(domain='app1', host='localhost', port=8001))
routes.insert(dict(domain='app2', host='localhost', port=8002))