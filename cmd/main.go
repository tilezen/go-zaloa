package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/tilezen/go-zaloa/pkg/fetcher"
	"github.com/tilezen/go-zaloa/pkg/service"
)

func main() {
	port := flag.Int("port", 8080, "The port to listen on")
	fetchMethod := flag.String("fetch-method", "", "Method to use when fetching tiles. Use http or s3.")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket to fetch tiles from when using S3 fetch method")
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

		tileFetcher = fetcher.NewS3TileFetcher(*s3Bucket, *requesterPays)
	default:
		log.Fatalf("No fetch-method specified")
	}

	zaloaService := service.NewZaloaService(tileFetcher)

	r := mux.NewRouter()
	r.HandleFunc("/health_check", zaloaService.GetHealthCheckHandler())
	r.HandleFunc("/tilezen/terrain/v1/{tilesize:[0-9]+}/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", zaloaService.GetTileHandler())
	r.HandleFunc("/tilezen/terrain/v1/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", zaloaService.GetTileHandler())

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening to %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Error listening to %s: %+v", addr, err)
	}
}
