package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lerko/helm/internal/config"
)

func Login(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if subtle.ConstantTimeCompare([]byte(req.Password), []byte(cfg.Auth.Password)) != 1 {
			respondError(w, http.StatusUnauthorized, "invalid password")
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": defaultUserID,
			"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
		})

		signed, err := token.SignedString([]byte(cfg.Auth.Secret))
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to sign token")
			return
		}

		respond(w, http.StatusOK, map[string]string{"token": signed})
	}
}
