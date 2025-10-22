package main

import (
    "log"
    "net"

    "google.golang.org/grpc"
)

type mockServer struct {
    // TODO: implement proto interface
}

func main() {
    lis, err := net.Listen("tcp", ":9091")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    // TODO: register service

    log.Println("Mock control server listening on :9091")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
