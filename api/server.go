package api

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"

	"github.com/datatogether/api/apiutil"
	// "github.com/qri-io/qri/api/handlers"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/logging"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// Server wraps a qri p2p node, providing traditional access via http
// Create one with New, start it up with Serve
type Server struct {
	// configuration options
	cfg     *Config
	log     logging.Logger
	qriNode *p2p.QriNode
}

// New creates a new qri server with optional configuration
// calling New() with no options will return the default configuration
// as specified in DefaultConfig
func New(r repo.Repo, options ...func(*Config)) (s *Server, err error) {
	cfg := DefaultConfig()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("server configuration error: %s", err.Error())
	}

	s = &Server{
		cfg: cfg,
		log: cfg.Logger,
	}

	// allocate a new node
	s.qriNode, err = p2p.NewQriNode(r, func(ncfg *p2p.NodeCfg) {
		ncfg.Logger = s.log
		ncfg.Online = s.cfg.Online
		if cfg.BoostrapAddrs != nil {
			ncfg.QriBootstrapAddrs = cfg.BoostrapAddrs
		}
	})
	if err != nil {
		return s, err
	}

	if s.cfg.Online {
		// s.log.Info("qri profile id:", s.qriNode.Identity.Pretty())
		s.log.Info("p2p addresses:")
		for _, a := range s.qriNode.EncapsulatedAddresses() {
			s.log.Infof("  %s", a.String())
			// s.log.Infln(a.Protocols())
		}
	} else {
		s.log.Info("running qri in offline mode, no peer-2-peer connections")
	}

	bootstrapped := false
	peerBootstrapped := func(peerId string) {
		if cfg.PostP2POnlineHook != nil && !bootstrapped {
			go cfg.PostP2POnlineHook(s.qriNode)
			bootstrapped = true
		}
	}

	err = s.qriNode.StartOnlineServices(peerBootstrapped)
	if err != nil {
		return nil, fmt.Errorf("error starting P2P service: %s", err.Error())
	}
	// p2p.PrintSwarmAddrs(qriNode)
	return s, nil
}

// Serve starts the server. It will block while the server
// is running
func (s *Server) Serve() (err error) {
	server := &http.Server{}
	server.Handler = NewServerRoutes(s)
	s.log.Infof("starting api server on port %s", s.cfg.Port)
	go s.ServeRPC()
	// http.ListenAndServe will not return unless there's an error
	return StartServer(s.cfg, server)
}

// ServeRPC checks for a configured RPC port, and registers a listner if so
func (s *Server) ServeRPC() {
	if s.cfg.RPCPort == "" {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", s.cfg.RPCPort))
	if err != nil {
		s.log.Infof("RPC listen on port %s error: %s", s.cfg.RPCPort, err)
		return
	}

	for _, rcvr := range core.Receivers(s.qriNode) {
		if err := rpc.Register(rcvr); err != nil {
			s.log.Infof("error registering RPC receiver %s: %s", rcvr.CoreRequestsName(), err.Error())
			return
		}
	}

	s.log.Infof("accepting RPC requests on port %s", s.cfg.RPCPort)
	rpc.Accept(listener)
	return
}

// NewServerRoutes returns a Muxer that has all API routes
func NewServerRoutes(s *Server) *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/", WebappHandler)
	m.Handle("/status", s.middleware(apiutil.HealthCheckHandler))
	m.Handle("/ipfs/", s.middleware(s.HandleIPFSPath))

	proh := NewProfileHandlers(s.log, s.qriNode.Repo)
	m.Handle("/profile", s.middleware(proh.ProfileHandler))
	m.Handle("/profile/photo", s.middleware(proh.SetProfilePhotoHandler))
	m.Handle("/profile/poster", s.middleware(proh.SetPosterHandler))

	sh := NewSearchHandlers(s.log, s.qriNode.Repo)
	m.Handle("/search", s.middleware(sh.SearchHandler))

	ph := NewPeerHandlers(s.log, s.qriNode.Repo, s.qriNode)
	m.Handle("/peers", s.middleware(ph.PeersHandler))
	m.Handle("/peers/", s.middleware(ph.PeerHandler))
	m.Handle("/connect/", s.middleware(ph.ConnectToPeerHandler))
	m.Handle("/connections", s.middleware(ph.ConnectionsHandler))
	m.Handle("/peernamespace/", s.middleware(ph.PeerNamespaceHandler))

	dsh := NewDatasetHandlers(s.log, s.qriNode.Repo)
	m.Handle("/datasets", s.middleware(dsh.DatasetsHandler))
	m.Handle("/datasets/", s.middleware(dsh.DatasetHandler))
	m.Handle("/add/", s.middleware(dsh.AddDatasetHandler))
	m.Handle("/init/", s.middleware(dsh.InitDatasetHandler))
	m.Handle("/rename", s.middleware(dsh.RenameDatasetHandler))
	m.Handle("/data/ipfs/", s.middleware(dsh.StructuredDataHandler))
	m.Handle("/download/", s.middleware(dsh.ZipDatasetHandler))

	hh := NewHistoryHandlers(s.log, s.qriNode.Repo)
	m.Handle("/history/", s.middleware(hh.LogHandler))

	qh := NewQueryHandlers(s.log, s.qriNode.Repo)
	m.Handle("/queries", s.middleware(qh.ListHandler))
	m.Handle("/queries/", s.middleware(qh.DatasetQueriesHandler))
	m.Handle("/run", s.middleware(qh.RunHandler))

	return m
}
