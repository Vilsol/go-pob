package main

import "sort"

type pyCounter struct {
	order  []string
	counts map[string]int
}

func newPyCounter() *pyCounter {
	return &pyCounter{counts: map[string]int{}}
}

func (c *pyCounter) add(key string) {
	if _, seen := c.counts[key]; !seen {
		c.order = append(c.order, key)
	}
	c.counts[key]++
}

func (c *pyCounter) total() int {
	n := 0
	for _, v := range c.counts {
		n += v
	}
	return n
}

func (c *pyCounter) mostCommon() []string {
	out := append([]string(nil), c.order...)
	sort.SliceStable(out, func(a, b int) bool { return c.counts[out[a]] > c.counts[out[b]] })
	return out
}

type pyListMap struct{ m map[string][]string }

func newPyListMap() *pyListMap { return &pyListMap{m: map[string][]string{}} }

func (d *pyListMap) add(key, value string) { d.m[key] = append(d.m[key], value) }

func (d *pyListMap) head(key string, n int) []string {
	v := d.m[key]
	if len(v) > n {
		v = v[:n]
	}
	return v
}
