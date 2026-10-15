package main

import (
	"fmt"
	"math/rand/v2"
	"os"
)

type luaStage string

func newLuaStage(repo string) (luaStage, error) {
	s := luaStage(fmt.Sprintf("/tmp/gopob-lua-%016x", rand.Uint64())) //nolint:gosec // a name, not a secret
	if err := os.Mkdir(string(s), 0o700); err != nil {
		return "", fmt.Errorf("could not create %s: %s", s, errnoText(err))
	}
	if _, err := s.link("oracle", pyPathJoin(repo, "oracle")); err != nil {
		s.remove()
		return "", err
	}
	return s, nil
}

func (s luaStage) path(name string) string { return pyPathJoin(string(s), name) }

func (s luaStage) link(name, target string) (string, error) {
	p := s.path(name)
	if err := os.Symlink(target, p); err != nil {
		return "", fmt.Errorf("could not link %s: %s", p, errnoText(err))
	}
	return p, nil
}

func (s luaStage) remove() { _ = os.RemoveAll(string(s)) }
