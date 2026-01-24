package shared

import "math"

func rotateY(v Vector3, degrees float32) Vector3 {
	rad := float64(degrees) * math.Pi / 180.0
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))
	return Vector3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

func distSqPointToSegment2D(px, pz, ax, az, bx, bz float32) float32 {
	abx := bx - ax
	abz := bz - az
	apx := px - ax
	apz := pz - az

	abLenSq := abx*abx + abz*abz
	if abLenSq <= 0 {
		dx := px - ax
		dz := pz - az
		return dx*dx + dz*dz
	}

	t := (apx*abx + apz*abz) / abLenSq
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	cx := ax + t*abx
	cz := az + t*abz
	dx := px - cx
	dz := pz - cz
	return dx*dx + dz*dz
}

func CollidesPointWithStaticObjectXZ(obj StaticObject, x, z, radius float32) bool {
	if len(obj.Colliders) == 0 {
		return false
	}

	for _, c := range obj.Colliders {
		switch c.Type {
		case ColliderCircle:
			o := rotateY(c.Offset, obj.Rotation)
			cx := obj.Position.X + o.X
			cz := obj.Position.Z + o.Z
			minDist := radius + c.Radius
			dx := x - cx
			dz := z - cz
			if dx*dx+dz*dz < minDist*minDist {
				return true
			}
		case ColliderCapsule:
			o := rotateY(c.Offset, obj.Rotation)
			segA := rotateY(Vector3{X: 0, Y: 0, Z: -c.HalfLength}, obj.Rotation)
			segB := rotateY(Vector3{X: 0, Y: 0, Z: c.HalfLength}, obj.Rotation)

			ax := obj.Position.X + o.X + segA.X
			az := obj.Position.Z + o.Z + segA.Z
			bx := obj.Position.X + o.X + segB.X
			bz := obj.Position.Z + o.Z + segB.Z

			minDist := radius + c.Radius
			if distSqPointToSegment2D(x, z, ax, az, bx, bz) < minDist*minDist {
				return true
			}
		}
	}

	return false
}

func CollidesPointWithWorldXZ(objects []StaticObject, x, z, radius float32) bool {
	for _, obj := range objects {
		if CollidesPointWithStaticObjectXZ(obj, x, z, radius) {
			return true
		}
	}
	return false
}

