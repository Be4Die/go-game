package rendering

import (
	"game/internal/client/components"
	"game/internal/shared"
	"math"

	"github.com/andygeiss/ecs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameRenderer struct {
	camera             *rl.Camera3D
	skyModel           rl.Model
	skyShader          rl.Shader
	skyCameraLoc       int32
	skyTopColorLoc     int32
	skyHorizonColorLoc int32
	skySunDirLoc       int32
	skySunColorLoc     int32

	floorModel         rl.Model
	floorTexture       rl.Texture2D
	floorShader        rl.Shader
	shaderTileLoc      int32
	floorFogCameraLoc  int32
	floorFogColorLoc   int32
	floorFogStartLoc   int32
	floorFogEndLoc     int32
	floorLightDirLoc   int32
	floorLightColorLoc int32
	floorAmbientLoc    int32
	nicknameTextures   map[string]rl.RenderTexture2D
	font               rl.Font

	seed               int64
	chunks             map[int64][]shared.StaticObject
	centerChunkX       int32
	centerChunkZ       int32
	models             map[string]rl.Model
	colormapTexture    rl.Texture2D
	litShader          rl.Shader
	litCameraLoc       int32
	litFogColorLoc     int32
	litFogStartLoc     int32
	litFogEndLoc       int32
	litLightDirLoc     int32
	litLightColorLoc   int32
	litAmbientLoc      int32
	litSpecStrengthLoc int32
	litShininessLoc    int32
	isWorldLoaded      bool
	renderRadius       int32
	debugColliders     bool
	skyTopColor        rl.Color
	skyHorizonColor    rl.Color
	fogColor           rl.Color
	fogStart           float32
	fogEnd             float32
	lightDir           rl.Vector3
	lightColor         rl.Color
	ambientColor       rl.Color
}

func NewGameRenderer(camera *rl.Camera3D) *GameRenderer {
	return &GameRenderer{
		camera:           camera,
		nicknameTextures: make(map[string]rl.RenderTexture2D),
		models:           make(map[string]rl.Model),
		chunks:           make(map[int64][]shared.StaticObject),
		renderRadius:     2,
		skyTopColor:      rl.NewColor(70, 130, 210, 255),
		skyHorizonColor:  rl.NewColor(200, 225, 245, 255),
		fogColor:         rl.NewColor(195, 215, 235, 255),
		fogStart:         60,
		fogEnd:           220,
		lightDir:         rl.Vector3{X: -0.35, Y: 0.85, Z: 0.25},
		lightColor:       rl.NewColor(255, 252, 245, 255),
		ambientColor:     rl.NewColor(252, 254, 206, 255),
	}
}

func (gr *GameRenderer) Setup() {
	gr.font = rl.GetFontDefault()

	skyMesh := rl.GenMeshSphere(1.0, 32, 16)
	gr.skyModel = rl.LoadModelFromMesh(skyMesh)
	gr.skyShader = rl.LoadShader("assets/shaders/sky.vs", "assets/shaders/sky.fs")
	gr.skyCameraLoc = rl.GetShaderLocation(gr.skyShader, "cameraPos")
	gr.skyTopColorLoc = rl.GetShaderLocation(gr.skyShader, "topColor")
	gr.skyHorizonColorLoc = rl.GetShaderLocation(gr.skyShader, "horizonColor")
	gr.skySunDirLoc = rl.GetShaderLocation(gr.skyShader, "sunDir")
	gr.skySunColorLoc = rl.GetShaderLocation(gr.skyShader, "sunColor")
	gr.skyModel.Materials.Shader = gr.skyShader
	gr.floorTexture = rl.LoadTexture("assets/ground_texture.png")
	rl.SetTextureWrap(gr.floorTexture, rl.WrapRepeat)
	rl.GenTextureMipmaps(&gr.floorTexture)
	rl.SetTextureFilter(gr.floorTexture, rl.FilterBilinear)

	// Load colormap
	gr.colormapTexture = rl.LoadTexture("assets/survival-kit/colormap.png")
	rl.GenTextureMipmaps(&gr.colormapTexture)
	rl.SetTextureFilter(gr.colormapTexture, rl.FilterBilinear)

	mesh := rl.GenMeshPlane(500, 500, 1, 1)
	gr.floorModel = rl.LoadModelFromMesh(mesh)

	gr.floorShader = rl.LoadShader(
		"assets/shaders/floor.vs",
		"assets/shaders/floor.fs",
	)

	gr.shaderTileLoc = rl.GetShaderLocation(gr.floorShader, "tileSize")
	gr.floorFogCameraLoc = rl.GetShaderLocation(gr.floorShader, "cameraPos")
	gr.floorFogColorLoc = rl.GetShaderLocation(gr.floorShader, "fogColor")
	gr.floorFogStartLoc = rl.GetShaderLocation(gr.floorShader, "fogStart")
	gr.floorFogEndLoc = rl.GetShaderLocation(gr.floorShader, "fogEnd")
	gr.floorLightDirLoc = rl.GetShaderLocation(gr.floorShader, "lightDir")
	gr.floorLightColorLoc = rl.GetShaderLocation(gr.floorShader, "lightColor")
	gr.floorAmbientLoc = rl.GetShaderLocation(gr.floorShader, "ambientColor")
	rl.SetShaderValue(
		gr.floorShader,
		gr.shaderTileLoc,
		[]float32{5.0},
		rl.ShaderUniformFloat,
	)

	gr.floorModel.Materials.Shader = gr.floorShader
	gr.floorModel.Materials.GetMap(rl.MapDiffuse).Texture = gr.floorTexture

	gr.litShader = rl.LoadShader(
		"assets/shaders/lit_fog.vs",
		"assets/shaders/lit_fog.fs",
	)
	gr.litCameraLoc = rl.GetShaderLocation(gr.litShader, "cameraPos")
	gr.litFogColorLoc = rl.GetShaderLocation(gr.litShader, "fogColor")
	gr.litFogStartLoc = rl.GetShaderLocation(gr.litShader, "fogStart")
	gr.litFogEndLoc = rl.GetShaderLocation(gr.litShader, "fogEnd")
	gr.litLightDirLoc = rl.GetShaderLocation(gr.litShader, "lightDir")
	gr.litLightColorLoc = rl.GetShaderLocation(gr.litShader, "lightColor")
	gr.litAmbientLoc = rl.GetShaderLocation(gr.litShader, "ambientColor")
	gr.litSpecStrengthLoc = rl.GetShaderLocation(gr.litShader, "specStrength")
	gr.litShininessLoc = rl.GetShaderLocation(gr.litShader, "shininess")
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
				materials[i].Shader = gr.litShader
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

	cameraPos := []float32{gr.camera.Position.X, gr.camera.Position.Y, gr.camera.Position.Z}
	fogColor := []float32{float32(gr.fogColor.R) / 255.0, float32(gr.fogColor.G) / 255.0, float32(gr.fogColor.B) / 255.0}
	lightDir := []float32{gr.lightDir.X, gr.lightDir.Y, gr.lightDir.Z}
	lightColor := []float32{float32(gr.lightColor.R) / 255.0, float32(gr.lightColor.G) / 255.0, float32(gr.lightColor.B) / 255.0}
	ambient := []float32{float32(gr.ambientColor.R) / 255.0, float32(gr.ambientColor.G) / 255.0, float32(gr.ambientColor.B) / 255.0}
	rl.SetShaderValue(gr.floorShader, gr.floorFogCameraLoc, cameraPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.floorShader, gr.floorFogColorLoc, fogColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.floorShader, gr.floorFogStartLoc, []float32{gr.fogStart}, rl.ShaderUniformFloat)
	rl.SetShaderValue(gr.floorShader, gr.floorFogEndLoc, []float32{gr.fogEnd}, rl.ShaderUniformFloat)
	rl.SetShaderValue(gr.floorShader, gr.floorLightDirLoc, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.floorShader, gr.floorLightColorLoc, lightColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.floorShader, gr.floorAmbientLoc, ambient, rl.ShaderUniformVec3)

	rl.SetShaderValue(gr.litShader, gr.litCameraLoc, cameraPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.litShader, gr.litFogColorLoc, fogColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.litShader, gr.litFogStartLoc, []float32{gr.fogStart}, rl.ShaderUniformFloat)
	rl.SetShaderValue(gr.litShader, gr.litFogEndLoc, []float32{gr.fogEnd}, rl.ShaderUniformFloat)
	rl.SetShaderValue(gr.litShader, gr.litLightDirLoc, lightDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.litShader, gr.litLightColorLoc, lightColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.litShader, gr.litAmbientLoc, ambient, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.litShader, gr.litSpecStrengthLoc, []float32{0.18}, rl.ShaderUniformFloat)
	rl.SetShaderValue(gr.litShader, gr.litShininessLoc, []float32{24.0}, rl.ShaderUniformFloat)
	rl.BeginMode3D(*gr.camera)

	skyTop := []float32{float32(gr.skyTopColor.R) / 255.0, float32(gr.skyTopColor.G) / 255.0, float32(gr.skyTopColor.B) / 255.0}
	skyHorizon := []float32{float32(gr.skyHorizonColor.R) / 255.0, float32(gr.skyHorizonColor.G) / 255.0, float32(gr.skyHorizonColor.B) / 255.0}
	sunDir := []float32{-0.35, 0.85, 0.25}
	sunColor := []float32{1.0, 0.95, 0.85}
	rl.SetShaderValue(gr.skyShader, gr.skyCameraLoc, cameraPos, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.skyShader, gr.skyTopColorLoc, skyTop, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.skyShader, gr.skyHorizonColorLoc, skyHorizon, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.skyShader, gr.skySunDirLoc, sunDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(gr.skyShader, gr.skySunColorLoc, sunColor, rl.ShaderUniformVec3)

	rl.DisableBackfaceCulling()
	rl.DisableDepthMask()
	rl.DrawModelEx(gr.skyModel, gr.camera.Position, rl.Vector3{X: 0, Y: 1, Z: 0}, 0, rl.Vector3{X: 800, Y: 800, Z: 800}, rl.White)
	rl.EnableDepthMask()
	rl.EnableBackfaceCulling()

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
		for i := range materials {
			materials[i].Shader = gr.litShader
		}

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
	rl.UnloadShader(gr.litShader)
	rl.UnloadShader(gr.skyShader)
	rl.UnloadTexture(gr.floorTexture)
	rl.UnloadTexture(gr.colormapTexture)
	rl.UnloadModel(gr.floorModel)
	rl.UnloadModel(gr.skyModel)
}
