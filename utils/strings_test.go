package utils_test

import (
	"testing"

	"github.com/RaymondJiangkw/Lazy/utils"
)

func TestIntegrate(t *testing.T) {
	type Data struct {
		src    []string
		dst    []string
		result []string
	}
	data := []Data{
		Data{src: []string{"a", "b", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: []string{"a", "b", "c", "d", "e"}},
		Data{src: []string{"a", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: []string{"a", "b", "c", "d", "e"}},
		Data{src: []string{"a", "b", "c", "f", "g", "h"}, dst: []string{"c", "d", "e", "f", "h"}, result: []string{"a", "b", "c", "d", "e", "f", "g", "h"}},
	}
	var compareStringSlice = func(u []string, v []string) bool {
		if len(u) != len(v) {
			return false
		}
		for i := 0; i < len(u); i++ {
			if u[i] != v[i] {
				return false
			}
		}
		return true
	}
	for _, d := range data {
		if !compareStringSlice(utils.IntegrateStringSlices(d.src, d.dst), d.result) {
			t.Errorf("Get %v from %v, %v. Expect %v.\n", utils.IntegrateStringSlices(d.src, d.dst), d.src, d.dst, d.result)
		}
	}
}

func TestIntersect(t *testing.T) {
	type Data struct {
		src    []string
		dst    []string
		result []string
	}
	data := []Data{
		Data{src: []string{"a", "b", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: []string{"b", "c", "e"}},
		Data{src: []string{"a", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: []string{"c", "e"}},
		Data{src: []string{"a", "b", "c", "f", "g", "h"}, dst: []string{"c", "d", "e", "f", "h"}, result: []string{"c", "f", "h"}},
	}
	var compareStringSlice = func(u []string, v []string) bool {
		if len(u) != len(v) {
			return false
		}
		for i := 0; i < len(u); i++ {
			if u[i] != v[i] {
				return false
			}
		}
		return true
	}
	for _, d := range data {
		if !compareStringSlice(utils.IntersectStringSlices(d.src, d.dst), d.result) {
			t.Errorf("Get %v from %v, %v. Expect %v.\n", utils.IntersectStringSlices(d.src, d.dst), d.src, d.dst, d.result)
		}
	}
}

func TestCalculateDiff(t *testing.T) {
	type Data struct {
		src    []string
		dst    []string
		result int
	}
	data := []Data{
		Data{src: []string{"a", "b", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: 2},
		Data{src: []string{"a", "c", "e"}, dst: []string{"b", "c", "d", "e"}, result: 3},
		Data{src: []string{"a", "b", "c", "f", "g", "h"}, dst: []string{"c", "d", "e", "f", "h"}, result: 5},
	}
	for _, d := range data {
		if utils.DiffStringSlices(d.src, d.dst) != d.result {
			t.Errorf("Get %v from %v, %v. Expect %v.\n", utils.DiffStringSlices(d.src, d.dst), d.src, d.dst, d.result)
		}
	}
}
