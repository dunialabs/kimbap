package connectors

type ConnectionStatus string

const (
	StatusNotConnected      ConnectionStatus = "not_connected"
	StatusConnecting        ConnectionStatus = "connecting"
	StatusConnected         ConnectionStatus = "connected"
	StatusDegraded          ConnectionStatus = "degraded"
	StatusRefreshFailed     ConnectionStatus = "refresh_failed"
	StatusReconnectRequired ConnectionStatus = "reconnect_required"
	StatusRevoked           ConnectionStatus = "revoked"
	StatusExpired           ConnectionStatus = "expired"
	StatusError             ConnectionStatus = "error"
)

func (s ConnectionStatus) String() string { return string(s) }

func (s ConnectionStatus) NeedsAttention() bool {
	switch s {
	case StatusDegraded, StatusRefreshFailed, StatusReconnectRequired,
		StatusRevoked, StatusExpired, StatusError:
		return true
	default:
		return false
	}
}

func MapLegacyStatus(legacy ConnectorStatus) ConnectionStatus {
	switch legacy {
	case StatusHealthy:
		return StatusConnected
	case StatusExpiring:
		return StatusDegraded
	case StatusOldExpired:
		return StatusExpired
	case StatusReauthNeeded:
		return StatusReconnectRequired
	case StatusPending:
		return StatusConnecting
	default:
		return StatusNotConnected
	}
}
