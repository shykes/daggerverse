package main

import "context"

// A module to detect your public IP
type Myip struct{}

// Return the public IP address of the current Dagger engine
func (m *Myip) IP(ctx context.Context) (string, error) {
	code, err := m.Code().Contents(ctx)
	if err != nil {
		return code, err
	}
	return dag.InlinePython().WithPackage("requests").Code(code).Stdout(ctx)
}

func (m *Myip) Code() *File {
	return dag.Host().File("myip.py")
}
