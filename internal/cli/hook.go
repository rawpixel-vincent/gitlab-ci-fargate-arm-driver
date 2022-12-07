package cli

import (
	"fmt"
)

type Hook func(ctx *Context) error

type Hooks []Hook

func (h Hooks) Execute(cliCtx *Context) error {
	for _, f := range h {
		err := f(cliCtx)
		if err != nil {
			return fmt.Errorf("hook failure: %w", err)
		}
	}

	return nil
}

func NewNopHook() Hook {
	return func(ctx *Context) error { return nil }
}
