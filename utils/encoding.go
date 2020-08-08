package utils

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"

	"golang.org/x/text/transform"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
)

// encoding determine for html page , eg: gbk gb2312 GB18030
func determineEncoding(r io.Reader) (encoding.Encoding, error) {
	bytes, err := bufio.NewReader(r).Peek(1024)
	if err != nil {
		return nil, err
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e, nil
}

func DecodeBytes(r io.Reader) (ret []byte, e error) {
	raw, e := ioutil.ReadAll(r)
	if e != nil {
		return
	}
	encode, e := determineEncoding(bytes.NewReader(raw))
	if e != nil {
		return
	}
	reader := transform.NewReader(bytes.NewReader(raw), encode.NewDecoder())
	ret, e = ioutil.ReadAll(reader)
	return
}

func DecodeString(r io.Reader) (string, error) {
	bytes, e := DecodeBytes(r)
	return string(bytes), e
}

func Id(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
