package main

import "strings"

func didYouMean(query string, candidates []string) string {
	query = strings.ToLower(query)
	best := ""
	bestDist := len(query)/2 + 2
	for _, c := range candidates {
		lower := strings.ToLower(c)
		if strings.HasPrefix(lower, query) || strings.HasPrefix(query, lower) {
			return c
		}
		d := editDistance(query, lower)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

func editDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	if len(a) > 32 || len(b) > 32 {
		return len(a) + len(b)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
