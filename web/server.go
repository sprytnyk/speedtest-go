package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/librespeed/speedtest-go/config"
)

// newServer returns an HTTP server for h. Only reading the request headers is
// time-limited, since the speed test itself needs long-running bodies.
func newServer(conf *config.Config, h http.Handler) *http.Server {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if conf.EnableTLS && !conf.EnableHTTP2 {
		// a non-nil empty map disables HTTP/2 over TLS
		srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
	return srv
}

// listenAndServe listens on the configured address, using TLS if enabled.
func listenAndServe(conf *config.Config, h http.Handler) error {
	addr := net.JoinHostPort(conf.BindAddress, conf.Port)
	log.Infof("Starting backend server on %s", addr)

	srv := newServer(conf, h)
	srv.Addr = addr

	if conf.EnableTLS {
		log.Info("Use TLS connection.")
		return srv.ListenAndServeTLS(conf.TLSCertFile, conf.TLSKeyFile)
	}
	if conf.EnableHTTP2 {
		log.Errorf("TLS is mandatory for HTTP/2. Ignore settings that enable HTTP/2.")
	}
	return srv.ListenAndServe()
}
