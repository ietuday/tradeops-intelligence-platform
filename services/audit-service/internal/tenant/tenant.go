package tenant

import (
	"net/http"
	"strings"
)

const DefaultTenantID = "default-tenant"
const Header = "X-Tenant-ID"

func FromHeader(r *http.Request) string {
	return Normalize(r.Header.Get(Header))
}

func Normalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultTenantID
	}
	return value
}
