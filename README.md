# go-zaloa

Serves Terrain tiles using the [Mapzen terrain tiles](https://registry.opendata.aws/terrain-tiles/) with more complex shapes. Zaloa supports 256x256, 512x512, 260x260, and 516x516 pixel tiles. The source tiles are 256x256 pixels, so Zaloa fetches multiple tiles and stitches them together to get the other tile sizes.   

This is a port of the Python [zaloa](https://github.com/tilezen/zaloa) to Go.
 
