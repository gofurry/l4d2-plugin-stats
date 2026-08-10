package service

import "regexp"

var sensitivePatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://[^/\s:@]+:)[^@/\s]+@`), `${1}[REDACTED]@`},
	{regexp.MustCompile(`(?i)([?&](?:password|passwd|pwd|token|secret|key|openid\.[^=&\s]+)=)[^&\s]+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?im)^(\s*(?:dsn|password(?:_hash)?|jwt_secret|setup_token|authorization|cookie|session_secret|client_secret|openid[^:\s]*)\s*:\s*).*$`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)("(?:dsn|password|password_hash|jwt_secret|setup_token|authorization|cookie|session_secret|client_secret|token)"\s*:\s*")[^"]*"`), `${1}[REDACTED]"`},
	{regexp.MustCompile(`(?i)\b(authorization|cookie)\s*:\s*[^\r\n]+`), `${1}: [REDACTED]`},
	{regexp.MustCompile(`(?i)\b(dsn|password|passwd|pwd|password_hash|jwt_secret|setup_token|authorization|cookie|session_secret|client_secret|token)\s*=\s*[^\s,;]+`), `${1}=[REDACTED]`},
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`), `[REDACTED]`},
}

func RedactSensitive(contents []byte) []byte {
	value := string(contents)
	for _, rule := range sensitivePatterns {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	return []byte(value)
}
