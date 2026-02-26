package elgon

import (
	"strings"
)

type segmentKind uint8

const (
	segmentStatic segmentKind = iota
	segmentParam
	segmentWildcard
)

type routeSegment struct {
	kind  segmentKind
	value string
}

func splitPath(path string) []string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return nil
	}
	return strings.Split(clean, "/")
}

func parseSegments(path string) []routeSegment {
	parts := splitPath(path)
	segs := make([]routeSegment, 0, len(parts))
	for _, p := range parts {
		if strings.HasPrefix(p, ":") && len(p) > 1 {
			segs = append(segs, routeSegment{kind: segmentParam, value: p[1:]})
			continue
		}
		if strings.HasPrefix(p, "*") && len(p) > 1 {
			segs = append(segs, routeSegment{kind: segmentWildcard, value: p[1:]})
			continue
		}
		segs = append(segs, routeSegment{kind: segmentStatic, value: p})
	}
	return segs
}
