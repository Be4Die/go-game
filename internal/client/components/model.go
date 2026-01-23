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
	m.Materials.GetMap(rl.MapDiffuse).Texture = rl.LoadTexture(texturePath)
	return m
}
