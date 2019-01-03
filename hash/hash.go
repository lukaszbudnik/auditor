package hash

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
)

// Serialize serializes passed struct to bytes using GOB
func Serialize(object interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(object); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ComputeHash computes hash for passed struct using default Serialize function
func ComputeHash(object interface{}) (string, error) {
	return ComputeHashWithSerialize(object, Serialize)
}

// ComputeHashWithSerialize computes hash based on passed struct
// before computing hash, serializes object using provided function
func ComputeHashWithSerialize(object interface{}, serialize func(object interface{}) ([]byte, error)) (string, error) {
	bytes, err := serialize(object)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(bytes)
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed), nil
}
