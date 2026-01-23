package components

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Transform struct {
	Position rl.Vector3
	Rotation rl.Quaternion
	Scale    rl.Vector3
}

func NewTransform() *Transform {
	return &Transform{
		Position: rl.Vector3{X: 0.0, Y: 0.0, Z: 0.0},
		Rotation: rl.Quaternion{X: 0.0, Y: 0.0, Z: 0.0, W: 1.0},
		Scale:    rl.Vector3{X: 1.0, Y: 1.0, Z: 1.0},
	}
}

func (t *Transform) Mask() uint64 {
	return MaskTransform
}

func (t *Transform) ToMatrix() rl.Matrix {
	// Создаем матрицу поворота из кватерниона
	rotationMat := rl.QuaternionToMatrix(t.Rotation)

	// Создаем матрицу масштабирования
	scaleMat := rl.MatrixScale(t.Scale.X, t.Scale.Y, t.Scale.Z)

	// Создаем матрицу трансляции
	translateMat := rl.MatrixTranslate(t.Position.X, t.Position.Y, t.Position.Z)

	// Комбинируем: сначала масштабирование, затем поворот, затем трансляция
	result := rl.MatrixMultiply(scaleMat, rotationMat)
	result = rl.MatrixMultiply(result, translateMat)

	return result
}

// Новый метод для получения угла поворота по оси Y (yaw)
func (t *Transform) GetYaw() float32 {
	// Извлекаем угол поворота вокруг оси Y из кватерниона
	sinYaw := 2.0 * (t.Rotation.W*t.Rotation.Y - t.Rotation.X*t.Rotation.Z)
	cosYaw := 1.0 - 2.0*(t.Rotation.Y*t.Rotation.Y+t.Rotation.Z*t.Rotation.Z)

	return float32(math.Atan2(float64(sinYaw), float64(cosYaw)))
}

// Метод для установки поворота по углу Y (yaw)
func (t *Transform) SetYaw(yaw float32) {
	// Создаем кватернион поворота вокруг оси Y
	t.Rotation = rl.QuaternionFromAxisAngle(rl.NewVector3(0, 1, 0), yaw)
}
