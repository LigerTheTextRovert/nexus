package config

// Validate port is set and valid.
// Validate each route has non-empty path.
// Enforce normalized path format (starts with /, no trailing slash except /).
// Validate backend_URL is parseable and has http/https scheme + host.
// Detect duplicate route paths.
// Return clear, actionable startup errors.

func PortValidator(port any) bool {
	p, ok := port.(int)
	if !ok {
		return false
	}

	if p < 1 || p > 65535 {
		return false
	}

	return true
}
