package domain

import "strings"

const DashScopeBaseURL = "https://dashscope-intl.aliyuncs.com/api/v2/apps/protocols/compatible-mode/v1/responses"

func ResolveAPIKey(flagValue string, env map[string]string) (apiKey string, source string, err error) {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue), "flag", nil
	}
	if value := strings.TrimSpace(env["DASHSCOPE_API_KEY"]); value != "" {
		return value, "env", nil
	}
	return "", "", ErrMissingAPIKey
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "***"
	}
	return value[:3] + "..." + value[len(value)-3:]
}
