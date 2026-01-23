package rendering

import (
	"game/internal/client"

	raygui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type MenuRenderer struct {
	inputText string
	editMode  bool
	container *client.DataContainer
}

func NewMenuRenderer(container *client.DataContainer) *MenuRenderer {
	return &MenuRenderer{
		inputText: "Player",
		editMode:  false,
		container: container,
	}
}

func (mr *MenuRenderer) Process() {
	// Draw white background
	rl.ClearBackground(rl.White)

	// Calculate center positions
	screenWidth := float32(rl.GetScreenWidth())
	screenHeight := float32(rl.GetScreenHeight())

	// Устанавливаем увеличенный размер шрифта для элементов raygui
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 30)

	// Text input field
	textBoxWidth := float32(400) // Увеличил ширину
	textBoxHeight := float32(60) // Увеличил высоту
	textBoxX := (screenWidth - textBoxWidth) / 2
	textBoxY := (screenHeight-textBoxHeight)/2 - 50

	textBounds := rl.NewRectangle(textBoxX, textBoxY, textBoxWidth, textBoxHeight)

	// Draw label
	labelText := "Enter your nickname:"
	// labelWidth := rl.MeasureText(labelText, 20)
	rl.DrawText(labelText, int32(textBoxX), int32(textBoxY-30), 25, rl.Black) // Увеличил размер текста с 20 до 25

	// Text input
	if raygui.TextBox(textBounds, &mr.inputText, 32, mr.editMode) {
		mr.editMode = !mr.editMode
	}

	// Activate on click
	if !mr.editMode && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), textBounds) {
			mr.editMode = true
		}
	}

	// Play button
	buttonWidth := float32(300) // Увеличил ширину
	buttonHeight := float32(70) // Увеличил высоту
	buttonX := (screenWidth - buttonWidth) / 2
	buttonY := textBoxY + textBoxHeight + 30

	buttonBounds := rl.NewRectangle(buttonX, buttonY, buttonWidth, buttonHeight)
	if raygui.Button(buttonBounds, "Play") {
		client.GameBoot(mr.container, mr.inputText)
	}

	// Title
	title := "My Game"
	titleWidth := rl.MeasureText(title, 40)
	rl.DrawText(title, int32((screenWidth-float32(titleWidth))/2), 100, 50, rl.Black) // Увеличил размер текста с 40 до 50
}
