package register

import (
	"api_gateway/config"
	"api_gateway/logger"
	"strings"
)

func getUrlProtocol() string {
	insecureValidation := config.GetEnv("API_GATEWAY_INSECURE_VALIDATION", "false")
	insecureValidation = strings.ToLower(insecureValidation)
	if insecureValidation == "true" || insecureValidation == "1" || insecureValidation == "yes" || insecureValidation == "on" {
		logger.Warningf("Insecure validation is enabled. Using HTTP protocol. Not recommended for production.")
		return "http"
	}
	return "https"
}
