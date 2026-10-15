package main

import (
	"flag"
	"fmt"
	_ "time/tzdata"

	"github.com/librespeed/speedtest-go/config"
	"github.com/librespeed/speedtest-go/database"
	"github.com/librespeed/speedtest-go/results"
	"github.com/librespeed/speedtest-go/web"

	_ "github.com/breml/rootcerts"
	log "github.com/sirupsen/logrus"
)

var (
	optConfig  = flag.String("c", "", "config file to be used, defaults to settings.toml in the same directory")
	optVersion = flag.Bool("version", false, "print the version and exit")

	// version is set at build time with -ldflags "-X main.version=..."
	version = "dev"
)

func main() {
	flag.Parse()
	if *optVersion {
		fmt.Println("speedtest-go", version)
		return
	}
	log.Infof("speedtest-go %s", version)
	conf := config.Load(*optConfig)
	results.Initialize(&conf)
	database.SetDBInfo(&conf)
	log.Fatal(web.ListenAndServe(&conf))
}
