package p2p

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/addressbook"
)

var upgrader = websocket.Upgrader{}

type serverConfig struct {
	pool *peerPool
}

type server struct {
	logger   log.Logger
	pool     *peerPool
	nodeInfo *NodeInfo

	// created during start
	httpServer *http.Server
}

func newServer(cfg *serverConfig) *server {
	return &server{
		pool: cfg.pool,
	}
}

func (s *server) start(logger log.Logger, nodeInfo *NodeInfo) error {
	s.logger = logger
	s.nodeInfo = nodeInfo
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.nodeInfo.Port),
		ReadHeaderTimeout: 1 * time.Second,
	}
	http.HandleFunc("/rpc/", s.handshake)
	return s.httpServer.ListenAndServe()
}

func (s *server) stop() error {
	ctx := context.Background()
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *server) handshake(w http.ResponseWriter, r *http.Request) {
	// parse request query and validate
	query := r.URL.Query()
	chainID := query.Get("chainID")
	if chainID == "" || chainID != s.nodeInfo.ChainID {
		s.logger.Debug("Invalid chainID", r.RemoteAddr)
		return
	}
	networkVersion := query.Get("networkVersion")
	if networkVersion == "" || !compareVersion(s.nodeInfo.NetworkVersion, networkVersion) {
		s.logger.Debug("Invalid network version", r.RemoteAddr)
		return
	}
	portStr := query.Get("port")
	if portStr == "" {
		s.logger.Debug("Invalid port", r.RemoteAddr)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		s.logger.Debug("Invalid port", r.RemoteAddr)
		return
	}
	nonce := query.Get("nonce")
	if nonce == "" || nonce == s.nodeInfo.Nonce {
		s.logger.Debug("Reject connection from itself")
		return
	}
	// Split IP and port
	parsedURL := strings.Split(r.RemoteAddr, ":")
	addr := addressbook.NewAddress(parsedURL[0], port)
	if err != nil {
		s.logger.Errorf("Fail to create address with %v", err)
		return
	}
	// if s.pool.connected(addr) {
	// 	s.logger.Errorf("Already has connection with %s", addr)
	// 	return
	// }
	// check extra query
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Upgrade error", err)
	}
	s.pool.addConnection(conn, addr)
}

func validateVersion(version string) bool {
	rule := regexp.MustCompile(`\d+\.\d+`)
	return rule.MatchString(version)
}

func compareVersion(selfVersion, version string) bool {
	if valid := validateVersion(version); !valid {
		return false
	}
	selfSplitVer := strings.Split(selfVersion, ".")
	splitVer := strings.Split(version, ".")

	return selfSplitVer[0] == splitVer[0]
}
