package model

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
)

// decodeBase64UInt32 decodes a base64-encoded big-endian uint32 string.
func decodeBase64UInt32(encoded string) (uint32, error) {
	// Decode the Base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, fmt.Errorf("error decoding base64: %w", err)
	}
	if len(decodedBytes) != 4 {
		return 0, fmt.Errorf("invalid byte length: expected 4, got %d", len(decodedBytes))
	}
	return binary.BigEndian.Uint32(decodedBytes), nil
}

// decodeBase64Float32 decodes a base64-encoded big-endian float32 string.
func decodeBase64Float32(encoded string) (float32, error) {
	// Decode the Base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, fmt.Errorf("error decoding base64: %w", err)
	}
	if len(decodedBytes) != 4 {
		return 0, fmt.Errorf("invalid byte length: expected 4, got %d", len(decodedBytes))
	}
	bitsValue := binary.BigEndian.Uint32(decodedBytes)
	return math.Float32frombits(bitsValue), nil
}

// decodeFloatArray decodes a base64-encoded array of float32 values.
// This expects the input to be big-endian, consistent with other decoders in this package.
func decodeFloatArray(encoded string) ([]float32, error) {
	// base64 decode
	rawData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	// sanity check: must be a multiple of 4 bytes
	if len(rawData)%4 != 0 {
		return nil, fmt.Errorf("invalid byte length %d: not divisible by 4", len(rawData))
	}

	// convert each 4-byte chunk into a float32
	count := len(rawData) / 4
	decoded := make([]float32, count)
	for i := 0; i < count; i++ {
		// slice out the 4 bytes for element i
		bits := binary.BigEndian.Uint32(rawData[i*4 : i*4+4])
		decoded[i] = math.Float32frombits(bits)
	}

	return decoded, nil
}

// decodeFloatMap decodes a map of base64-encoded uint32 keys and float32 values.
func decodeFloatMap(encoded map[string]string) (map[int]float32, error) {
	decoded := make(map[int]float32)

	for b64Key, b64Value := range encoded {
		// Decode the Base64 string
		key, err := decodeBase64UInt32(b64Key)
		if err != nil {
			return nil, fmt.Errorf("error decoding key %s: %w", b64Key, err)
		}
		value, err := decodeBase64Float32(b64Value)
		if err != nil {
			return nil, fmt.Errorf("error decoding value %s: %w", b64Value, err)
		}

		// Convert the key to an integer and store the decoded value
		decoded[int(key)] = value
	}

	return decoded, nil
}
