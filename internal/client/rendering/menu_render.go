package rendering

import (
	"fmt"
	"game/internal/client"
	"os"
	"path/filepath"
	"strings"

	raygui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type MenuRenderer struct {
	inputText string
	container *client.DataContainer

	models        []rl.Model
	modelPaths    []string
	animations    [][]rl.ModelAnimation
	selectedIndex int
	texture       rl.Texture2D
	hasTexture    bool

	camera      rl.Camera3D
	animFrame   int32
	currentAnim int32
	animLoop    bool
	animPlaying bool

	targetIndex    int
	isSliding      bool
	slideProgress  float32
	slideDirection int
}

func NewMenuRenderer(container *client.DataContainer) *MenuRenderer {
	return &MenuRenderer{
		inputText: "Player",
		container: container,
		camera: rl.Camera3D{
			Position:   rl.NewVector3(0.0, 3.0, 8.0),
			Target:     rl.NewVector3(0.0, 1.0, 0.0),
			Up:         rl.NewVector3(0.0, 1.0, 0.0),
			Fovy:       45.0,
			Projection: rl.CameraPerspective,
		},

		currentAnim: 1,
		animLoop:    true,
		animPlaying: true,
	}
}

func (mr *MenuRenderer) Setup() {
	files, err := os.ReadDir("assets/characters")
	if err != nil {
		fmt.Printf("Error reading assets directory: %v\n", err)
		wd, _ := os.Getwd()
		fmt.Printf("Current working directory: %s\n", wd)
		return
	}

	mr.texture = rl.LoadTexture("assets/characters/colormap.png")
	mr.hasTexture = mr.texture.ID != 0

	fmt.Printf("Found %d files in assets/characters\n", len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasPrefix(name, "character-") && strings.HasSuffix(name, ".glb") {
			path := filepath.Join("assets/characters", name)
			model := rl.LoadModel(path)
			if mr.hasTexture {
				materials := model.GetMaterials()
				for i := range materials {
					materials[i].GetMap(rl.MapAlbedo).Texture = mr.texture
				}
			}

			anims := rl.LoadModelAnimations(path)

			mr.models = append(mr.models, model)
			mr.modelPaths = append(mr.modelPaths, path)
			mr.animations = append(mr.animations, anims)
		}
	}

	if len(mr.models) == 0 {
		fmt.Println("No character models found! Ensure assets/characters contains character-*.glb files.")
	} else {
		fmt.Printf("Loaded %d character models.\n", len(mr.models))
	}
}

func (mr *MenuRenderer) Teardown() {
	for _, model := range mr.models {
		rl.UnloadModel(model)
	}
	if mr.hasTexture {
		rl.UnloadTexture(mr.texture)
	}
}

func (mr *MenuRenderer) Process() {
	mr.updateAnimation()
	mr.updateSlide()

	rl.BeginMode3D(mr.camera)

	mr.drawCharacterCarousel()

	rl.EndMode3D()

	mr.drawUI()
}

func (mr *MenuRenderer) updateAnimation() {
	if len(mr.models) == 0 {
		return
	}

	if mr.isSliding {
	}

	for i := range mr.models {
		anims := mr.animations[i]
		if len(anims) == 0 {
			continue
		}

		animIdx := 1
		if animIdx >= len(anims) {
			animIdx = 0
		}
		if i == mr.selectedIndex && !mr.isSliding {
			animIdx = int(mr.currentAnim)
			if animIdx >= len(anims) {
				animIdx = 0
			}
		}

		anim := anims[animIdx]
		frameCount := int32(anim.FrameCount)

		if i == mr.selectedIndex && !mr.isSliding {
			mr.animFrame++
			if mr.animFrame >= frameCount {
				if mr.animLoop {
					mr.animFrame = 0
				} else {
					mr.currentAnim = 1
					mr.animLoop = true
					mr.animFrame = 0
				}
			}
			rl.UpdateModelAnimation(mr.models[i], anim, int32(mr.animFrame))
		} else {
			idleIdx := 1
			if idleIdx >= len(anims) {
				idleIdx = 0
			}
			loopFrame := (int(rl.GetTime() * 30)) % int(anims[idleIdx].FrameCount)
			rl.UpdateModelAnimation(mr.models[i], anims[idleIdx], int32(loopFrame))
		}
	}
}

func (mr *MenuRenderer) updateSlide() {
	if !mr.isSliding {
		return
	}

	slideSpeed := float32(2.0)
	mr.slideProgress += slideSpeed * rl.GetFrameTime()

	if mr.slideProgress >= 1.0 {
		mr.isSliding = false
		mr.slideProgress = 0
		mr.selectedIndex = mr.targetIndex

		mr.currentAnim = 11
		mr.animLoop = false
		mr.animFrame = 0
	}
}

func (mr *MenuRenderer) drawCharacterCarousel() {
	if len(mr.models) == 0 {
		return
	}

	count := len(mr.models)

	centerPos := rl.NewVector3(0, 0, 0)
	leftPos := rl.NewVector3(-2.5, 0, 0)
	rightPos := rl.NewVector3(2.5, 0, 0)

	curr := mr.selectedIndex
	prev := (curr - 1 + count) % count
	next := (curr + 1) % count

	offset := float32(0)
	if mr.isSliding {
		if mr.slideDirection == -1 {
			offset = -2.5 * mr.slideProgress
		} else {
			offset = 2.5 * mr.slideProgress
		}
	}

	draw := func(index int, basePos rl.Vector3) {
		pos := rl.Vector3Add(basePos, rl.NewVector3(offset, 0, 0))
		scale := float32(2.0)
		rl.DrawModel(mr.models[index], pos, scale, rl.White)
	}

	if mr.isSliding {
		if mr.slideDirection == -1 {
			draw(prev, leftPos)
			draw(curr, centerPos)
			draw(next, rightPos)
			nextNext := (next + 1) % count
			draw(nextNext, rl.NewVector3(5.0, 0, 0))
		} else {
			draw(next, rightPos)
			draw(curr, centerPos)
			draw(prev, leftPos)
			prevPrev := (prev - 1 + count) % count
			draw(prevPrev, rl.NewVector3(-5.0, 0, 0))
		}
	} else {
		draw(prev, leftPos)
		draw(curr, centerPos)
		draw(next, rightPos)
	}
}

func (mr *MenuRenderer) drawUI() {
	screenWidth := float32(rl.GetScreenWidth())
	screenHeight := float32(rl.GetScreenHeight())

	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 30)

	title := "Select Character"
	titleWidth := rl.MeasureText(title, 40)
	rl.DrawText(title, int32((screenWidth-float32(titleWidth))/2), 30, 40, rl.DarkGray)

	status := fmt.Sprintf("Characters: %d", len(mr.models))
	rl.DrawText(status, 10, 30, 20, rl.DarkGray)
	if len(mr.modelPaths) > 0 {
		current := filepath.Base(mr.modelPaths[mr.selectedIndex])
		rl.DrawText(current, 10, 55, 20, rl.DarkGray)
	}

	arrowY := float32(screenHeight)/2 - 30
	if raygui.Button(rl.NewRectangle(50, arrowY, 60, 60), "<") {
		if !mr.isSliding && len(mr.models) > 0 {
			mr.targetIndex = (mr.selectedIndex - 1 + len(mr.models)) % len(mr.models)
			mr.isSliding = true
			mr.slideDirection = 1
			mr.slideProgress = 0
		}
	}

	if raygui.Button(rl.NewRectangle(screenWidth-110, arrowY, 60, 60), ">") {
		if !mr.isSliding && len(mr.models) > 0 {
			mr.targetIndex = (mr.selectedIndex + 1) % len(mr.models)
			mr.isSliding = true
			mr.slideDirection = -1
			mr.slideProgress = 0
		}
	}

	textBoxWidth := float32(400)
	textBoxHeight := float32(50)
	textBoxX := (screenWidth - textBoxWidth) / 2

	// Поле Nickname
	nicknameY := screenHeight - 180
	rl.DrawText("Nickname:", int32(textBoxX), int32(nicknameY-25), 20, rl.DarkGray)
	nicknameEdit := raygui.TextBox(rl.NewRectangle(textBoxX, nicknameY, textBoxWidth, textBoxHeight), &mr.inputText, 32, true)
	if nicknameEdit {
		// Поле активно
	}

	// Статус подключения (если есть)
	isConnecting := mr.container.GameState == client.GameStateConnecting
	hasError := mr.container.GameState == client.GameStateError && mr.container.NetworkError != ""

	buttonY := nicknameY + textBoxHeight + 20
	buttonLabel := "PLAY"
	if isConnecting {
		buttonLabel = "CONNECTING..."
	}

	// Кнопка PLAY — блокируем во время подключения
	if isConnecting {
		raygui.Button(rl.NewRectangle(textBoxX, buttonY, textBoxWidth, 55), buttonLabel)
	} else {
		if raygui.Button(rl.NewRectangle(textBoxX, buttonY, textBoxWidth, 55), buttonLabel) {
			modelPath := "assets/characters/character-male-c.glb"
			if len(mr.modelPaths) > 0 {
				modelPath = mr.modelPaths[mr.selectedIndex]
				modelPath = strings.ReplaceAll(modelPath, "\\", "/")
			}
			// Запускаем подключение в отдельной горутине, чтобы не блокировать UI
			go client.GameBoot(mr.container, mr.inputText, modelPath)
		}
	}

	// Отображаем статус подключения
	statusY := buttonY + 70
	if isConnecting && mr.container.ConnectionStatus != "" {
		statusText := mr.container.ConnectionStatus
		statusWidth := rl.MeasureText(statusText, 20)
		rl.DrawText(statusText, int32((screenWidth-float32(statusWidth))/2), int32(statusY), 20, rl.DarkBlue)
	} else if hasError {
		errText := mr.container.NetworkError
		errWidth := rl.MeasureText(errText, 20)
		rl.DrawText(errText, int32((screenWidth-float32(errWidth))/2), int32(statusY), 20, rl.Red)
	}
}
