package shared

import (
	"math"
	"math/rand"
)

type ObjectType int

const (
	ObjectTypeTree ObjectType = iota
	ObjectTypeRock
	ObjectTypeLog
	ObjectTypeGrass
)

type CollisionShape int

const (
	CollisionShapeNone CollisionShape = iota
	CollisionShapeCylinder
	CollisionShapeBox
)

type ColliderType uint8

const (
	ColliderNone ColliderType = iota
	ColliderCircle
	ColliderCapsule
)

type Collider struct {
	Type       ColliderType
	Offset     Vector3
	Radius     float32
	HalfLength float32
}

type StaticObject struct {
	Type      ObjectType
	Position  Vector3
	Rotation  float32
	Scale     float32
	Colliders []Collider
	ModelName string
}

type WorldConfig struct {
	Seed      int64
	WorldSize float32
}

const ChunkSize = float32(80.0)

func ChunkCoord(v float32) int32 {
	return int32(math.Floor(float64(v / ChunkSize)))
}

func ChunkKey(cx, cz int32) int64 {
	return (int64(cx) << 32) | int64(uint32(cz))
}

func hashChunkSeed(seed int64, cx, cz int32) int64 {
	// splitmix64-ish mixing
	x := uint64(seed) + 0x9E3779B97F4A7C15
	x ^= uint64(uint32(cx)) * 0xBF58476D1CE4E5B9
	x ^= uint64(uint32(cz)) * 0x94D049BB133111EB
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return int64(x)
}

func GenerateChunk(seed int64, cx, cz int32) []StaticObject {
	r := rand.New(rand.NewSource(hashChunkSeed(seed, cx, cz)))

	baseX := float32(cx) * ChunkSize
	baseZ := float32(cz) * ChunkSize

	randRange := func(min, max float32) float32 {
		return min + r.Float32()*(max-min)
	}

	randomPos := func() Vector3 {
		return Vector3{
			X: baseX + r.Float32()*ChunkSize,
			Y: 0,
			Z: baseZ + r.Float32()*ChunkSize,
		}
	}

	objects := make([]StaticObject, 0, 64)

	// Trees
	numTrees := 5 + r.Intn(6)
	for i := 0; i < numTrees; i++ {
		pos := randomPos()
		scale := randRange(3.6, 6.75)
		treeType := r.Intn(3)
		modelName := "tree.glb"
		if treeType == 1 {
			modelName = "tree-tall.glb"
		} else if treeType == 2 {
			modelName = "tree-autumn.glb"
		}
		objects = append(objects, StaticObject{
			Type:      ObjectTypeTree,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders(modelName, scale),
			ModelName: modelName,
		})
	}

	// Rocks
	numRocks := 3 + r.Intn(5)
	for i := 0; i < numRocks; i++ {
		pos := randomPos()
		scale := randRange(3.6, 9.0)
		rockType := r.Intn(3)
		modelName := "rock-a.glb"
		if rockType == 1 {
			modelName = "rock-b.glb"
		} else if rockType == 2 {
			modelName = "rock-c.glb"
		}
		objects = append(objects, StaticObject{
			Type:      ObjectTypeRock,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders(modelName, scale),
			ModelName: modelName,
		})
	}

	// Logs
	numLogs := r.Intn(3)
	for i := 0; i < numLogs; i++ {
		pos := randomPos()
		scale := randRange(3.6, 5.4)
		objects = append(objects, StaticObject{
			Type:      ObjectTypeLog,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders("tree-log.glb", scale),
			ModelName: "tree-log.glb",
		})
	}

	// Grass
	numGrass := 30 + r.Intn(40)
	for i := 0; i < numGrass; i++ {
		pos := randomPos()
		scale := randRange(3.6, 6.75)
		objects = append(objects, StaticObject{
			Type:      ObjectTypeGrass,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: nil,
			ModelName: "grass.glb",
		})
	}

	return objects
}

func GenerateWorld(seed int64) []StaticObject {
	// Fixed world size for now, can be parameterized if needed
	const worldSize = 500.0
	const halfSize = worldSize / 2.0

	r := rand.New(rand.NewSource(seed))

	objects := make([]StaticObject, 0)

	// Helper to random float range
	randRange := func(min, max float32) float32 {
		return min + r.Float32()*(max-min)
	}

	// 1. Generate Trees
	// Distribute trees
	numTrees := 200
	for i := 0; i < numTrees; i++ {
		pos := Vector3{
			X: randRange(-halfSize, halfSize),
			Y: 0,
			Z: randRange(-halfSize, halfSize),
		}

		// Avoid center spawn area
		if pos.X > -10 && pos.X < 10 && pos.Z > -10 && pos.Z < 10 {
			continue
		}

		scale := randRange(3.6, 6.75)

		// Randomly pick a tree model
		treeType := r.Intn(3)
		modelName := "tree.glb"
		if treeType == 1 {
			modelName = "tree-tall.glb"
		} else if treeType == 2 {
			modelName = "tree-autumn.glb"
		}

		objects = append(objects, StaticObject{
			Type:      ObjectTypeTree,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders(modelName, scale),
			ModelName: modelName,
		})
	}

	// 2. Generate Rocks
	numRocks := 100
	for i := 0; i < numRocks; i++ {
		pos := Vector3{
			X: randRange(-halfSize, halfSize),
			Y: 0,
			Z: randRange(-halfSize, halfSize),
		}

		if pos.X > -5 && pos.X < 5 && pos.Z > -5 && pos.Z < 5 {
			continue
		}

		scale := randRange(3.6, 9.0)

		rockType := r.Intn(3)
		modelName := "rock-a.glb"
		if rockType == 1 {
			modelName = "rock-b.glb"
		} else if rockType == 2 {
			modelName = "rock-c.glb"
		}

		objects = append(objects, StaticObject{
			Type:      ObjectTypeRock,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders(modelName, scale),
			ModelName: modelName,
		})
	}

	// 3. Generate Logs
	numLogs := 50
	for i := 0; i < numLogs; i++ {
		pos := Vector3{
			X: randRange(-halfSize, halfSize),
			Y: 0,
			Z: randRange(-halfSize, halfSize),
		}

		scale := randRange(3.6, 5.4)

		objects = append(objects, StaticObject{
			Type:      ObjectTypeLog,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: getColliders("tree-log.glb", scale),
			ModelName: "tree-log.glb",
		})
	}

	// 4. Generate Grass (No collision)
	numGrass := 1000
	for i := 0; i < numGrass; i++ {
		pos := Vector3{
			X: randRange(-halfSize, halfSize),
			Y: 0,
			Z: randRange(-halfSize, halfSize),
		}

		scale := randRange(3.6, 6.75)

		modelName := "grass.glb"

		objects = append(objects, StaticObject{
			Type:      ObjectTypeGrass,
			Position:  pos,
			Rotation:  randRange(0, 360),
			Scale:     scale,
			Colliders: nil,
			ModelName: modelName,
		})
	}

	return objects
}

func getColliders(modelName string, scale float32) []Collider {
	switch modelName {
	case "tree.glb", "tree-trunk.glb", "tree-tall.glb", "tree-autumn.glb", "tree-autumn-trunk.glb", "tree-autumn-tall.glb":
		return []Collider{{
			Type:   ColliderCircle,
			Offset: Vector3{X: 0, Y: 0, Z: 0},
			Radius: 0.15 * scale,
		}}
	case "tree-log.glb", "tree-log-small.glb":
		return []Collider{{
			Type:       ColliderCapsule,
			Offset:     Vector3{X: 0, Y: 0, Z: 0},
			Radius:     0.14 * scale,
			HalfLength: 0.95 * scale,
		}}
	case "rock-a.glb", "rock-b.glb", "rock-c.glb", "rock-flat.glb", "rock-flat-grass.glb", "rock-sand-a.glb", "rock-sand-b.glb", "rock-sand-c.glb":
		return []Collider{
			{Type: ColliderCircle, Offset: Vector3{X: -0.18 * scale, Y: 0, Z: -0.05 * scale}, Radius: 0.1 * scale},
			{Type: ColliderCircle, Offset: Vector3{X: 0.22 * scale, Y: 0, Z: 0.10 * scale}, Radius: 0.07 * scale},
		}
	default:
		return nil
	}
}
