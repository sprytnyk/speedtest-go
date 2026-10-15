//go:build !linux

package web

import (
	"github.com/go-chi/chi/v5"
	"github.com/librespeed/speedtest-go/config"
)

func startListener(conf *config.Config, r *chi.Mux) error {
	return listenAndServe(conf, r)
}
