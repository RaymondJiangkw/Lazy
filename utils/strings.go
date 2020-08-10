package utils

import (
	"math"
)

type StringSlices []string

/*
 * IntegrateStringSlices integrate src to dst, using variant Myers Algorithm provided by https://github.com/cj1128/myers-diff.
 */
func IntegrateStringSlices(src, dst StringSlices) StringSlices {
	return generateDiff(src, dst)
}

func DiffStringSlices(src, dst StringSlices) int {
	return calculateDiff(src, dst)
}

func IntersectStringSlices(src, dst StringSlices) StringSlices {
	return excludeDiff(src, dst)
}

type operation uint

const (
	INSERT operation = 1
	DELETE           = 2
	MOVE             = 3
)

func excludeDiff(src, dst StringSlices) (ret StringSlices) {
	script := shortestEditScript(src, dst)

	srcIndex, dstIndex := 0, 0

	for _, op := range script {
		switch op {
		case INSERT:
			dstIndex += 1

		case MOVE:
			ret = append(ret, src[srcIndex])
			srcIndex += 1
			dstIndex += 1

		case DELETE:
			srcIndex += 1
		}
	}
	return ret
}

func generateDiff(src, dst StringSlices) (ret StringSlices) {
	script := shortestEditScript(src, dst)

	srcIndex, dstIndex := 0, 0

	for _, op := range script {
		switch op {
		case INSERT:
			ret = append(ret, dst[dstIndex])
			dstIndex += 1

		case MOVE:
			ret = append(ret, src[srcIndex])
			srcIndex += 1
			dstIndex += 1

		case DELETE:
			ret = append(ret, src[srcIndex])
			srcIndex += 1
		}
	}
	return ret
}

func calculateDiff(src, dst StringSlices) (ret int) {
	script := shortestEditScript(src, dst)

	srcIndex, dstIndex := 0, 0

	for _, op := range script {
		switch op {
		case INSERT:
			ret++
			dstIndex += 1

		case MOVE:
			srcIndex += 1
			dstIndex += 1

		case DELETE:
			ret++
			srcIndex += 1
		}
	}
	return ret
}

// 生成最短的编辑脚本
func shortestEditScript(src, dst StringSlices) []operation {
	n := len(src)
	m := len(dst)
	max := n + m
	var trace []map[int]int
	var x, y int

loop:
	for d := 0; d <= max; d++ {
		// 最多只有 d+1 个 k
		v := make(map[int]int, d+2)
		trace = append(trace, v)

		// 需要注意处理对角线
		if d == 0 {
			t := 0
			for len(src) > t && len(dst) > t && src[t] == dst[t] {
				t++
			}
			v[0] = t
			if t == len(src) && t == len(dst) {
				break loop
			}
			continue
		}

		lastV := trace[d-1]

		for k := -d; k <= d; k += 2 {
			// 向下
			if k == -d || (k != d && lastV[k-1] < lastV[k+1]) {
				x = lastV[k+1]
			} else { // 向右
				x = lastV[k-1] + 1
			}

			y = x - k

			// 处理对角线
			for x < n && y < m && src[x] == dst[y] {
				x, y = x+1, y+1
			}

			v[k] = x

			if x == n && y == m {
				break loop
			}
		}
	}

	// 反向回溯
	var script []operation

	x = n
	y = m
	var k, prevK, prevX, prevY int

	for d := len(trace) - 1; d > 0; d-- {
		k = x - y
		lastV := trace[d-1]

		if k == -d || (k != d && lastV[k-1] < lastV[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX = lastV[prevK]
		prevY = prevX - prevK

		for x > prevX && y > prevY {
			script = append(script, MOVE)
			x -= 1
			y -= 1
		}

		if x == prevX {
			script = append(script, INSERT)
		} else {
			script = append(script, DELETE)
		}

		x, y = prevX, prevY
	}

	if trace[0][0] != 0 {
		for i := 0; i < trace[0][0]; i++ {
			script = append(script, MOVE)
		}
	}

	return reverse(script)
}

func reverse(s []operation) []operation {
	result := make([]operation, len(s))

	for i, v := range s {
		result[len(s)-1-i] = v
	}

	return result
}

func PartitionStringSlices(data []*StringSlices, criterion interface{}) ([][]*StringSlices, error) {
	// Interpret Criterion
	var threshold int
	var ratio float64
	switch criterion.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		threshold = criterion.(int)
	case float32, float64:
		ratio = criterion.(float64)
	default:
		return nil, Invalid
	}
	// Check Special Cases
	if threshold < 0 || ratio < 0 {
		return nil, Invalid
	}
	if threshold == 0 && ratio == 0 {
		return [][]*StringSlices{data}, nil
	}
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) == 1 {
		return [][]*StringSlices{data}, nil
	}
	// Initialize
	var ret [][]*StringSlices
	ret = append(ret, []*StringSlices{data[0]})
	cnt := 1

	for i := 1; i < len(data); i++ {
		inserted := false
		// Loop over every group to check whether is able to insert into.
		for j := 0; j < cnt; j++ {
			valid := true
			// Loop over every item to check difference.
			for _, d := range ret[j] {
				diff := DiffStringSlices(*data[i], *d)
				if ratio > 0 {
					if float64(diff) > math.Min(float64(len(*data[i]))*ratio, float64(len(*d))*ratio) {
						valid = false
						break
					}
				} else {
					if diff > threshold {
						valid = false
						break
					}
				}
			}
			if valid {
				ret[j] = append(ret[j], data[i])
				inserted = true
				break
			}
		}
		if !inserted {
			ret = append(ret, []*StringSlices{data[i]})
			cnt++
		}
	}
	return ret[:cnt], nil
}
