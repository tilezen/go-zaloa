package common

type TileKind string
type TileEncoding string

const (
	TileType_TERRARIUM = TileKind("terrarium")
	TileType_NORMAL    = TileKind("normal")
	TileEncoding_PNG   = TileEncoding("png")
	TileEncoding_WEBP  = TileEncoding("webp")
)
