package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"

	"github.com/tilezen/go-zaloa/pkg/common"
	"github.com/tilezen/go-zaloa/pkg/fetcher"
)

const (
	maxZoom = 15
)

type FetchResponse struct {
	Data []byte
	Tile common.Tile
	Spec ImageSpec
}

type ZaloaService interface {
	GetHealthCheckHandler() func(http.ResponseWriter, *http.Request)
	GetTileHandler() func(http.ResponseWriter, *http.Request)
}

type zaloaService struct {
	fetcher fetcher.TileFetcher
}

func (z zaloaService) GetHealthCheckHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		_, err := z.fetcher.GetTile(ctx, common.Tile{Z: 0, X: 0, Y: 0}, common.TileType_TERRARIUM)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			log.Printf("Couldn't get healthcheck Tile: %+v", err)
		}

		writer.WriteHeader(http.StatusOK)
		return
	}
}

type instruction struct {
	// tileToFetch is the Tile to fetch
	tileToFetch common.Tile
	spec        ImageSpec
}

func (z zaloaService) GetTileHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		vars := mux.Vars(request)
		var err error

		tileSize := uint64(256)
		if vars["tilesize"] != "" {
			tileSize, err = strconv.ParseUint(vars["tilesize"], 10, 32)
			if err != nil {
				writer.WriteHeader(http.StatusNotFound)
				_, _ = writer.Write([]byte("Invalid tilesize"))
				return
			}
		}

		var tileset common.TileKind
		switch vars["tileset"] {
		case "terrarium":
			tileset = common.TileType_TERRARIUM
		case "normal":
			tileset = common.TileType_NORMAL
		default:
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("Invalid tileset"))
			return
		}

		parsedTile, err := common.ParseTile(vars["z"], vars["x"], vars["y"])
		if err != nil {
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("Invalid Tile coordinate"))
			return
		}

		if parsedTile.Z > maxZoom {
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("Invalid zoom"))
			return
		}

		if parsedTile.Z == maxZoom && tileSize != 260 {
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("Invalid zoom"))
			return
		}

		// Build parsedTile coordinates to request
		log.Printf("Requested Tile: %s", *parsedTile)
		var imageInstructions []instruction
		switch tileSize {
		case 256:
			imageInstructions = generate256Instructions(*parsedTile)
		case 260:
			imageInstructions = generate260Instructions(*parsedTile)
		case 512:
			imageInstructions = generate512Instructions(*parsedTile)
		case 516:
			imageInstructions = generate516Instructions(*parsedTile)
		default:
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte("Invalid tilesize"))
			return
		}

		tileData, err := z.ProcessTile(ctx, int(tileSize), imageInstructions, tileset)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Error fetching parsedTile"))
			return
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(tileData)
		return
	}
}

func (z zaloaService) ProcessTile(ctx context.Context, tileSize int, instructions []instruction, tileset common.TileKind) ([]byte, error) {
	// Fetch the tiles required to process the requested Tile
	imageInputs, err := z.FetchTiles(ctx, tileset, instructions)
	if err != nil {
		return nil, fmt.Errorf("error fetching tiles: %w", err)
	}

	// Reduce the images into a single output Tile
	dst := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
	for _, input := range imageInputs {
		log.Printf("take crop %s and put at %s", input.spec.crop, input.spec.location)
		draw.Draw(
			dst,
			image.Rect(input.spec.location.X, input.spec.location.Y, input.spec.location.X+256, input.spec.location.Y+256),
			input.image,
			input.spec.crop.Min,
			draw.Src,
		)
	}
	b := &bytes.Buffer{}
	err = png.Encode(b, dst)
	if err != nil {
		return nil, fmt.Errorf("couldn't encode result image: %w", err)
	}

	return b.Bytes(), nil
}

type ImageSpec struct {
	location image.Point
	crop     image.Rectangle
}

type ImageInput struct {
	image image.Image
	spec  ImageSpec
}

func (z zaloaService) FetchTiles(ctx context.Context, tileset common.TileKind, instructions []instruction) ([]ImageInput, error) {
	errs, ctx := errgroup.WithContext(ctx)
	fetchResults := make(chan *FetchResponse, len(instructions))

	for _, inst := range instructions {
		// https://golang.org/doc/faq#closures_and_goroutines
		inst := inst

		errs.Go(func() error {
			resp, err := z.fetcher.GetTile(ctx, inst.tileToFetch, tileset)
			if err != nil {
				return fmt.Errorf("couldn't fetch Tile %s: %w", inst.tileToFetch, err)
			}

			resp.Spec = inst.spec
			fetchResults <- resp
			return nil
		})
	}

	err := errs.Wait()
	if err != nil {
		return nil, fmt.Errorf("error while fetching images: %w", err)
	}
	close(fetchResults)

	inputResults := make([]ImageInput, 0, len(instructions))
	for result := range fetchResults {
		decodedImage, _, err := image.Decode(bytes.NewBuffer(result.Data))
		if err != nil {
			return nil, fmt.Errorf("couldn't decode image data: %w", err)
		}

		inputResults = append(inputResults, ImageInput{
			image: decodedImage,
			spec:  result.Spec,
		})
	}

	return inputResults, nil
}

func generate260Instructions(t common.Tile) []instruction {
	xyMax := uint(math.Pow(2, float64(t.Z))) - 1

	locations := []image.Point{
		{0, 0}, {2, 0}, {258, 0},
		{0, 2}, {2, 2}, {258, 2},
		{0, 258}, {2, 258}, {258, 258},
	}

	tiles := make([]common.Tile, 9)
	// Top row of tiles
	topY := t.Y - 1
	if t.Y == 0 {
		topY = 0
	}
	if t.X == 0 {
		tiles[0] = common.Tile{Z: t.Z, X: xyMax, Y: topY}
	} else {
		tiles[0] = common.Tile{Z: t.Z, X: t.X - 1, Y: topY}
	}
	tiles[1] = common.Tile{Z: t.Z, X: t.X, Y: topY}
	if t.X == xyMax {
		tiles[2] = common.Tile{Z: t.Z, X: 0, Y: topY}
	} else {
		tiles[2] = common.Tile{Z: t.Z, X: t.X + 1, Y: topY}
	}

	// Middle row of tiles
	if t.X == 0 {
		tiles[3] = common.Tile{Z: t.Z, X: xyMax, Y: t.Y}
	} else {
		tiles[3] = common.Tile{Z: t.Z, X: t.X - 1, Y: t.Y}
	}
	tiles[4] = common.Tile{Z: t.Z, X: t.X, Y: t.Y}
	if t.X == xyMax {
		tiles[5] = common.Tile{Z: t.Z, X: 0, Y: t.Y}
	} else {
		tiles[5] = common.Tile{Z: t.Z, X: t.X + 1, Y: t.Y}
	}

	// Bottom row of tiles
	botY := t.Y + 1
	if t.Y == xyMax {
		botY = xyMax
	}
	if t.X == 0 {
		tiles[6] = common.Tile{Z: t.Z, X: xyMax, Y: botY}
	} else {
		tiles[6] = common.Tile{Z: t.Z, X: t.X - 1, Y: botY}
	}
	tiles[7] = common.Tile{Z: t.Z, X: t.X, Y: botY}
	if t.X == xyMax {
		tiles[8] = common.Tile{Z: t.Z, X: 0, Y: botY}
	} else {
		tiles[8] = common.Tile{Z: t.Z, X: t.X + 1, Y: botY}
	}

	croppings := make([]image.Rectangle, 9)
	if t.Y == 0 {
		croppings[0] = image.Rect(254, 0, 256, 2)
		croppings[1] = image.Rect(0, 0, 256, 2)
		croppings[2] = image.Rect(0, 0, 2, 2)
	} else {
		croppings[0] = image.Rect(254, 254, 256, 256)
		croppings[1] = image.Rect(0, 254, 256, 256)
		croppings[2] = image.Rect(0, 254, 2, 256)
	}
	croppings[3] = image.Rect(254, 0, 256, 256)
	croppings[4] = image.Rect(0, 0, 256, 256)
	croppings[5] = image.Rect(0, 0, 2, 256)
	if t.Y == xyMax {
		croppings[6] = image.Rect(254, 254, 256, 256)
		croppings[7] = image.Rect(0, 254, 256, 256)
		croppings[8] = image.Rect(0, 254, 2, 256)
	} else {
		croppings[6] = image.Rect(254, 0, 256, 2)
		croppings[7] = image.Rect(0, 0, 256, 2)
		croppings[8] = image.Rect(0, 0, 2, 2)
	}

	instructions := make([]instruction, 9)
	for i := 0; i < 9; i++ {
		instructions[i] = instruction{
			tileToFetch: tiles[i],
			spec:        ImageSpec{location: locations[i], crop: croppings[i]},
		}
	}

	return instructions
}

func generate516Instructions(t common.Tile) []instruction {
	// pre-bump the coordinates to the next highest zoom
	z := t.Z + 1
	x := t.X * 2
	y := t.Y * 2

	xyMax := uint(math.Pow(2, float64(z))) - 1

	// Tiles to fetch
	tiles := make([]common.Tile, 0, 16)
	for yI := y - 1; yI < y+3; yI++ {
		var yVal uint

		switch {
		case yI < 0:
			yVal = 0
		case yI > xyMax:
			yVal = xyMax
		default:
			yVal = yI
		}

		for xI := x - 1; xI < x+3; xI++ {
			var xVal uint

			switch {
			case xI < 0:
				xVal = xyMax
			case xI > xyMax:
				xVal = 0
			default:
				xVal = xI
			}

			t := common.Tile{Z: z, X: xVal, Y: yVal}
			log.Printf("%s", t)
			tiles = append(tiles, t)
		}
	}

	// Tile placements
	locations := []image.Point{
		image.Pt(0, 0), image.Pt(2, 0), image.Pt(258, 0), image.Pt(514, 0),
		image.Pt(0, 2), image.Pt(2, 2), image.Pt(258, 2), image.Pt(514, 2),
		image.Pt(0, 258), image.Pt(2, 258), image.Pt(258, 258), image.Pt(514, 258),
		image.Pt(0, 514), image.Pt(2, 514), image.Pt(258, 514), image.Pt(514, 514),
	}

	// Croppings
	croppings := make([]image.Rectangle, 16)
	if y == 0 {
		croppings[0] = image.Rect(254, 0, 256, 2)
		croppings[1] = image.Rect(0, 0, 256, 2)
		croppings[2] = image.Rect(0, 0, 256, 2)
		croppings[3] = image.Rect(0, 0, 2, 2)
	} else {
		croppings[0] = image.Rect(254, 254, 256, 256)
		croppings[1] = image.Rect(0, 254, 256, 256)
		croppings[2] = image.Rect(0, 254, 256, 256)
		croppings[3] = image.Rect(0, 254, 2, 256)
	}

	croppings[4] = image.Rect(254, 0, 256, 256)
	croppings[5] = image.Rect(0, 0, 256, 256)
	croppings[6] = image.Rect(0, 0, 256, 256)
	croppings[7] = image.Rect(0, 0, 2, 256)

	croppings[8] = image.Rect(254, 0, 256, 256)
	croppings[9] = image.Rect(0, 0, 256, 256)
	croppings[10] = image.Rect(0, 0, 256, 256)
	croppings[11] = image.Rect(0, 0, 2, 256)

	if y+1 == xyMax {
		croppings[12] = image.Rect(254, 254, 256, 256)
		croppings[13] = image.Rect(0, 254, 256, 256)
		croppings[14] = image.Rect(0, 254, 256, 256)
		croppings[15] = image.Rect(0, 254, 2, 256)
	} else {
		croppings[12] = image.Rect(254, 0, 256, 2)
		croppings[13] = image.Rect(0, 0, 256, 2)
		croppings[14] = image.Rect(0, 0, 256, 2)
		croppings[15] = image.Rect(0, 0, 2, 2)
	}

	instructions := make([]instruction, 16)
	for i := 0; i < 16; i++ {
		instructions[i] = instruction{
			tileToFetch: tiles[i],
			spec:        ImageSpec{location: locations[i], crop: croppings[i]},
		}
	}

	return instructions
}

func generate512Instructions(t common.Tile) []instruction {
	zPlus1 := t.Z + 1
	doubleX := t.X * 2
	doubleY := t.Y * 2

	return []instruction{
		{
			tileToFetch: common.Tile{Z: zPlus1, X: doubleX, Y: doubleY},
			spec: ImageSpec{
				location: image.Point{X: 0, Y: 0},
				crop:     image.Rect(0, 0, 256, 256),
			},
		},
		{
			tileToFetch: common.Tile{Z: zPlus1, X: doubleX + 1, Y: doubleY},
			spec: ImageSpec{
				location: image.Point{X: 256, Y: 0},
				crop:     image.Rect(0, 0, 256, 256),
			},
		},
		{
			tileToFetch: common.Tile{Z: zPlus1, X: doubleX, Y: doubleY + 1},
			spec: ImageSpec{
				location: image.Point{X: 0, Y: 256},
				crop:     image.Rect(0, 0, 256, 256),
			},
		},
		{
			tileToFetch: common.Tile{Z: zPlus1, X: doubleX + 1, Y: doubleY + 1},
			spec: ImageSpec{
				location: image.Point{X: 256, Y: 256},
				crop:     image.Rect(0, 0, 256, 256),
			},
		},
	}
}

func generate256Instructions(t common.Tile) []instruction {
	return []instruction{
		{
			tileToFetch: t,
			spec: ImageSpec{
				location: image.Point{X: 0, Y: 0},
				crop:     image.Rect(0, 0, 256, 256),
			},
		},
	}
}

func NewZaloaService(fetcher fetcher.TileFetcher) ZaloaService {
	return &zaloaService{
		fetcher: fetcher,
	}
}
