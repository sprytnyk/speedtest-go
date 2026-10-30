package web

import (
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librespeed/speedtest-go/config"
)

// newServer returns an HTTP server for h. Only reading the request headers is
// time-limited, since the speed test itself needs long-running bodies.
func newServer(h http.Handler) *http.Server {
	// HTTP/2 would put the download streams and the pings on one TCP connection
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		Protocols:         &protocols,
	}
}

// listenAndServe listens on the configured address, using TLS if enabled.
func listenAndServe(conf *config.Config, h http.Handler) error {
	addr := net.JoinHostPort(conf.BindAddress, conf.Port)
	log.Infof("Starting backend server on %s", addr)

	srv := newServer(h)
	srv.Addr = addr

	if conf.EnableTLS {
		log.Info("Use TLS connection.")
		return srv.ListenAndServeTLS(conf.TLSCertFile, conf.TLSKeyFile)
	}
	return srv.ListenAndServe()
}
