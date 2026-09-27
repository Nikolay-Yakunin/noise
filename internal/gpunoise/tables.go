package gpunoise

import (
	"encoding/binary"
	"math"
)

// PerlinTableStride is the byte size of one packed table entry: the
// permutation value as float32 followed by the three gradient components.
const PerlinTableStride = 16

// EncodePerlinTables packs the permutation table and the 3D gradients into a
// single buffer of vec4<f32> entries, ready to be uploaded to a storage
// binding. The shader reads the permutation value from .x and the gradient
// from .yzw.
//
// The tables are built on the Go side so that a GPU render matches the CPU
// noise exactly for the same seed. float32 represents every permutation
// value (0..511) without loss, so the packing is exact.
func EncodePerlinTables(perm []int32, grads [][3]float32) []byte {
	if len(perm) != len(grads) {
		panic("gpunoise: permutation and gradient tables have different lengths")
	}

	buf := make([]byte, len(perm)*PerlinTableStride)

	for i, p := range perm {
		offset := i * PerlinTableStride
		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(float32(p)))
		binary.LittleEndian.PutUint32(buf[offset+4:], math.Float32bits(grads[i][0]))
		binary.LittleEndian.PutUint32(buf[offset+8:], math.Float32bits(grads[i][1]))
		binary.LittleEndian.PutUint32(buf[offset+12:], math.Float32bits(grads[i][2]))
	}

	return buf
}
