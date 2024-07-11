package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	log "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pb "github.com/sonix-network/dc908_exporter/proto"
	"google.golang.org/grpc"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

var (
	gnmiPort = flag.Int("gnmi-port", 8888, "port to listen for gNMI connections on")
	metricPort = flag.Int("metric-port", 9908, "port to listen for Prometheus scrapes on")
	maxConns = flag.Int("max-gnmi-connections", 100, "maximum number of concurrent gNMI connecitons")
)

type Server struct {
	s      *grpc.Server
	lis    net.Listener
	config *Config
	mr     *metricRegistry

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
	srv.lis = netutil.LimitListener(srv.lis, *maxConns)
	pb.RegisterGNMIDialoutServer(srv.s, srv)
	log.V(1).Infof("Created server on %s with maximum gNMI connections set to %d", srv.Address(), *maxConns)

	srv.mr = NewMetricRegistry()
	return srv, nil
}

func (srv *Server) PrometheusRegistry() *prometheus.Registry {
	return srv.mr.PrometheusRegistry()
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

	c := NewClient(pr.Addr, srv.mr)
	defer c.Close()
	return c.Run(srv, stream)
}

type Client struct {
	addr net.Addr
	mr   *metricRegistry
}

func NewClient(addr net.Addr, mr *metricRegistry) *Client {
	return &Client{
		addr: addr,
		mr:   mr,
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
			// TODO: Verify that the timestamp is not too far off
			c.mr.Update(fqn, json)
		}, nil)
	}
}

func (c *Client) Close() {
}

func main() {
	flag.Parse()

	opts := []grpc.ServerOption{}
	cfg := &Config{}
	cfg.Port = int64(*gnmiPort)
	s, err := NewServer(cfg, opts)
	if err != nil {
		log.Fatalf("Failed to create gNMI server: %v", err)
	}

	reg := s.PrometheusRegistry()
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *metricPort), nil))
	}()

	log.V(1).Infof("Starting RPC server on address: %s", s.Address())
	s.Serve()
	log.Flush()
}
