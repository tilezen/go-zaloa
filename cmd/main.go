package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/tilezen/go-zaloa/pkg/fetcher"
	"github.com/tilezen/go-zaloa/pkg/service"
)

const (
	// The time to wait after responding /ready with non-200 before starting to shut down the HTTP server
	gracefulShutdownSleep = 20 * time.Second
	// The time to wait for the in-flight HTTP requests to complete before exiting
	gracefulShutdownTimeout = 5 * time.Second
)

func main() {
	port := flag.Int("port", 8080, "The port to listen on")
	fetchMethod := flag.String("fetch-method", "", "Method to use when fetching tiles. Use http or s3.")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket to fetch tiles from when using S3 fetch method")
	iamRole := flag.String("iam-role", "", "IAM role to assume when setting up connection to S3")
	awsRegion := flag.String("region", "", "Region to use when setting up connection to S3")
	requesterPays := flag.Bool("requester-pays", false, "Set the requester pays flag when using the S3 fetch method")
	httpPrefix := flag.String("http-prefix", "", "HTTP prefix when fetching tiles using HTTP fetch method")
	flag.Parse()

	var tileFetcher fetcher.TileFetcher
	switch *fetchMethod {
	case "http":
		if *httpPrefix == "" {
			log.Fatalf("http-prefix must be set when using the http fetch method")
		}

		tileFetcher = fetcher.NewHTTPTileFetcher(*httpPrefix)
	case "s3":
		if *s3Bucket == "" {
			log.Fatalf("s3-bucket must be set when using the s3 fetch method")
		}

		if *awsRegion == "" {
			log.Fatalf("region must be set when using the s3 fetch method")
		}

		var awsSession *session.Session
		var err error
		if *iamRole == "" {
			awsSession, err = session.NewSessionWithOptions(session.Options{
				Config: aws.Config{Region: awsRegion},
			})
		} else {
			log.Printf("Configured to use AWS role %s", *iamRole)
			awsSession, err = session.NewSessionWithOptions(session.Options{
				Config: aws.Config{
					Credentials: stscreds.NewCredentials(session.Must(session.NewSession()), *iamRole),
					Region:      awsRegion,
				},
				SharedConfigState: session.SharedConfigEnable,
			})
		}
		if err != nil {
			log.Fatalf("Unable to set up AWS session: %s", err.Error())
		}

		s3Client := s3.New(awsSession)

		tileFetcher = fetcher.NewS3TileFetcher(s3Client, *s3Bucket, *requesterPays)
	default:
		log.Fatalf("No fetch-method specified")
	}

	zaloaService := service.NewZaloaService(tileFetcher)

	r := mux.NewRouter()

	// Readiness probe for graceful shutdown support
	readinessResponseCode := uint32(http.StatusOK)
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(atomic.LoadUint32(&readinessResponseCode)))
	})

	r.HandleFunc("/live", zaloaService.GetHealthCheckHandler())

	r.HandleFunc("/tilezen/terrain/{version:v[0-9]+}/{tilesize:[0-9]+}/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", zaloaService.GetTileHandler())
	r.HandleFunc("/tilezen/terrain/{version:v[0-9]+}/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", zaloaService.GetTileHandler())

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening to %s", addr)

	// Support for upgrading an http/1.1 connection to http/2
	// See https://github.com/thrawn01/h2c-golang-example
	http2Server := &http2.Server{}
	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(r, http2Server),
	}

	// Code to handle shutdown gracefully
	shutdownChan := make(chan struct{})
	go func() {
		defer close(shutdownChan)

		// Wait for SIGTERM to come in
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		<-signals

		log.Printf("SIGTERM received. Starting graceful shutdown.")

		// Start failing readiness probes
		atomic.StoreUint32(&readinessResponseCode, http.StatusInternalServerError)
		// Wait for upstream clients
		time.Sleep(gracefulShutdownSleep)
		// Begin shutdown of in-flight requests
		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error waiting for server shutdown: %+v", err)
		}
		shutdownCtxCancel()
	}()

	log.Printf("Service started")
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Couldn't start HTTP server: %+v", err)
	}
	<-shutdownChan
}
