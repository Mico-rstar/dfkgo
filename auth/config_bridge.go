package auth

import "dfkgo/config"

func getJwtKeyFromConfig() string {
	cfg := config.GetConfig()
	if cfg.JwtPriKey == "" {
		panic("JWT_PRIVATE_KEY is not set")
	}
	return cfg.JwtPriKey
}
