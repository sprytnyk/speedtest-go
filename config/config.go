package config

import (
	"errors"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	BindAddress       string `mapstructure:"bind_address"`
	Port              string `mapstructure:"listen_port"`
	BaseURL           string `mapstructure:"url_base"`
	ProxyProtocolPort string `mapstructure:"proxyprotocol_port"`

	StatsPassword string `mapstructure:"statistics_password"`
	RedactIP      bool   `mapstructure:"redact_ip_addresses"`

	AssetsPath string `mapstructure:"assets_path"`

	IPLookupURL string `mapstructure:"ip_lookup_url"`

	DatabaseType     string `mapstructure:"database_type"`
	DatabaseHostname string `mapstructure:"database_hostname"`
	DatabaseName     string `mapstructure:"database_name"`
	DatabaseUsername string `mapstructure:"database_username"`
	DatabasePassword string `mapstructure:"database_password"`

	DatabaseFile string `mapstructure:"database_file"`
	DatabasePort string `mapstructure:"database_port"`

	EnableHTTP2 bool   `mapstructure:"enable_http2"`
	EnableTLS   bool   `mapstructure:"enable_tls"`
	TLSCertFile string `mapstructure:"tls_cert_file"`
	TLSKeyFile  string `mapstructure:"tls_key_file"`
}

// IPLookupPlaceholder is replaced with the client's IP in ip_lookup_url.
const IPLookupPlaceholder = "{ip}"

var (
	configFile   string
	loadedConfig *Config = nil
)

func init() {
	// every key needs a default, otherwise viper ignores SPEEDTEST_* env vars
	// for keys that are missing from the config file
	viper.SetDefault("bind_address", "")
	viper.SetDefault("listen_port", "8989")
	viper.SetDefault("url_base", "")
	viper.SetDefault("proxyprotocol_port", "0")
	viper.SetDefault("statistics_password", "PASSWORD")
	viper.SetDefault("redact_ip_addresses", false)
	viper.SetDefault("assets_path", "")
	viper.SetDefault("ip_lookup_url", "") // no lookups unless settings.toml or the environment names an API
	viper.SetDefault("database_type", "postgresql")
	viper.SetDefault("database_hostname", "localhost")
	viper.SetDefault("database_name", "speedtest")
	viper.SetDefault("database_username", "postgres")
	viper.SetDefault("database_password", "")
	viper.SetDefault("database_file", "speedtest.db")
	viper.SetDefault("database_port", "")
	viper.SetDefault("enable_tls", false)
	viper.SetDefault("enable_http2", false)
	viper.SetDefault("tls_cert_file", "")
	viper.SetDefault("tls_key_file", "")

	viper.SetConfigName("settings")
	viper.AddConfigPath(".")
}

func Load(configPath string) Config {
	var conf Config

	configFile = configPath
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("speedtest")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if configPath != "" || !errors.As(err, &notFound) {
			log.Fatalf("Error reading config: %s", err)
		}
		log.Warnf("No settings.toml found, using defaults and environment variables")
	}

	if err := viper.Unmarshal(&conf); err != nil {
		log.Fatalf("Error parsing config: %s", err)
	}

	// ID obfuscation was removed: test IDs are random, so it only hid part of
	// the test's time, which the JSON result shows anyway
	if viper.GetBool("enable_id_obfuscation") {
		log.Warnf("enable_id_obfuscation is no longer supported and is ignored; test IDs are not obfuscated")
	}

	if conf.IPLookupURL != "" {
		if !strings.Contains(conf.IPLookupURL, IPLookupPlaceholder) {
			log.Fatalf("ip_lookup_url must contain the %s placeholder: %s", IPLookupPlaceholder, conf.IPLookupURL)
		}
		if u, err := url.Parse(conf.IPLookupURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			log.Fatalf("ip_lookup_url must be an absolute http(s) URL: %s", conf.IPLookupURL)
		}
	}

	loadedConfig = &conf

	return conf
}

func LoadedConfig() *Config {
	if loadedConfig == nil {
		Load(configFile)
	}
	return loadedConfig
}
