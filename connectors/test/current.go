package test

import (
	"context"
	"errors"
	"sync"
)

var currentConnector = struct {
	sync.Mutex
	conn  *TestConnector
	ready chan *TestConnector
}{
	ready: make(chan *TestConnector, 1),
}

func ResetCurrentConnector() {
	currentConnector.Lock()
	currentConnector.conn = nil
	currentConnector.ready = make(chan *TestConnector, 1)
	currentConnector.Unlock()
}

func CurrentConnector() (*TestConnector, bool) {
	currentConnector.Lock()
	defer currentConnector.Unlock()
	if currentConnector.conn == nil {
		return nil, false
	}
	return currentConnector.conn, true
}

func WaitForConnector(ctx context.Context) (*TestConnector, error) {
	currentConnector.Lock()
	conn := currentConnector.conn
	ready := currentConnector.ready
	currentConnector.Unlock()
	if conn != nil {
		return conn, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-ready:
		if conn == nil {
			return nil, errors.New("test connector readiness signaled with nil connector")
		}
		return conn, nil
	}
}

func publishCurrentConnector(conn *TestConnector) {
	currentConnector.Lock()
	currentConnector.conn = conn
	ready := currentConnector.ready
	currentConnector.Unlock()
	select {
	case ready <- conn:
	default:
	}
}
