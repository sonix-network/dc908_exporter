package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/golang/glog"
	pb "github.com/sonix-network/dc908_exporter/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 8888, "port to listen on")
)

type Server struct {
	s      *grpc.Server
	lis    net.Listener
	config *Config
	pb.UnimplementedGNMIDialoutServer
}

type Config struct {
	Port int64
}

func NewServer(config *Config, opts []grpc.ServerOption) (*Server, error) {
	if config == nil {
		panic("config not provided")
	}

	s := grpc.NewServer(opts...)
	reflection.Register(s)

	srv := &Server{
		s:      s,
		config: config,
	}
	var err error
	if srv.config.Port < 0 {
		srv.config.Port = 0
	}
	srv.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", srv.config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %v", srv.config.Port, err)
	}
	pb.RegisterGNMIDialoutServer(srv.s, srv)
	log.V(1).Infof("Created Server on %s", srv.Address())
	return srv, nil
}

func (srv *Server) Serve() error {
	s := srv.s
	if s == nil {
		return fmt.Errorf("Serve() failed: not initialized")
	}
	return srv.s.Serve(srv.lis)
}

func (srv *Server) Stop() error {
	s := srv.s
	if s == nil {
		return fmt.Errorf("Serve() failed: not initialized")
	}
	srv.s.Stop()
	log.V(1).Infof("Server stopped on %s", srv.Address())
	return nil
}

func (srv *Server) Address() string {
	return srv.lis.Addr().String()
}

func (srv *Server) Port() int64 {
	return srv.config.Port
}

func (srv *Server) Publish(stream pb.GNMIDialout_PublishServer) error {
	ctx := stream.Context()

	pr, ok := peer.FromContext(ctx)
	if !ok {
		return grpc.Errorf(codes.InvalidArgument, "failed to get peer from ctx")
	}
	if pr.Addr == net.Addr(nil) {
		return grpc.Errorf(codes.InvalidArgument, "failed to get peer address")
	}

	c := NewClient(pr.Addr)
	defer c.Close()
	return c.Run(srv, stream)
}

type Client struct {
	addr net.Addr
}

func NewClient(addr net.Addr) *Client {
	return &Client{
		addr: addr,
	}
}

func (c *Client) String() string {
	return c.addr.String()
}

func (c *Client) Run(srv *Server, stream pb.GNMIDialout_PublishServer) (err error) {
	defer log.V(1).Infof("Client %s shutdown", c)

	if stream == nil {
		return grpc.Errorf(codes.FailedPrecondition, "cannot start client: stream is nil")
	}

	for {
		subscribeResponse, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return grpc.Errorf(codes.Aborted, "stream EOF received")
			}
			return grpc.Errorf(grpc.Code(err), "received error from client")
		}

		notif := subscribeResponse.GetUpdate()
		WalkNotification(notif, func(fqn string, _ *time.Time, json string) {
			log.Infof("%s: %s: %s", c.String(), fqn, json)
		}, nil)
	}
}

func (c *Client) Close() {
}

func main() {
	flag.Parse()

	opts := []grpc.ServerOption{}
	cfg := &Config{}
	cfg.Port = int64(*port)
	s, err := NewServer(cfg, opts)
	if err != nil {
		log.Fatalf("Failed to create gNMI server: %v", err)
	}

	log.V(1).Infof("Starting RPC server on address: %s", s.Address())
	s.Serve()
	log.Flush()
}
