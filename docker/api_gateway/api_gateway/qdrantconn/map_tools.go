package qdrantconn

import "sort"

// keysFromMap and valuesFromMap unwrap your sparse vector map into
// two parallel slices of ints and floats.
func keysFromMap(m map[int]float32) []uint32 {
	ks := make([]uint32, 0, len(m))
	for k := range m {
		ks = append(ks, uint32(k))
	}
	sort.Slice(ks, func(i, j int) bool {
		return ks[i] < ks[j]
	})
	return ks
}
func valuesFromMap(m map[int]float32) []float32 {
	// assumes keysFromMap order
	vs := make([]float32, 0, len(m))
	for _, k := range keysFromMap(m) {
		vs = append(vs, m[int(k)])
	}
	return vs
}
