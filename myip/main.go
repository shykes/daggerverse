package main

import "context"

// A module to detect your public IP
type Myip struct{}

// Return the public IP address of the current Dagger engine
func (m *Myip) IP(ctx context.Context) (string, error) {
	code := `import sys,requests as r; sys.stdout.write(r.get('https://api.ipify.org?format=json').json()['ip'])`
	return dag.InlinePython().WithPackage("requests").Code(code).Stdout(ctx)
}
