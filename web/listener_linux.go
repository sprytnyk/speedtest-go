//go:build linux

package web

import (
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/go-chi/chi/v5"
	"github.com/librespeed/speedtest-go/config"
	log "github.com/sirupsen/logrus"
)

func startListener(conf *config.Config, r *chi.Mux) error {
	// See if systemd socket activation has been used when starting our process
	listeners, err := activation.Listeners()
	if err != nil {
		log.Fatalf("Error whilst checking for systemd socket activation %s", err)
	}

	switch len(listeners) {
	case 0:
		return listenAndServe(conf, r)
	case 1:
		log.Info("Starting backend server on inherited file descriptor via systemd socket activation")
		if conf.BindAddress != "" || conf.Port != "" {
			log.Errorf("Both an address/port (%s:%s) has been specificed in the config AND externally configured socket activation has been detected", conf.BindAddress, conf.Port)
			log.Fatal(`Please deconfigure socket activation (e.g. in systemd unit files), or set both 'bind_address' and 'listen_port' to ''`)
		}
		return newServer(r).Serve(listeners[0])
	default:
		log.Fatalf("Asked to listen on %d sockets via systemd activation.  Sorry we currently only support listening on 1 socket.", len(listeners))
	}
	return nil
}
