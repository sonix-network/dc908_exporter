package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pb "github.com/sonix-network/dc908_exporter/proto"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	gnmiPort   = flag.Int("gnmi-port", 8888, "port to listen for gNMI connections on")
	metricPort = flag.Int("metric-port", 9908, "port to listen for Prometheus scrapes on")
	maxConns   = flag.Int("max-gnmi-connections", 100, "maximum number of concurrent gNMI connecitons")
)

type Server struct {
	s             *grpc.Server
	lis           net.Listener
	config        *Config
	lock          sync.RWMutex
	gnmiMetricMap map[string]*metricRegistry

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
		s:             s,
		config:        config,
		gnmiMetricMap: make(map[string]*metricRegistry),
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

	mr := NewMetricRegistry()
	ip := pr.Addr.(*net.TCPAddr).IP.String()
	srv.lock.Lock()
	if _, exists := srv.gnmiMetricMap[ip]; exists {
		srv.lock.Unlock()
		log.Errorf("Duplicate gNMI session from sender %q, rejecting", ip)
		return grpc.Errorf(codes.AlreadyExists, "gNMI session for this client already in progress")
	}
	srv.gnmiMetricMap[ip] = mr
	srv.lock.Unlock()
	log.Infof("New gNMI session registered for sender %q", ip)

	c := NewClient(pr.Addr, mr)
	defer c.Close()
	defer func() {
		srv.lock.Lock()
		defer srv.lock.Unlock()
		delete(srv.gnmiMetricMap, ip)
		log.Infof("gNMI session terminated for sender %q", ip)
	}()
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

		if (log.V(4)) {
			log.V(4).Infof("Received SubscribeResponse: %s", prototext.Format(subscribeResponse))
		}

		notif := subscribeResponse.GetUpdate()
		WalkNotification(notif, func(fqn string, _ *time.Time, json string) {
			// TODO: Verify that the timestamp is not too far off
			if err := c.mr.Update(fqn, json); err != nil {
				log.Warningf("Failed to parse metric update: %v", err)
			}
		}, nil)
	}
}

func (c *Client) Close() {
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	paramMap := make(map[string]string)
	target := params.Get("target")
	paramMap["target"] = params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter missing or empty", http.StatusBadRequest)
		return
	}

	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Whether or not the probe succeeded",
	})

	srv.lock.RLock()
	mr, ok := srv.gnmiMetricMap[target]
	srv.lock.RUnlock()

	ireg := prometheus.NewPedanticRegistry()
	ireg.MustRegister(probeSuccessGauge)

	regs := prometheus.Gatherers{ireg}
	if ok {
		probeSuccessGauge.Set(1)
		log.V(1).Infof("Probe of %q succeeded", target)
		// Assuming the Prometheus Registry object is multi-thread safe this should
		// be fine without locking
		regs = append(regs, mr.PrometheusRegistry())
	} else {
		log.Infof("Probe of %q failed, no gNMI data available at this time", target)
	}

	h := promhttp.HandlerFor(regs, promhttp.HandlerOpts{Registry: ireg})
	h.ServeHTTP(w, r)
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

	http.Handle("/probe", s)
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *metricPort), nil))
	}()

	log.V(1).Infof("Starting RPC server on address: %s", s.Address())
	s.Serve()
	log.Flush()
}
