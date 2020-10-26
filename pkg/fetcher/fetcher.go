package fetcher

import (
	"context"
	"net/http"

	"github.com/tilezen/go-zaloa/pkg/common"
	"github.com/tilezen/go-zaloa/pkg/service"
)

type TileFetcher interface {
	GetTile(ctx context.Context, t common.Tile, kind common.TileKind) (*service.FetchResponse, error)
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
