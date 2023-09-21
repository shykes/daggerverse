import sys,requests as r
sys.stdout.write(r.get('https://api.ipify.org?format=json').json()['ip'])