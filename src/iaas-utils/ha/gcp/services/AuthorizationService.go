package services

import (
	"iaas-utils/ha/common/models"
	"golang.org/x/net/context"
	gcpjwt "golang.org/x/oauth2/jwt"
	"log"
	"strings"
)

type AuthorizationService struct {
	iaasDescriptors models.IaaSDescriptors
}

func (authService *AuthorizationService) Initialize(initParams ...interface{}) {
}

func (authService *AuthorizationService) Authorize(authorizationRequest models.AuthorizationRequest) (*models.AuthorizationToken, bool) {

	var authorizationToken = &models.AuthorizationToken{}
	var privateKeyBytes []byte

	authorizationRequest.PrivateKey = strings.Replace(authorizationRequest.PrivateKey, `\n`, "\n", -1)
	privateKeyBytes = []byte(authorizationRequest.PrivateKey)
	jwtConfig := &gcpjwt.Config{
		Email:        authorizationRequest.ClientEmailId,
		PrivateKey:   privateKeyBytes,
		PrivateKeyID: authorizationRequest.PrivateKeyId,
		Scopes:       authorizationRequest.Scopes,
		TokenURL:     authorizationRequest.AuthBaseURL,
	}

	authToken, err := jwtConfig.TokenSource(context.Background()).Token()
	if err != nil {
		log.Println("Authorization Request failed with error : ", err)
		return nil, false
	}
	if !authToken.Valid() {
		log.Println("Authorization Request failed - Authorization token retrieved is invalid")
		return nil, false
	}

	log.Println("AccessToken: |", authToken.AccessToken, "| TokenType: |", authToken.TokenType, "|")
	log.Println("RefreshToken: |", authToken.RefreshToken, "| Expiry: |", authToken.Expiry, "|")

	authorizationToken.AccessKey = authToken.AccessToken
	authorizationToken.TokenType = authToken.TokenType
	// NOTE: Expiry is not used as calls would be completed in 10-15 min and token re-use scenario is absent.

	return authorizationToken, true

}
