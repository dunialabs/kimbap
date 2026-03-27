package main

import (
	"errors"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
)

type runtimeStoreUnavailableError struct {
	err error
}

func (e *runtimeStoreUnavailableError) Error() string {
	if e == nil || e.err == nil {
		return "runtime store unavailable"
	}
	return e.err.Error()
}

func (e *runtimeStoreUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func isRuntimeStoreUnavailable(err error) bool {
	var target *runtimeStoreUnavailableError
	return errors.As(err, &target)
}

func withRuntimeStore(cfg *config.KimbapConfig, fn func(*store.SQLStore) error) error {
	st, err := openRuntimeStore(cfg)
	if err != nil {
		return &runtimeStoreUnavailableError{err: err}
	}
	defer st.Close()
	return fn(st)
}
