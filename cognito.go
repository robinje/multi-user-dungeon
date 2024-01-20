package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

func SignUpUser(email string, password string, config Configuration) (*cognitoidentityprovider.SignUpOutput, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.UserPoolRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}
	cognitoClient := cognitoidentityprovider.New(sess)

	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	input := &cognitoidentityprovider.SignUpInput{
		ClientId:   aws.String(config.ClientID),
		Username:   aws.String(email),
		Password:   aws.String(password),
		SecretHash: aws.String(secretHash),
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
		},
	}
	result, err := cognitoClient.SignUp(input)
	if err != nil {
		return nil, fmt.Errorf("error signing up user: %v", err)
	}
	return result, nil
}

func calculateSecretHash(cognitoAppClientID, clientSecret, email string) string {
	message := []byte(email + cognitoAppClientID)
	key := []byte(clientSecret)
	hash := hmac.New(sha256.New, key)
	hash.Write(message)
	hashedMessage := hash.Sum(nil)
	encodedMessage := base64.StdEncoding.EncodeToString(hashedMessage)
	return encodedMessage
}

func ConfirmUser(email string, userSub string, confirmationCode string, config Configuration) (*cognitoidentityprovider.ConfirmSignUpOutput, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.UserPoolRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}
	cognitoClient := cognitoidentityprovider.New(sess)

	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	input := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(config.ClientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(confirmationCode),
		SecretHash:       aws.String(secretHash),
	}
	result, err := cognitoClient.ConfirmSignUp(input)
	if err != nil {
		return nil, fmt.Errorf("error confirming user: %v", err)
	}
	return result, nil
}

// SignInUser authenticates a user with the given email and password using Cognito User Pool.
func SignInUser(email string, password string, config Configuration) (*cognitoidentityprovider.InitiateAuthOutput, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.UserPoolRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}
	cognitoClient := cognitoidentityprovider.New(sess)

	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	params := map[string]*string{
		"USERNAME":    aws.String(email),
		"PASSWORD":    aws.String(password),
		"SECRET_HASH": aws.String(secretHash),
	}

	authInput := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       aws.String(cognitoidentityprovider.AuthFlowTypeUserPasswordAuth),
		AuthParameters: params,
		ClientId:       aws.String(config.ClientID),
	}

	authOutput, err := cognitoClient.InitiateAuth(authInput)
	if err != nil {
		return nil, fmt.Errorf("error initiating authentication flow: %v", err)
	}

	return authOutput, nil
}

// GetUserData gets the user data from Cognito User Pool.
func GetUserData(cognitoClient *cognitoidentityprovider.CognitoIdentityProvider, accessToken string) (*cognitoidentityprovider.GetUserOutput, error) {
	params := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	}

	userOutput, err := cognitoClient.GetUser(params)
	if err != nil {
		return nil, fmt.Errorf("error getting user data: %v", err)
	}

	return userOutput, nil
}
