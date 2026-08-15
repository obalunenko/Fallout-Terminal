package tunnel

import (
	"context"
	"errors"
	"net"
)

const maximumPublicAccessDiagnosticBytes = 512

type publicAccessCategorizedError struct {
	category ErrorCategory
}

func (failure publicAccessCategorizedError) Error() string {
	return failure.category.SafeMessage()
}

func (failure publicAccessCategorizedError) PublicAccessCategory() ErrorCategory {
	return failure.category
}

type publicAccessErrorCategory interface {
	PublicAccessCategory() ErrorCategory
}

type providerErrorCode interface {
	Code() string
}

func newRedactedPublicAccessError(err error) error {
	if err == nil {
		return nil
	}
	category, _ := redactedPublicAccessFailure(err)
	return publicAccessCategorizedError{category: category}
}

// redactedPublicAccessFailure returns only a stable category and its fixed
// corrective message. Raw SDK, network, account, domain, and credential text
// is neither copied nor retained.
func redactedPublicAccessFailure(err error) (ErrorCategory, string) {
	category := ErrorProviderFailure
	var categorized publicAccessErrorCategory
	var coded providerErrorCode
	var networkError net.Error
	switch {
	case errors.As(err, &categorized) && categorized.PublicAccessCategory().Valid():
		category = categorized.PublicAccessCategory()
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		category = ErrorTimeout
	case errors.Is(err, ErrSecretStoreLocked):
		category = ErrorSecretStoreLocked
	case errors.Is(err, ErrSecretStoreDenied), errors.Is(err, ErrSecretStoreUserCancelled):
		category = ErrorSecretStoreDenied
	case errors.Is(err, ErrSecretStoreUnavailable):
		category = ErrorSecretStoreUnavailable
	case errors.As(err, &coded):
		category = ngrokErrorCategory(coded.Code())
	case errors.As(err, &networkError):
		category = ErrorNetworkUnavailable
	}
	return category, category.SafeMessage()
}

func ngrokErrorCategory(code string) ErrorCategory {
	switch code {
	case "ERR_NGROK_105", "ERR_NGROK_106", "ERR_NGROK_107", "ERR_NGROK_109", "ERR_NGROK_200", "ERR_NGROK_201", "ERR_NGROK_202", "ERR_NGROK_203", "ERR_NGROK_300":
		return ErrorProviderAuthentication
	case "ERR_NGROK_307", "ERR_NGROK_308", "ERR_NGROK_309", "ERR_NGROK_310", "ERR_NGROK_311", "ERR_NGROK_312", "ERR_NGROK_313", "ERR_NGROK_314", "ERR_NGROK_315", "ERR_NGROK_316", "ERR_NGROK_317", "ERR_NGROK_318", "ERR_NGROK_319", "ERR_NGROK_320", "ERR_NGROK_321", "ERR_NGROK_322", "ERR_NGROK_401", "ERR_NGROK_415", "ERR_NGROK_417", "ERR_NGROK_430":
		return ErrorDomainUnavailable
	default:
		return ErrorProviderFailure
	}
}
