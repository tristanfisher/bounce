package main

import (
	"bounce/config"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	stdLibLog "log" // needed  until we have our app logger configured
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

func main() {
	configFilePath := os.Getenv("CONFIG_FILE")
	configFilePathFlag := flag.String("config_file", "", "path to configuration file")
	flag.Parse()

	// command line arg takes precedence over env
	if configFilePathFlag != nil && len(*configFilePathFlag) > 0 {
		configFilePath = *configFilePathFlag
	}

	conf, err := config.New(configFilePath)
	if err != nil {
		stdLibLog.Fatal(err)
	}

	// logger
	zerolog.TimeFieldFormat = time.RFC3339
	lvl, err := zerolog.ParseLevel(conf.LogLevel)
	if err != nil {
		stdLibLog.Fatalf("could not configure logger.  reason: %s", err)
	}

	zerolog.SetGlobalLevel(lvl)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	baseContext := context.Background()
	httpServerContext, httpServerCancel := context.WithCancel(baseContext)
	httpsServerContext, httpsServerCancel := context.WithCancel(baseContext)

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	hs := HandlerSupport{Log: logger}

	// run the servers
	if len(conf.HttpServerAddr) > 0 {
		// not using http2 h2c (http/2 without TLS) on purpose
		httpServer := &http.Server{
			Addr:         conf.HttpServerAddr,
			Handler:      Mux(hs),
			ReadTimeout:  conf.HttpReadTimeout,
			WriteTimeout: conf.HttpWriteTimeout,
			IdleTimeout:  conf.HttpIdleTimeout,
			//ErrorLog:     stdLibLog.New(nil, "", 0),
			BaseContext: func(listener net.Listener) context.Context { return httpServerContext },
		}
		httpServer.SetKeepAlivesEnabled(false)
		go func() {
			err := httpServer.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error().Err(err).Msg("error listening or serving")
			}
		}()
	}

	if len(conf.HttpsServerAddr) > 0 {
		keyPair, err := tls.LoadX509KeyPair(conf.HttpsCertificatePath, conf.HttpsKeyPath)
		if err != nil {
			logger.Fatal().Err(err).
				Bool("missing_cert", len(conf.HttpsCertificatePath) < 1).
				Bool("missing_key", len(conf.HttpsKeyPath) < 1).
				Msg("error loading keypair")
		}

		tlsServer := http.Server{
			Addr:    conf.HttpsServerAddr,
			Handler: Mux(hs),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{keyPair},
				ServerName:   conf.HttpsServerName,
				CipherSuites: []uint16{
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_AES_128_GCM_SHA256, // disable to force to more secure 384
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_CHACHA20_POLY1305_SHA256, // not FIPS
				},
				SessionTicketsDisabled: false,
				ClientSessionCache:     nil,
				MinVersion:             tls.VersionTLS12,
			},
			ReadTimeout:  conf.HttpReadTimeout,
			WriteTimeout: conf.HttpWriteTimeout,
			IdleTimeout:  conf.HttpIdleTimeout,
			BaseContext:  func(listener net.Listener) context.Context { return httpsServerContext },
		}

		go func() {
			// empty certFile, keyFile tells internals to use the tls.Certificate we previously loaded
			err := tlsServer.ListenAndServeTLS("", "")
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error().Err(err).Msg("error listening or serving")
			}
		}()
	}

	<-shutdownSignal
	logger.Debug().Msg("shutdown signal received")
	// shut the incoming stream of requests
	httpServerCancel()
	httpsServerCancel()

	// blocking timer to keep the process alive to allow anything in the background to finish
	shutdownTimer := time.NewTicker(conf.ShutdownDeadline)
	<-shutdownTimer.C

}
