package rendering

import (
	"game/internal/client/components"

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
}

func NewGameRenderer(camera *rl.Camera3D) *GameRenderer {
	return &GameRenderer{
		camera:           camera,
		nicknameTextures: make(map[string]rl.RenderTexture2D),
	}
}

func (gr *GameRenderer) Setup() {
	gr.font = rl.GetFontDefault()
	gr.floorTexture = rl.LoadTexture("assets/ground_texture.png")
	rl.SetTextureWrap(gr.floorTexture, rl.WrapRepeat)
	rl.GenTextureMipmaps(&gr.floorTexture)
	rl.SetTextureFilter(gr.floorTexture, rl.FilterBilinear)

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

func (gr *GameRenderer) Process(em ecs.EntityManager) {
	rl.ClearBackground(rl.SkyBlue)
	rl.BeginMode3D(*gr.camera)

	floorPos := rl.Vector3{
		X: gr.camera.Position.X,
		Y: 0,
		Z: gr.camera.Position.Z,
	}
	rl.DrawModel(gr.floorModel, floorPos, 1.0, rl.White)

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

	for _, e := range em.FilterByMask(components.MaskTransform) {
		var nickname string

		if player := e.Get(components.MaskPlayer); player != nil {
			nickname = player.(*components.Player).Nickname
		} else if remote := e.Get(components.MaskRemote); remote != nil {
			nickname = remote.(*components.Remote).Nickname
		} else {
			continue
		}

		transform := e.Get(components.MaskTransform).(*components.Transform)
		gr.drawNickname(transform.Position, nickname)
	}

	rl.EndMode3D()
	rl.DrawFPS(10, 10)
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

	rl.UnloadShader(gr.floorShader)
	rl.UnloadTexture(gr.floorTexture)
	rl.UnloadModel(gr.floorModel)
}
