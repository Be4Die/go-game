package rendering

import (
	"game/internal/client/components"
	"game/internal/shared"
	"math"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameRenderer struct {
	camera           *rl.Camera3D
	floorModel       rl.Model
	floorTexture     rl.Texture2D
	floorShader      rl.Shader
	shaderTileLoc    int32
	nicknameTextures map[string]rl.RenderTexture2D
	font             rl.Font

	seed            int64
	chunks          map[int64][]shared.StaticObject
	centerChunkX    int32
	centerChunkZ    int32
	models          map[string]rl.Model
	colormapTexture rl.Texture2D
	isWorldLoaded   bool
	renderRadius    int32
	debugColliders  bool
}

func NewGameRenderer(camera *rl.Camera3D) *GameRenderer {
	return &GameRenderer{
		camera:           camera,
		nicknameTextures: make(map[string]rl.RenderTexture2D),
		models:           make(map[string]rl.Model),
		chunks:           make(map[int64][]shared.StaticObject),
		renderRadius:     2,
	}
}

func (gr *GameRenderer) Setup() {
	gr.font = rl.GetFontDefault()
	gr.floorTexture = rl.LoadTexture("assets/ground_texture.png")
	rl.SetTextureWrap(gr.floorTexture, rl.WrapRepeat)
	rl.GenTextureMipmaps(&gr.floorTexture)
	rl.SetTextureFilter(gr.floorTexture, rl.FilterBilinear)

	// Load colormap
	gr.colormapTexture = rl.LoadTexture("assets/survival-kit/colormap.png")
	rl.SetTextureFilter(gr.colormapTexture, rl.FilterBilinear)

	mesh := rl.GenMeshPlane(500, 500, 1, 1)
	gr.floorModel = rl.LoadModelFromMesh(mesh)

	gr.floorShader = rl.LoadShader(
		"assets/shaders/floor.vs",
		"assets/shaders/floor.fs",
	)

	gr.shaderTileLoc = rl.GetShaderLocation(gr.floorShader, "tileSize")
	rl.SetShaderValue(
		gr.floorShader,
		gr.shaderTileLoc,
		[]float32{5.0},
		rl.ShaderUniformFloat,
	)

	gr.floorModel.Materials.Shader = gr.floorShader
	gr.floorModel.Materials.GetMap(rl.MapDiffuse).Texture = gr.floorTexture
}

func (gr *GameRenderer) LoadWorld(seed int64) {
	if gr.isWorldLoaded && gr.seed == seed {
		return
	}

	gr.seed = seed
	gr.chunks = make(map[int64][]shared.StaticObject)
	gr.centerChunkX = 0
	gr.centerChunkZ = 0
	gr.isWorldLoaded = true

	// Ensure models are loaded
	requiredModels := []string{
		"tree.glb", "tree-tall.glb", "tree-autumn.glb",
		"tree-log.glb",
		"rock-a.glb", "rock-b.glb", "rock-c.glb",
		"grass.glb",
	}

	for _, name := range requiredModels {
		if _, ok := gr.models[name]; !ok {
			path := "assets/survival-kit/" + name
			model := rl.LoadModel(path)

			materials := model.GetMaterials()
			for i := range materials {
				materials[i].GetMap(rl.MapAlbedo).Texture = gr.colormapTexture
				materials[i].GetMap(rl.MapDiffuse).Texture = gr.colormapTexture
			}

			gr.models[name] = model
		}
	}
}

func (gr *GameRenderer) IsWorldLoaded() bool {
	return gr.isWorldLoaded
}

func (gr *GameRenderer) Process(em ecs.EntityManager) {
	if rl.IsKeyPressed(rl.KeyF3) {
		gr.debugColliders = !gr.debugColliders
	}

	rl.ClearBackground(rl.SkyBlue)
	rl.BeginMode3D(*gr.camera)

	floorPos := rl.Vector3{
		X: gr.camera.Position.X,
		Y: 0,
		Z: gr.camera.Position.Z,
	}
	rl.DrawModel(gr.floorModel, floorPos, 1.0, rl.White)

	if gr.isWorldLoaded {
		cx := shared.ChunkCoord(gr.camera.Position.X)
		cz := shared.ChunkCoord(gr.camera.Position.Z)
		if cx != gr.centerChunkX || cz != gr.centerChunkZ {
			gr.centerChunkX = cx
			gr.centerChunkZ = cz
			gr.ensureChunksLoaded()
		}
		if len(gr.chunks) == 0 {
			gr.ensureChunksLoaded()
		}
	}

	for _, objs := range gr.chunks {
		for _, obj := range objs {
			if model, ok := gr.models[obj.ModelName]; ok {
				rl.DrawModelEx(model,
					rl.Vector3{X: obj.Position.X, Y: obj.Position.Y, Z: obj.Position.Z},
					rl.Vector3{X: 0, Y: 1, Z: 0},
					obj.Rotation,
					rl.Vector3{X: obj.Scale, Y: obj.Scale, Z: obj.Scale},
					rl.White)
			}
		}
	}

	if gr.debugColliders {
		gr.drawWorldColliders()
	}

	for _, e := range em.FilterByMask(components.MaskTransform | components.MaskModel) {
		transform := e.Get(components.MaskTransform).(*components.Transform)
		model := e.Get(components.MaskModel).(*components.Model)

		matrix := transform.ToMatrix()
		meshes := model.GetMeshes()
		materials := model.GetMaterials()

		for _, m := range meshes {
			rl.DrawMesh(
				m,
				materials[0],
				matrix,
			)
		}
	}

	for _, e := range em.FilterByMask(components.MaskTransform | components.MaskNetworkIdentity) {
		identity := e.Get(components.MaskNetworkIdentity).(*components.NetworkIdentity)
		transform := e.Get(components.MaskTransform).(*components.Transform)
		gr.drawNickname(transform.Position, identity.Nickname)
	}

	rl.EndMode3D()
	rl.DrawFPS(10, 10)
}

func (gr *GameRenderer) ensureChunksLoaded() {
	required := make(map[int64]struct{})
	for dx := -gr.renderRadius; dx <= gr.renderRadius; dx++ {
		for dz := -gr.renderRadius; dz <= gr.renderRadius; dz++ {
			cx := gr.centerChunkX + dx
			cz := gr.centerChunkZ + dz
			k := shared.ChunkKey(cx, cz)
			required[k] = struct{}{}
			if _, ok := gr.chunks[k]; !ok {
				gr.chunks[k] = shared.GenerateChunk(gr.seed, cx, cz)
			}
		}
	}

	for k := range gr.chunks {
		if _, ok := required[k]; !ok {
			delete(gr.chunks, k)
		}
	}
}

func (gr *GameRenderer) drawWorldColliders() {
	color := rl.NewColor(255, 0, 0, 200)
	y := float32(0.05)

	for _, objs := range gr.chunks {
		for _, obj := range objs {
			if len(obj.Colliders) == 0 {
				continue
			}
			for _, c := range obj.Colliders {
				switch c.Type {
				case shared.ColliderCircle:
					o := rotateYDegrees(c.Offset, obj.Rotation)
					cx := obj.Position.X + o.X
					cz := obj.Position.Z + o.Z
					gr.drawCircleXZ(cx, y, cz, c.Radius, color)
				case shared.ColliderCapsule:
					o := rotateYDegrees(c.Offset, obj.Rotation)
					segA := rotateYDegrees(shared.Vector3{X: 0, Y: 0, Z: -c.HalfLength}, obj.Rotation)
					segB := rotateYDegrees(shared.Vector3{X: 0, Y: 0, Z: c.HalfLength}, obj.Rotation)

					ax := obj.Position.X + o.X + segA.X
					az := obj.Position.Z + o.Z + segA.Z
					bx := obj.Position.X + o.X + segB.X
					bz := obj.Position.Z + o.Z + segB.Z

					gr.drawCapsuleXZ(ax, y, az, bx, y, bz, c.Radius, color)
				}
			}
		}
	}
}

func rotateYDegrees(v shared.Vector3, degrees float32) shared.Vector3 {
	rad := float64(degrees) * math.Pi / 180.0
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))
	return shared.Vector3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

func (gr *GameRenderer) drawCircleXZ(x, y, z, r float32, color rl.Color) {
	const segments = 24
	step := float32(2.0 * math.Pi / segments)

	prevX := x + r
	prevZ := z
	for i := 1; i <= segments; i++ {
		a := float32(i) * step
		nx := x + r*float32(math.Cos(float64(a)))
		nz := z + r*float32(math.Sin(float64(a)))
		rl.DrawLine3D(rl.Vector3{X: prevX, Y: y, Z: prevZ}, rl.Vector3{X: nx, Y: y, Z: nz}, color)
		prevX = nx
		prevZ = nz
	}
}

func (gr *GameRenderer) drawCapsuleXZ(ax, ay, az, bx, by, bz, r float32, color rl.Color) {
	a := rl.Vector3{X: ax, Y: ay, Z: az}
	b := rl.Vector3{X: bx, Y: by, Z: bz}
	rl.DrawLine3D(a, b, color)

	dx := bx - ax
	dz := bz - az
	lenSq := dx*dx + dz*dz
	if lenSq <= 0 {
		gr.drawCircleXZ(ax, ay, az, r, color)
		return
	}

	invLen := 1 / float32(math.Sqrt(float64(lenSq)))
	fx := dx * invLen
	fz := dz * invLen
	rx := -fz
	rz := fx

	p1 := rl.Vector3{X: ax + rx*r, Y: ay, Z: az + rz*r}
	p2 := rl.Vector3{X: bx + rx*r, Y: by, Z: bz + rz*r}
	p3 := rl.Vector3{X: ax - rx*r, Y: ay, Z: az - rz*r}
	p4 := rl.Vector3{X: bx - rx*r, Y: by, Z: bz - rz*r}

	rl.DrawLine3D(p1, p2, color)
	rl.DrawLine3D(p3, p4, color)

	gr.drawCircleXZ(ax, ay, az, r, color)
	gr.drawCircleXZ(bx, by, bz, r, color)
}

func (gr *GameRenderer) createNicknameTexture(nickname string) rl.RenderTexture2D {
	fontSize := float32(32)
	spacing := float32(2)
	textSize := rl.MeasureTextEx(gr.font, nickname, fontSize, spacing)

	padding := int32(20)
	width := int32(textSize.X) + padding*2
	height := int32(textSize.Y) + padding*2

	rt := rl.LoadRenderTexture(width, height)

	rl.BeginTextureMode(rt)
	rl.ClearBackground(rl.Blank)

	bgColor := rl.NewColor(0, 0, 0, 180)
	rect := rl.NewRectangle(float32(padding/2), float32(padding/2),
		float32(width-padding), float32(height-padding))
	rl.DrawRectangleRounded(rect, 0.3, 10, bgColor)

	textPos := rl.NewVector2(
		float32(width)/2-textSize.X/2,
		float32(height)/2-textSize.Y/2,
	)

	shadowPos := rl.NewVector2(textPos.X+3, textPos.Y+3)
	rl.DrawTextEx(gr.font, nickname, shadowPos, fontSize, spacing, rl.Black)
	rl.DrawTextEx(gr.font, nickname, textPos, fontSize, spacing, rl.White)

	rl.EndTextureMode()

	return rt
}

func (gr *GameRenderer) drawNickname(position rl.Vector3, nickname string) {
	if nickname == "" {
		return
	}

	rt := gr.getNicknameTexture(nickname)
	desiredHeight := float32(0.8)
	aspect := float32(rt.Texture.Width) / float32(rt.Texture.Height)
	size := rl.NewVector2(desiredHeight*aspect, desiredHeight)

	textPos := rl.Vector3{
		X: position.X,
		Y: position.Y + 2.2,
		Z: position.Z,
	}

	sourceRec := rl.NewRectangle(0, 0,
		float32(rt.Texture.Width),
		-float32(rt.Texture.Height))

	up := rl.NewVector3(0.0, 1.0, 0.0)
	origin := rl.NewVector2(size.X/2, size.Y/2)
	rotation := float32(0.0)

	rl.DrawBillboardPro(
		*gr.camera,
		rt.Texture,
		sourceRec,
		textPos,
		up,
		size,
		origin,
		rotation,
		rl.White,
	)
}

func (gr *GameRenderer) getNicknameTexture(nickname string) rl.RenderTexture2D {
	if nickname == "" {
		nickname = "Player"
	}

	if tex, exists := gr.nicknameTextures[nickname]; exists {
		return tex
	}

	tex := gr.createNicknameTexture(nickname)
	gr.nicknameTextures[nickname] = tex
	return tex
}

func (gr *GameRenderer) Teardown() {
	for _, rt := range gr.nicknameTextures {
		rl.UnloadRenderTexture(rt)
	}

	for _, model := range gr.models {
		rl.UnloadModel(model)
	}

	rl.UnloadShader(gr.floorShader)
	rl.UnloadTexture(gr.floorTexture)
	rl.UnloadTexture(gr.colormapTexture)
	rl.UnloadModel(gr.floorModel)
}
