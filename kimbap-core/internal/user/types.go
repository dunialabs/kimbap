package user

type UserActionType int

const (
	GetCapabilities   UserActionType = 1001
	SetCapabilities   UserActionType = 1002
	ConfigureServer   UserActionType = 2001
	UnconfigureServer UserActionType = 2002
	GetOnlineSessions UserActionType = 3001
)

type UserRequest struct {
	Action UserActionType `json:"action"`
	Data   any            `json:"data,omitempty"`
}

type UserResponse struct {
	Success bool         `json:"success"`
	Data    any          `json:"data,omitempty"`
	Error   *UserRespErr `json:"error,omitempty"`
}

type UserRespErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	UserErrorInvalidRequest      = 1001
	UserErrorUnauthorized        = 1002
	UserErrorServerNotFound      = 2001
	UserErrorServerDisabled      = 2002
	UserErrorServerConfigInvalid = 2003
	UserErrorServerNoUserInput   = 2004
	UserErrorServerNoTemplate    = 2005
	UserErrorInternal            = 5001
)

type UserError struct {
	Message string
	Code    int
}

func (e *UserError) Error() string {
	return e.Message
}
