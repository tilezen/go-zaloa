package main

import (
	"log"
	"os"

	"github.com/akrylysov/algnhsa"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"

	"github.com/tilezen/go-zaloa/pkg/fetcher"
	"github.com/tilezen/go-zaloa/pkg/service"
)

func main() {
	fetchMethod, _ := os.LookupEnv("ZALOA_FETCH_METHOD")
	httpPrefix, _ := os.LookupEnv("ZALOA_HTTP_PREFIX")
	s3Bucket, _ := os.LookupEnv("ZALOA_S3_BUCKET")
	awsRegion, _ := os.LookupEnv("ZALOA_AWS_REGION")
	iamRole, _ := os.LookupEnv("ZALOA_AWS_ROLE")
	_, requesterPays := os.LookupEnv("ZALOA_S3_REQUESTER_PAYS")

	var tileFetcher fetcher.TileFetcher
	switch fetchMethod {
	case "http":
		if httpPrefix == "" {
			log.Fatalf("http-prefix must be set when using the http fetch method")
		}

		log.Printf("Using '%s' as http prefix", httpPrefix)

		tileFetcher = fetcher.NewHTTPTileFetcher(httpPrefix)
	case "s3":
		if s3Bucket == "" {
			log.Fatalf("s3-bucket must be set when using the s3 fetch method")
		}

		if awsRegion == "" {
			log.Fatalf("region must be set when using the s3 fetch method")
		}

		var awsSession *session.Session
		var err error
		if iamRole == "" {
			awsSession, err = session.NewSessionWithOptions(session.Options{
				Config: aws.Config{Region: aws.String(awsRegion)},
			})
		} else {
			log.Printf("Configured to use AWS role %s", iamRole)
			awsSession, err = session.NewSessionWithOptions(session.Options{
				Config: aws.Config{
					Credentials: stscreds.NewCredentials(session.Must(session.NewSession()), iamRole),
					Region:      aws.String(awsRegion),
				},
				SharedConfigState: session.SharedConfigEnable,
			})
		}
		if err != nil {
			log.Fatalf("Unable to set up AWS session: %s", err.Error())
		}

		s3Client := s3.New(awsSession)

		tileFetcher = fetcher.NewS3TileFetcher(s3Client, s3Bucket, requesterPays)
	default:
		log.Fatalf("No fetch-method specified")
	}

	zaloaService := service.NewZaloaService(tileFetcher)

	r := mux.NewRouter()

	r.HandleFunc("/tilezen/terrain/{version:v[0-9]+}/{tilesize:[0-9]+}/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", zaloaService.GetTileHandler())
	r.HandleFunc("/tilezen/terrain/{version:v[0-9]+}/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", zaloaService.GetTileHandler())

	algnhsa.ListenAndServe(r, &algnhsa.Options{BinaryContentTypes: []string{"*/*"}})
}
