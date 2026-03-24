package oauth

import "fmt"

type OAuthExchangeErrorType string

const (
	OAuthExchangeErrorHTTP            OAuthExchangeErrorType = "http"
	OAuthExchangeErrorParse           OAuthExchangeErrorType = "parse"
	OAuthExchangeErrorUnknownProvider OAuthExchangeErrorType = "unknown_provider"
)

type OAuthExchangeError struct {
	Message      string
	Type         OAuthExchangeErrorType
	Provider     string
	Status       int
	ResponseBody string
	Cause        error
}

func (e *OAuthExchangeError) Error() string {
	return e.Message
}

func NewOAuthExchangeError(message string, errType OAuthExchangeErrorType, provider string, status int, responseBody string, cause error) *OAuthExchangeError {
	return &OAuthExchangeError{
		Message:      message,
		Type:         errType,
		Provider:     provider,
		Status:       status,
		ResponseBody: responseBody,
		Cause:        cause,
	}
}

func NewOAuthHTTPError(provider string, status int, responseBody string) *OAuthExchangeError {
	return NewOAuthExchangeError(
		fmt.Sprintf("OAuth token exchange failed for %s: HTTP %d", provider, status),
		OAuthExchangeErrorHTTP,
		provider,
		status,
		responseBody,
		nil,
	)
}

func NewOAuthParseError(provider string, responseBody string, cause error) *OAuthExchangeError {
	return NewOAuthExchangeError(
		fmt.Sprintf("Failed to parse OAuth response for %s", provider),
		OAuthExchangeErrorParse,
		provider,
		0,
		responseBody,
		cause,
	)
}

func NewOAuthUnknownProviderError(provider string) *OAuthExchangeError {
	return NewOAuthExchangeError(
		fmt.Sprintf("Unknown OAuth provider: %s", provider),
		OAuthExchangeErrorUnknownProvider,
		provider,
		0,
		"",
		nil,
	)
}
