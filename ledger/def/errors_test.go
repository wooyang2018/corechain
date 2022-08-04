package def

import (
	"errors"
	"testing"
)

func TestNormalizeKVError(t *testing.T) {
	kvErr := errors.New("Key not found")
	err := NormalizeKVError(kvErr)
	if err != nil {
		t.Log(err)
	}
	p2pErr := errors.New("invalid stream")
	err = NormalizeKVError(p2pErr)
	if err != nil {
		t.Log(err)
	}
}
