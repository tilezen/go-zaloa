package fetcher

import (
	"context"
	"image"
	"net/http"

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
	GetTile(ctx context.Context, t common.Tile, kind common.TileKind) (*FetchResponse, error)
}

func NewHTTPTileFetcher(baseURL string) TileFetcher {
	return &httpFetcher{
		baseURL: baseURL,
		client:  http.DefaultClient,
	}
}

func NewS3TileFetcher(s3Bucket string, requesterPays bool) TileFetcher {
	return &s3tileFetcher{
		s3Bucket:      s3Bucket,
		requesterPays: requesterPays,
	}
}
