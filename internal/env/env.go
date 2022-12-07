package env

import (
	"os"
)

type Env interface {
	Get(key string) string
}

func New() Env {
	return new(osEnv)
}

type osEnv struct{}

func (e *osEnv) Get(key string) string {
	return os.Getenv(key)
}

type Stubs map[string]string

func NewWithStubs(stubs Stubs) Env {
	return &stubbedEnv{
		stubs: stubs,
	}
}

type stubbedEnv struct {
	stubs Stubs
}

func (e *stubbedEnv) Get(key string) string {
	found, ok := e.stubs[key]
	if !ok {
		return ""
	}

	return found
}
