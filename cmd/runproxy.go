package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/jhmitchell/GoProxy/rproxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// parseArgs parses the command-line arguments for proxy parameters.
func parseArgs() (string, int, int, string, string, string, *zap.Logger) {
	var rhost, logfile, mode, certFile, keyFile string
	var rport, lport int

	log, _ := zap.NewProduction()

	// Set flags. Note: I want to try colorizing this
	flag.StringVar(&mode, "mode", "http", "The mode to run the proxy server in: http or https")
	flag.StringVar(&rhost, "rhost", "", "The host to be proxied")
	flag.IntVar(&rport, "rport", 80, "The port of the host to be proxied")
	flag.IntVar(&lport, "lport", 8080, "The port the proxy will listen on")
	flag.StringVar(&logfile, "logging", "", "Logfile name")
	flag.StringVar(&certFile, "cert", "path/to/cert.pem", "Path to SSL certificate file")
	flag.StringVar(&keyFile, "key", "path/to/key.pem", "Path to SSL private key file")

	// Set custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if rhost == "" {
		// Require rhost, otherwise exit
		fmt.Fprintf(os.Stderr, "Missing required argument --rhost\n")
		flag.Usage()
		os.Exit(1)
	}

	if logfile != "" {
		// Create or open the log file
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		log = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(file),
			zap.InfoLevel,
		))
	}

	return mode, lport, rport, rhost, certFile, keyFile, log
}

func main() {
	mode, lport, rport, rhost, certFile, keyFile, log := parseArgs()

	p, err := rproxy.NewProxy(rhost, rport, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create proxy: %v\n", err)
		os.Exit(1)
	}

	log.Info("Reverse Proxy running")

	http.Handle("/", rproxy.RateLimiterMiddleware(p))

	lportString := fmt.Sprintf(":%d", lport)
	if mode == "https" {
		log.Info("Starting server in HTTPS mode", zap.String("port", lportString))
		if err := http.ListenAndServeTLS(lportString, certFile, keyFile, nil); err != nil {
			log.Fatal("Failed to start HTTPS server", zap.Error(err))
		}
	} else {
		log.Info("Starting server in HTTP mode", zap.String("port", lportString))
		if err := http.ListenAndServe(lportString, nil); err != nil {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}
}
