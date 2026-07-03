package semverlite

import (
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
}

func ParseVersion(s string) (Version, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	v := Version{}
	nums := make([]int, 0, 3)
	for _, p := range parts {
		if p == "" || p == "x" || p == "X" || p == "*" {
			break
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, false
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return Version{}, false
	}
	v.Major = nums[0]
	if len(nums) > 1 {
		v.Minor = nums[1]
	}
	if len(nums) > 2 {
		v.Patch = nums[2]
	}
	return v, true
}

func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		return cmp(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return cmp(v.Minor, o.Minor)
	}
	return cmp(v.Patch, o.Patch)
}

func cmp(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}

type clause struct {
	op  string
	ver Version
	// wildcard: true for truncated forms like "8.2" or "20.x", comparing only sigParts components
	wildcard bool
	sigParts int
}

func parseClause(term string) (clause, bool) {
	term = strings.TrimSpace(term)
	if term == "" {
		return clause{}, false
	}
	op := "="
	switch {
	case strings.HasPrefix(term, "^"):
		op, term = "^", term[1:]
	case strings.HasPrefix(term, "~"):
		op, term = "~", term[1:]
	case strings.HasPrefix(term, ">="):
		op, term = ">=", term[2:]
	case strings.HasPrefix(term, "<="):
		op, term = "<=", term[2:]
	case strings.HasPrefix(term, ">"):
		op, term = ">", term[1:]
	case strings.HasPrefix(term, "<"):
		op, term = "<", term[1:]
	case strings.HasPrefix(term, "="):
		op, term = "=", term[1:]
	}
	term = strings.TrimSpace(term)
	parts := strings.Split(term, ".")
	sigParts := len(parts)
	for i, p := range parts {
		if p == "x" || p == "X" || p == "*" || p == "" {
			sigParts = i
			break
		}
	}
	ver, ok := ParseVersion(term)
	if !ok {
		return clause{}, false
	}
	return clause{op: op, ver: ver, wildcard: len(parts) < 3 || sigParts < len(parts), sigParts: sigParts}, true
}

func (c clause) satisfies(v Version) bool {
	switch c.op {
	case "^":
		if c.ver.Major != v.Major {
			return false
		}
		return v.Compare(c.ver) >= 0
	case "~":
		if c.ver.Major != v.Major || c.ver.Minor != v.Minor {
			return false
		}
		return v.Compare(c.ver) >= 0
	case ">=":
		return v.Compare(c.ver) >= 0
	case "<=":
		return v.Compare(c.ver) <= 0
	case ">":
		return v.Compare(c.ver) > 0
	case "<":
		return v.Compare(c.ver) < 0
	default: // "="
		if c.wildcard {
			if c.sigParts >= 1 && v.Major != c.ver.Major {
				return false
			}
			if c.sigParts >= 2 && v.Minor != c.ver.Minor {
				return false
			}
			return true
		}
		return v.Compare(c.ver) == 0
	}
}

// constraint combines terms with spaces for AND, "|" or "||" for OR, e.g. "^8.1|^8.2".
func Resolve(constraint string, available []string) (string, bool) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return "", false
	}
	orGroups := strings.FieldsFunc(constraint, func(r rune) bool { return r == '|' })

	var candidates []Version
	byString := map[Version]string{}
	for _, raw := range available {
		v, ok := ParseVersion(raw)
		if !ok {
			continue
		}
		candidates = append(candidates, v)
		byString[v] = raw
	}
	if len(candidates) == 0 {
		return "", false
	}

	var best *Version
	for _, group := range orGroups {
		terms := strings.Fields(group)
		if len(terms) == 0 {
			continue
		}
		clauses := make([]clause, 0, len(terms))
		for _, t := range terms {
			c, ok := parseClause(t)
			if !ok {
				clauses = nil
				break
			}
			clauses = append(clauses, c)
		}
		if len(clauses) == 0 {
			continue
		}
		for _, v := range candidates {
			ok := true
			for _, c := range clauses {
				if !c.satisfies(v) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			if best == nil || v.Compare(*best) > 0 {
				vCopy := v
				best = &vCopy
			}
		}
	}
	if best == nil {
		return "", false
	}
	return byString[*best], true
}
