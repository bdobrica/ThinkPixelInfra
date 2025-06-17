package register

import (
	"api_gateway/config"
	"api_gateway/logger"
)

func getUrlProtocol() string {
	insecureValidation := config.GetEnvBool("API_GATEWAY_INSECURE_VALIDATION", false)
	if insecureValidation {
		logger.Warningf("Insecure validation is enabled. Using HTTP protocol. Not recommended for production.")
		return "http"
	}
	return "https"
}
