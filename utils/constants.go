package utils

import (
	"errors"
	"time"
)

var Invalid = errors.New("Invalid Instance.")

const (
	defaultDelayTime = time.Millisecond * 100
	defaultTimeout   = time.Second * 0
)
