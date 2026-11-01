package middleware

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func AddAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	userID int64,
	userUUID uuid.UUID,
	role string,
	duration time.Duration,
) {
	accessToken, payload, err := tokenMaker.CreateToken(username, userID, userUUID, role, duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, accessToken)
	request.Header.Set(AuthorizationHeaderKey, authorizationHeader)
}
