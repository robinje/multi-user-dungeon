package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

func calculateSecretHash(cognitoAppClientID, clientSecret, email string) string {
	message := []byte(email + cognitoAppClientID)
	key := []byte(clientSecret)
	hash := hmac.New(sha256.New, key)
	hash.Write(message)
	hashedMessage := hash.Sum(nil)
	encodedMessage := base64.StdEncoding.EncodeToString(hashedMessage)
	return encodedMessage
}

func handleCognitoError(err error, email string) error {
	if awsErr, ok := err.(awserr.Error); ok {
		switch awsErr.Code() {
		case cognitoidentityprovider.ErrCodeNotAuthorizedException:
			return fmt.Errorf("incorrect username or password")
		case cognitoidentityprovider.ErrCodeUserNotConfirmedException:
			return fmt.Errorf("user is not confirmed")
		case cognitoidentityprovider.ErrCodePasswordResetRequiredException:
			return fmt.Errorf("password reset required")
		default:
			return fmt.Errorf("authentication failed for user %s: %w", email, awsErr)
		}
	}
	return fmt.Errorf("unexpected error during authentication for user %s: %w", email, err)
}

// SignInUser attempts to sign in a user with the provided credentials
func SignInUser(email, password string, config Configuration) (*cognitoidentityprovider.InitiateAuthOutput, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(config.UserPoolRegion)})
	if err != nil {
		return nil, fmt.Errorf("create AWS session: %w", err)
	}

	cognitoClient := cognitoidentityprovider.New(sess)
	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	authInput := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String(cognitoidentityprovider.AuthFlowTypeUserPasswordAuth),
		AuthParameters: map[string]*string{
			"USERNAME":    aws.String(email),
			"PASSWORD":    aws.String(password),
			"SECRET_HASH": aws.String(secretHash),
		},
		ClientId: aws.String(config.ClientID),
	}

	authOutput, err := cognitoClient.InitiateAuth(authInput)
	if err != nil {
		return nil, handleCognitoError(err, email)
	}

	if authOutput.AuthenticationResult == nil {
		return authOutput, fmt.Errorf("authentication successful for user %s, but no AuthenticationResult returned", email)
	}

	return authOutput, nil
}

func SignUpUser(email, password string, config Configuration) (*cognitoidentityprovider.SignUpOutput, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(config.UserPoolRegion)})
	if err != nil {
		log.Printf("Error creating AWS session for sign-up: %v", err)
		return nil, fmt.Errorf("an internal error occurred while creating AWS session")
	}

	cognitoClient := cognitoidentityprovider.New(sess)
	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	signUpInput := &cognitoidentityprovider.SignUpInput{
		ClientId:   aws.String(config.ClientID),
		Username:   aws.String(email),
		Password:   aws.String(password),
		SecretHash: aws.String(secretHash),
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
	}

	signUpOutput, err := cognitoClient.SignUp(signUpInput)
	if err != nil {
		log.Printf("Error signing up user %s with Cognito: %v", email, err)
		return nil, fmt.Errorf("error signing up, please try again")
	}

	return signUpOutput, nil
}

func ConfirmUser(email, confirmationCode string, config Configuration) (*cognitoidentityprovider.ConfirmSignUpOutput, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(config.UserPoolRegion)})
	if err != nil {
		log.Printf("Error creating AWS session for user confirmation: %v", err)
		return nil, fmt.Errorf("an internal error occurred while creating AWS session")
	}

	cognitoClient := cognitoidentityprovider.New(sess)
	secretHash := calculateSecretHash(config.ClientID, config.ClientSecret, email)

	confirmSignUpInput := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(config.ClientID),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(confirmationCode),
		SecretHash:       aws.String(secretHash),
	}

	confirmSignUpOutput, err := cognitoClient.ConfirmSignUp(confirmSignUpInput)
	if err != nil {
		log.Printf("Error confirming sign up for user %s: %v", email, err)
		return nil, fmt.Errorf("error confirming sign up, please check your code and try again")
	}

	return confirmSignUpOutput, nil
}

func GetUserData(accessToken string, config Configuration) (*cognitoidentityprovider.GetUserOutput, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(config.UserPoolRegion)})
	if err != nil {
		log.Printf("Error creating AWS session for getting user data: %v", err)
		return nil, fmt.Errorf("an internal error occurred while creating AWS session")
	}

	cognitoClient := cognitoidentityprovider.New(sess)

	getUserInput := &cognitoidentityprovider.GetUserInput{AccessToken: aws.String(accessToken)}
	userOutput, err := cognitoClient.GetUser(getUserInput)
	if err != nil {
		log.Printf("Error getting user data with access token: %v", err)
		return nil, fmt.Errorf("error retrieving user data, please try again")
	}

	return userOutput, nil
}

func ChangePassword(server *Server, username, oldPassword, newPassword string) error {
	log.Printf("Attempting to change password for user: %s", username)

	// Step 1: Authenticate the user
	log.Printf("Step 1: Authenticating user %s", username)
	signInOutput, err := SignInUser(username, oldPassword, server.Config)
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", username, err)
		return fmt.Errorf("authentication failed: %v", err)
	}
	log.Printf("Authentication successful for user %s", username)
	log.Printf("SignInOutput for user %s: %+v", username, signInOutput)

	// Step 2: Handle NEW_PASSWORD_REQUIRED challenge if present
	if signInOutput.ChallengeName != nil && *signInOutput.ChallengeName == cognitoidentityprovider.ChallengeNameTypeNewPasswordRequired {
		log.Printf("NEW_PASSWORD_REQUIRED challenge detected for user %s", username)

		// Create a new AWS session
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(server.Config.UserPoolRegion),
		})
		if err != nil {
			log.Printf("Failed to create AWS session for user %s: %v", username, err)
			return fmt.Errorf("failed to create AWS session: %v", err)
		}

		// Create Cognito Identity Provider client
		cognitoClient := cognitoidentityprovider.New(sess)

		// Calculate SECRET_HASH
		secretHash := calculateSecretHash(server.Config.ClientID, server.Config.ClientSecret, username)

		// Respond to the NEW_PASSWORD_REQUIRED challenge
		challengeResponseInput := &cognitoidentityprovider.RespondToAuthChallengeInput{
			ChallengeName: aws.String(cognitoidentityprovider.ChallengeNameTypeNewPasswordRequired),
			ClientId:      aws.String(server.Config.ClientID),
			ChallengeResponses: map[string]*string{
				"USERNAME":     aws.String(username),
				"NEW_PASSWORD": aws.String(newPassword),
				"SECRET_HASH":  aws.String(secretHash),
			},
			Session: signInOutput.Session,
		}

		log.Printf("Sending challenge response for user %s", username)
		challengeResponse, err := cognitoClient.RespondToAuthChallenge(challengeResponseInput)
		if err != nil {
			log.Printf("Failed to respond to NEW_PASSWORD_REQUIRED challenge for user %s: %v", username, err)
			return fmt.Errorf("failed to set new password: %v", err)
		}

		log.Printf("Successfully responded to NEW_PASSWORD_REQUIRED challenge for user %s", username)
		log.Printf("Challenge response: %+v", challengeResponse)

		// Password has been changed successfully
		return nil
	}

	// If we're here, it means there was no NEW_PASSWORD_REQUIRED challenge
	// In this case, we need to use the ChangePassword API as before

	if signInOutput.AuthenticationResult == nil || signInOutput.AuthenticationResult.AccessToken == nil {
		log.Printf("No valid access token for user %s", username)
		return fmt.Errorf("no valid access token available")
	}

	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(server.Config.UserPoolRegion),
	})
	if err != nil {
		log.Printf("Failed to create AWS session for user %s: %v", username, err)
		return fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create Cognito Identity Provider client
	cognitoClient := cognitoidentityprovider.New(sess)

	// Perform the change password operation
	input := &cognitoidentityprovider.ChangePasswordInput{
		PreviousPassword: aws.String(oldPassword),
		ProposedPassword: aws.String(newPassword),
		AccessToken:      signInOutput.AuthenticationResult.AccessToken,
	}

	_, err = cognitoClient.ChangePassword(input)
	if err != nil {
		log.Printf("Failed to change password for user %s: %v", username, err)
		return fmt.Errorf("failed to change password: %v", err)
	}

	log.Printf("Password successfully changed for user %s", username)
	return nil
}
