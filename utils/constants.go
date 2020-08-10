package utils

import (
	"errors"
	"time"
)

var Invalid = errors.New("Invalid Instance.")

var Shortage = errors.New("Shortage of Expected Values.")

const (
	defaultSleepTime = time.Second
	defaultDelayTime = time.Millisecond * 100
	defaultTimeout   = time.Second * 5
	defaultItems     = 10
	headUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36 Edg/83.0.478.50"
	headAccept       = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
)
