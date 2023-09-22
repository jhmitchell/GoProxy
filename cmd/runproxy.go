package main

import (
	"net/http"
	"flag"
	"os"
	"fmt"

	"github.com/jhmitchell/GoProxy/rproxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// parseArgs parses the command-line arguments for proxy parameters.
func parseArgs() (string, int, int, *zap.Logger) {
	var rhost, logfile string
	var rport, lport int
	
	log, _ := zap.NewProduction()

	// Set flags. Note: I want to try colorizing this
	flag.StringVar(&rhost, "rhost", "", "The host to be proxied")
	flag.IntVar(&rport, "rport", 80, "The port of the host to be proxied")
	flag.IntVar(&lport, "lport", 8080, "The port the proxy will listen on")
	flag.StringVar(&logfile, "logging", "", "Logfile name")

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

	return rhost, rport, lport, log
}

func main() {
	rhost, rport, lport, log := parseArgs()

	p, err := rproxy.NewProxy(rhost, rport, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create proxy: %v\n", err)
		os.Exit(1)
	}

	log.Info("Reverse Proxy running")

	// Register the reverse proxy as the handler for all incoming requests
	http.HandleFunc("/", rproxy.ProxyRequestHandler(p.ReverseProxy))
	
	// Start the http server
	// Note: More control over the server's behavior is available by creating
	// a custom Server
	lportString := fmt.Sprintf(":%d", lport)
	if err := http.ListenAndServe(lportString, nil); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
