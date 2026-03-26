package types

const (
	AdminActionDisableUser                         = 1001
	AdminActionUpdateUserPermissions               = 1002
	AdminActionCreateUser                          = 1010
	AdminActionGetUsers                            = 1011
	AdminActionUpdateUser                          = 1012
	AdminActionDeleteUser                          = 1013
	AdminActionDeleteUsersByProxy                  = 1014
	AdminActionCountUsers                          = 1015
	AdminActionGetOwner                            = 1016
	AdminActionStartServer                         = 2001
	AdminActionStopServer                          = 2002
	AdminActionUpdateServerCapabilities            = 2003
	AdminActionUpdateServerLaunchCmd               = 2004
	AdminActionConnectAllServers                   = 2005
	AdminActionCreateServer                        = 2010
	AdminActionGetServers                          = 2011
	AdminActionUpdateServer                        = 2012
	AdminActionDeleteServer                        = 2013
	AdminActionDeleteServersByProxy                = 2014
	AdminActionCountServers                        = 2015
	AdminActionGetAvailableServersCapabilities     = 3002
	AdminActionGetUserAvailableServersCapabilities = 3003
	AdminActionGetServersStatus                    = 3004
	AdminActionGetServersCapabilities              = 3005
	AdminActionGetProxy                            = 5001
	AdminActionCreateProxy                         = 5002
	AdminActionUpdateProxy                         = 5003
	AdminActionDeleteProxy                         = 5004
	AdminActionStopProxy                           = 5005
	AdminActionSetLogWebhookURL                    = 7001
	AdminActionGetLogs                             = 7002
	AdminActionCreatePolicySet                     = 9101
	AdminActionGetPolicySets                       = 9102
	AdminActionUpdatePolicySet                     = 9103
	AdminActionDeletePolicySet                     = 9104
	AdminActionGetEffectivePolicy                  = 9105
	AdminActionListApprovalRequests                = 9201
	AdminActionGetApprovalRequest                  = 9202
	AdminActionDecideApprovalRequest               = 9203
	AdminActionGetPendingApprovalsCount            = 9204
	AdminActionListServices                        = 10040
	AdminActionUploadService                       = 10041
	AdminActionDeleteService                       = 10042
	AdminActionDeleteServerServices                = 10043
)

type AdminRequest struct {
	Action int `json:"action"`
	Data   any `json:"data"`
}

type AdminResponse struct {
	Success bool                `json:"success"`
	Data    any                 `json:"data,omitempty"`
	Error   *AdminResponseError `json:"error,omitempty"`
}

type AdminResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	AdminErrorCodeInvalidRequest           = 1001
	AdminErrorCodeUnauthorized             = 1002
	AdminErrorCodeForbidden                = 1003
	AdminErrorCodeUserNotFound             = 2001
	AdminErrorCodeUserAlreadyExists        = 2003
	AdminErrorCodeServerNotFound           = 3001
	AdminErrorCodeServerAlreadyExists      = 3003
	AdminErrorCodeProxyNotFound            = 5001
	AdminErrorCodeProxyAlreadyExists       = 5002
	AdminErrorCodeDatabaseOpFailed         = 5201
	AdminErrorCodeInvalidCredentialsFormat = 8002
	AdminErrorCodeServiceNotFound          = 9001
	AdminErrorCodeServiceUploadFailed      = 9002
	AdminErrorCodeServiceDeleteFailed      = 9003
	AdminErrorCodeInvalidServiceFormat     = 9004
)

type AdminError struct {
	Message string
	Code    int
}

func (e *AdminError) Error() string {
	return e.Message
}
