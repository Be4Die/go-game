package components

import rl "github.com/gen2brain/raylib-go/raylib"

type Model struct {
	rl.Model
}

func (m *Model) Mask() uint64 {
	return MaskModel
}

func NewModel(modelPath string) *Model {
	model := rl.LoadModel(modelPath)
	return &Model{
		Model: model,
	}
}

func (m *Model) WithTexture(texturePath string) *Model {
	texture := rl.LoadTexture(texturePath)
	materials := m.GetMaterials()
	for i := range materials {
		materials[i].GetMap(rl.MapAlbedo).Texture = texture
	}
	return m
}
