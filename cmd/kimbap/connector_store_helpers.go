package main

import "github.com/dunialabs/kimbap/internal/connectors"

type connectorStoreCloser interface {
	connectors.ConnectorStore
	Close() error
}

func closeConnectorStoreIfPossible(store connectors.ConnectorStore) {
	if closer, ok := store.(connectorStoreCloser); ok {
		_ = closer.Close()
	}
}
