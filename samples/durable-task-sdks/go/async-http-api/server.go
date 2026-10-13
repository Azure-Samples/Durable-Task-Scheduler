package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type httpServer struct {
	URL    string
	server *http.Server
	done   <-chan error
}

func startHTTPServer(ctx context.Context, address string, handler http.Handler) (*httpServer, error) {
	address, err := loopbackAddress(address)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	server := &http.Server{
		Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	done := make(chan error, 1)
	go func() {
		defer close(done)
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()
	return &httpServer{URL: "http://" + listener.Addr().String(), server: server, done: done}, nil
}

func (s *httpServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, s.server.Close())
	}
	return errors.Join(err, <-s.done)
}

func serveHTTP(ctx context.Context, address string, handler http.Handler) (err error) {
	server, err := startHTTPServer(ctx, address, handler)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, server.Close()) }()
	fmt.Printf("Async HTTP API listening on %s (until -timeout or Ctrl+C)\n", server.URL)
	select {
	case <-ctx.Done():
		return nil
	case err := <-server.done:
		return err
	}
}

func loopbackAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("listen address must be a loopback host:port: %w", err)
	}
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return "", errors.New("listen address must use a loopback IP or localhost")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return "", errors.New("invalid listen port")
	}
	return net.JoinHostPort(host, port), nil
}
