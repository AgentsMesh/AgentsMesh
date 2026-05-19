package slugkit

// reserved enumerates slug values that conflict with built-in routes,
// public marketing pages, or system endpoints. Both the Go and TypeScript
// reserved sets MUST stay in sync — see clients/web/src/lib/slug/reserved.ts.
//
// Categories:
//   - Auth flows: auth, login, logout, register, verify-email, ...
//   - Dashboard chrome: settings, support, dashboard, billing
//   - Marketing pages (clients/web/src/app/*): about, blog, careers, ...
//   - API/system paths: api, www, app
//   - Onboarding/special endpoints: onboarding, personal, invite
//   - Polite booby-traps that look like literals: me, new, null, true, false, undefined
var reserved = map[string]struct{}{
	"auth":             {},
	"forgot-password":  {},
	"invite":           {},
	"login":            {},
	"logout":           {},
	"onboarding":       {},
	"register":         {},
	"reset-password":   {},
	"verify-email":     {},
	"admin":     {},
	"billing":   {},
	"dashboard": {},
	"settings":  {},
	"support":   {},
	"about":         {},
	"blog":          {},
	"careers":       {},
	"changelog":     {},
	"demo":          {},
	"docs":          {},
	"enterprise":    {},
	"mock-checkout": {},
	"offline":       {},
	"popout":        {},
	"privacy":       {},
	"terms":         {},
	"api": {},
	"app": {},
	"www": {},
	"organizations": {},
	"orgs":          {},
	"personal":      {},
	"runners":       {},
	"agents":        {},
	"me":        {},
	"new":       {},
	"null":      {},
	"true":      {},
	"false":     {},
	"undefined": {},
}

func IsReserved(s string) bool {
	_, ok := reserved[s]
	return ok
}

func ReservedList() []string {
	out := make([]string, 0, len(reserved))
	for k := range reserved {
		out = append(out, k)
	}
	return out
}
