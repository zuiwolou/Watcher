package errcode

const (
	OK                  = "OK"
	AuthDenied          = "AUTH_DENIED"
	ScopeDenied         = "SCOPE_DENIED"
	StateInvalid        = "STATE_INVALID"
	IdempotentReplay    = "IDEMPOTENT_REPLAY"
	ValidationError     = "VALIDATION_ERROR"
	UpstreamUnavailable = "UPSTREAM_UNAVAILABLE"
	LDAPError           = "LDAP_ERROR"
	NotFound            = "NOT_FOUND"
	Internal            = "INTERNAL_ERROR"
)
