package semver

import (
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Raw                 string
}

func Parse(s string) (Version, bool) {
	raw := s
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return Version{}, false
	}
	parts := strings.SplitN(s, "-", 2)
	core := parts[0]
	nums := strings.Split(core, ".")
	if len(nums) == 0 || len(nums) > 3 {
		return Version{}, false
	}
	get := func(i int) (int, bool) {
		if i >= len(nums) {
			return 0, true
		}
		n, err := strconv.Atoi(nums[i])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	major, ok := get(0)
	if !ok {
		return Version{}, false
	}
	minor, ok := get(1)
	if !ok {
		return Version{}, false
	}
	patch, ok := get(2)
	if !ok {
		return Version{}, false
	}
	return Version{Major: major, Minor: minor, Patch: patch, Raw: raw}, true
}

func Compare(a, b Version) int {
	if a.Major != b.Major {
		return sign(a.Major - b.Major)
	}
	if a.Minor != b.Minor {
		return sign(a.Minor - b.Minor)
	}
	if a.Patch != b.Patch {
		return sign(a.Patch - b.Patch)
	}
	return 0
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

type Constraint struct {
	Raw      string
	IsLatest bool
	Exact    *Version
	clauses  []clause
}

type clauseOp int

const (
	opGTE clauseOp = iota
	opGT
	opLTE
	opLT
	opEQ
)

type clause struct {
	op  clauseOp
	ver Version
}

func ParseConstraint(s string) Constraint {
	s = strings.TrimSpace(s)
	c := Constraint{Raw: s}
	if s == "" || strings.EqualFold(s, "latest") {
		c.IsLatest = true
		return c
	}

	if strings.Contains(s, "x") || strings.Contains(s, "X") || strings.Contains(s, "*") {
		if lo, hi, ok := xRangeBounds(s); ok {
			c.clauses = append(c.clauses, clause{op: opGTE, ver: lo})
			c.clauses = append(c.clauses, clause{op: opLT, ver: hi})
			return c
		}
	}

	if strings.HasPrefix(s, "^") {
		v, ok := Parse(strings.TrimPrefix(s, "^"))
		if ok {
			upper := Version{Major: v.Major + 1}
			c.clauses = append(c.clauses, clause{op: opGTE, ver: v})
			c.clauses = append(c.clauses, clause{op: opLT, ver: upper})
			return c
		}
	}

	fields := strings.Fields(s)
	if len(fields) > 1 {
		for _, f := range fields {
			if cl, ok := parseComparator(f); ok {
				c.clauses = append(c.clauses, cl)
			}
		}
		if len(c.clauses) > 0 {
			return c
		}
	}
	if len(fields) == 1 {
		if cl, ok := parseComparator(fields[0]); ok {
			c.clauses = append(c.clauses, cl)
			return c
		}
	}

	if v, ok := Parse(s); ok {
		vv := v
		c.Exact = &vv
	}
	return c
}

func parseComparator(f string) (clause, bool) {
	ops := []struct {
		prefix string
		op     clauseOp
	}{
		{">=", opGTE},
		{"<=", opLTE},
		{">", opGT},
		{"<", opLT},
		{"=", opEQ},
	}
	for _, o := range ops {
		if strings.HasPrefix(f, o.prefix) {
			v, ok := Parse(strings.TrimPrefix(f, o.prefix))
			if !ok {
				return clause{}, false
			}
			return clause{op: o.op, ver: v}, true
		}
	}
	return clause{}, false
}

func xRangeBounds(s string) (Version, Version, bool) {
	parts := strings.Split(s, ".")
	nums := []int{}
	for _, p := range parts {
		if p == "x" || p == "X" || p == "*" {
			break
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, Version{}, false
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return Version{}, Version{}, false
	}
	lo := Version{}
	if len(nums) >= 1 {
		lo.Major = nums[0]
	}
	if len(nums) >= 2 {
		lo.Minor = nums[1]
	}
	if len(nums) >= 3 {
		lo.Patch = nums[2]
	}
	hi := lo
	switch len(nums) {
	case 1:
		hi = Version{Major: lo.Major + 1}
	case 2:
		hi = Version{Major: lo.Major, Minor: lo.Minor + 1}
	case 3:
		hi = Version{Major: lo.Major, Minor: lo.Minor, Patch: lo.Patch + 1}
	}
	return lo, hi, true
}

func (c Constraint) Matches(v Version) bool {
	if c.IsLatest {
		return true
	}
	if c.Exact != nil {
		return Compare(v, *c.Exact) == 0
	}
	for _, cl := range c.clauses {
		cmp := Compare(v, cl.ver)
		switch cl.op {
		case opGTE:
			if cmp < 0 {
				return false
			}
		case opGT:
			if cmp <= 0 {
				return false
			}
		case opLTE:
			if cmp > 0 {
				return false
			}
		case opLT:
			if cmp >= 0 {
				return false
			}
		case opEQ:
			if cmp != 0 {
				return false
			}
		}
	}
	return true
}

func (c Constraint) Best(candidates []Version) (Version, bool) {
	var best *Version
	for i := range candidates {
		v := candidates[i]
		if !c.Matches(v) {
			continue
		}
		if best == nil || Compare(v, *best) > 0 {
			vv := v
			best = &vv
		}
	}
	if best == nil {
		return Version{}, false
	}
	return *best, true
}
