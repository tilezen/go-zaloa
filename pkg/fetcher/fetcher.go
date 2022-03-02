package fetcher

import (
	"context"
	"image"
	"net/http"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"github.com/tilezen/go-zaloa/pkg/common"
)

type ImageInput struct {
	Image image.Image
	Spec  ImageSpec
}

type FetchResponse struct {
	Data []byte
	Tile common.Tile
	Spec ImageSpec
}

type ImageSpec struct {
	Location image.Point
	Crop     image.Rectangle
}

type TileFetcher interface {
	GetTile(ctx context.Context, t common.Tile, kind common.TileKind, version common.TileVersion) (*FetchResponse, error)
}

func NewHTTPTileFetcher(baseURL string) TileFetcher {
	return &httpFetcher{
		baseURL: baseURL,
		client:  http.DefaultClient,
	}
}

func NewS3TileFetcher(s3 s3iface.S3API, bucket string, requesterPays bool) TileFetcher {
	return &s3tileFetcher{
		s3Bucket:      bucket,
		requesterPays: requesterPays,
		s3:            s3,
	}
}
