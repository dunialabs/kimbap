package middleware

type contextKey string

const (
	AuthContextKey contextKey = "authContext"
	SessionIDKey   contextKey = "sessionId"
)
