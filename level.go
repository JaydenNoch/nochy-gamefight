package main

// Tile indices inside tiles.png (32x32 cells in a single row).
const (
	TileGround = 0
	TileDirt   = 1
	TileStone  = 2
	TileWood   = 3
	TileBrick  = 4
	TileSpike  = 5
	TileTorch  = 6
	TileCoin   = 7
	tileSize   = 32
	tileCols   = 8
)

// Platform is a solid surface made of one tile type.
type Platform struct {
	Rect
	Tile int
}

// Level describes the static world: ground, platforms, hazards, pickups and
// enemy spawn points.
type Level struct {
	Width     float64
	Height    float64
	GroundY   float64
	Platforms []Platform
	Coins     []Rect
	Spawns    []Vec2
	BossSpawn Vec2
}

// Vec2 is a simple 2D vector.
type Vec2 struct {
	X, Y float64
}

// NewLevel builds the playable level layout.
func NewLevel() *Level {
	const groundY = 470.0
	const width = 3600.0
	lvl := &Level{
		Width:   width,
		Height:  540,
		GroundY: groundY,
	}

	// Solid ground strips (with gaps to make the level interesting).
	lvl.Platforms = append(lvl.Platforms,
		Platform{Rect{0, groundY, 900, 70}, TileGround},
		Platform{Rect{980, groundY, 700, 70}, TileGround},
		Platform{Rect{1760, groundY, 620, 70}, TileGround},
		Platform{Rect{2460, groundY, 1140, 70}, TileGround},
	)

	// Floating platforms. Every one sits high enough that the warrior can walk
	// underneath while standing on the ground (top <= 340 leaves 16px of
	// clearance above a 90px tall player).
	lvl.Platforms = append(lvl.Platforms,
		Platform{Rect{300, 340, 128, 24}, TileBrick},
		Platform{Rect{520, 280, 160, 24}, TileWood},
		Platform{Rect{820, 340, 128, 24}, TileBrick},
		Platform{Rect{1080, 300, 128, 24}, TileWood},
		Platform{Rect{1300, 340, 160, 24}, TileBrick},
		Platform{Rect{1560, 300, 128, 24}, TileWood},
		Platform{Rect{1900, 340, 128, 24}, TileBrick},
		Platform{Rect{2140, 280, 160, 24}, TileWood},
		Platform{Rect{2420, 340, 128, 24}, TileBrick},
		Platform{Rect{2700, 300, 128, 24}, TileWood},
		Platform{Rect{2980, 340, 160, 24}, TileBrick},
		Platform{Rect{3240, 280, 128, 24}, TileWood},
	)

	// Spikes (damage hazards) sitting on the ground.
	lvl.Platforms = append(lvl.Platforms,
		Platform{Rect{1450, groundY - 20, 96, 20}, TileSpike},
		Platform{Rect{2620, groundY - 20, 96, 20}, TileSpike},
	)

	// Coins to collect for extra score. Every fourth coin sits at walking
	// height so it is grabbed on the run, the rest reward a jump.
	for i, x := range []float64{360, 560, 860, 1120, 1360, 1600, 1960, 2200, 2480, 2760, 3040, 3300} {
		var y float64
		switch i % 4 {
		case 0:
			y = 430
		case 1:
			y = 300
		case 2:
			y = 250
		default:
			y = 345
		}
		lvl.Coins = append(lvl.Coins, Rect{x, y, 26, 26})
	}

	// Enemy spawn points (x, groundY). Twenty of them, spread evenly across the
	// four solid ground strips so none of them starts over a pit.
	lvl.Spawns = []Vec2{
		{300, groundY}, {480, groundY}, {660, groundY}, {840, groundY},
		{1040, groundY}, {1180, groundY}, {1320, groundY}, {1460, groundY}, {1620, groundY},
		{1820, groundY}, {1960, groundY}, {2100, groundY}, {2220, groundY}, {2300, groundY},
		{2520, groundY}, {2680, groundY}, {2820, groundY}, {2960, groundY}, {3040, groundY}, {3120, groundY},
	}

	// The final boss waits at the end of the last ground strip.
	lvl.BossSpawn = Vec2{3450, groundY}

	return lvl
}

// Hazard reports whether a platform is a damaging hazard.
func (p Platform) Hazard() bool { return p.Tile == TileSpike }
