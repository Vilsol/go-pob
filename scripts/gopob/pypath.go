package main

import (
	"os"
	"os/user"
	"strings"
)

func pyPathJoin(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		switch {
		case strings.HasPrefix(p, "/"):
			out = p
		case out == "" || strings.HasSuffix(out, "/"):
			out += p
		default:
			out += "/" + p
		}
	}
	return out
}

func pyUserHome() (string, bool) {
	if v, ok := os.LookupEnv("HOME"); ok {
		return v, true
	}
	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		return "", false
	}
	return u.HomeDir, true
}

func pyExpandUser(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	i := strings.Index(p, "/")
	if i < 0 {
		i = len(p)
	}
	var home string
	if i == 1 {
		h, ok := pyUserHome()
		if !ok {
			return p
		}
		home = h
	} else {
		u, err := user.Lookup(p[1:i])
		if err != nil {
			return p
		}
		home = u.HomeDir
	}
	home = strings.TrimRight(home, "/")
	if r := home + p[i:]; r != "" {
		return r
	}
	return "/"
}

func pyPathHome() (string, bool) {
	h, ok := pyUserHome()
	if !ok {
		return "", false
	}
	h = strings.TrimRight(h, "/")
	if h == "" {
		return "/", true
	}
	return h, true
}

func pyPathStr(p string) string {
	if p == "" {
		return "."
	}
	n := 0
	for n < len(p) && p[n] == '/' {
		n++
	}
	root := ""
	switch {
	case n == 2:
		root = "//"
	case n > 0:
		root = "/"
	}
	parts := make([]string, 0, 8)
	for _, c := range strings.Split(p[n:], "/") {
		if c != "" && c != "." {
			parts = append(parts, c)
		}
	}
	if len(parts) == 0 {
		if root != "" {
			return root
		}
		return "."
	}
	return root + strings.Join(parts, "/")
}

func pyPathName(p string) string {
	s := pyPathStr(p)
	if s == "." || strings.Trim(s, "/") == "" {
		return ""
	}
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}
