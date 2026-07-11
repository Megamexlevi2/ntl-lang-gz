package native

import "runtime"

func SupportsNative() bool { return false }

func Arch() string { return runtime.GOARCH }

func MmapExec(_ int) []byte                               { return nil }
func MunmapExec(_ []byte)                                 {}
func CallNativeI64(_ []byte, _ *int64, _ *int64, _ int64) {}
func EmitCountSum() []byte                                { return nil }
func EmitCount() []byte                                   { return nil }

func RunCount(start, limit int64) int64 {
	if limit <= start {
		return start
	}
	return limit
}

func RunCountInclusive(start, limit int64) int64 {
	if limit < start {
		return start
	}
	return limit + 1
}

func RunCountSum(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 0
	}
	n := limit - start + 1
	sum := n * (start + limit) / 2
	return limit + 1, sum
}

func RunCountSumExclusive(start, limit int64) (int64, int64) {
	if limit <= start {
		return start, 0
	}
	return RunCountSum(start, limit-1)
}

func RunSumSquares(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 0
	}
	sumSquaresUpTo := func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		return n * (n + 1) * (2*n + 1) / 6
	}
	sum := sumSquaresUpTo(limit) - sumSquaresUpTo(start-1)
	return limit + 1, sum
}

func RunCountAccum(start, limit, step, accumStart, delta int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limit < start {
		return start, accumStart
	}

	n := (limit-start)/step + 1
	return start + n*step, accumStart + n*delta
}

func RunCountAccumMul(start, limit, step, accumStart, factor int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limit < start {
		return start, accumStart
	}
	n := (limit-start)/step + 1
	return start + n*step, accumStart * intPow(factor, n)
}

func RunStepAccum(start, limitExcl, step, accumStart, delta int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limitExcl <= start {
		return start, accumStart
	}
	n := (limitExcl - start + step - 1) / step
	finalI := start + n*step
	finalAccum := accumStart + n*delta
	return finalI, finalAccum
}

func RunCountStep(start, limitExcl, step int64) int64 {
	if step <= 0 {
		step = 1
	}
	if limitExcl <= start {
		return start
	}
	n := (limitExcl - start + step - 1) / step
	return start + n*step
}

func RunFib(a, b, count int64) (int64, int64) {
	for k := int64(0); k < count; k++ {
		a, b = b, a+b
	}
	return a, b
}

func RunCountMul(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 1
	}
	product := int64(1)
	for i := start; i <= limit; i++ {
		product *= i
	}
	return limit + 1, product
}

func intPow(base, exp int64) int64 {
	if exp == 0 {
		return 1
	}
	result := int64(1)
	b := base
	e := exp
	for e > 0 {
		if e&1 == 1 {
			result *= b
		}
		b *= b
		e >>= 1
	}
	return result
}
