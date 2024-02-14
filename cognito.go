package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "log"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

// Configuration struct placeholder for AWS Cognito details
type Configuration struct {
    UserPoolRegion string
    ClientID       string
    ClientSecret   string
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

func SignInUser(email, password string, config Configuration) (*cognitoidentityprovider.InitiateAuthOutput, error) {
    sess, err := session.NewSession(&aws.Config{Region: aws.String(config.UserPoolRegion)})
    if err != nil {
        log.Printf("Error creating AWS session for sign-in: %v", err)
        return nil, fmt.Errorf("an internal error occurred while creating AWS session")
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
        log.Printf("Error during user %s sign-in with Cognito: %v", email, err)
        return nil, fmt.Errorf("authentication failed, please check your credentials")
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

func (s *Server) Authenticate(username, password string) bool {
    _, err := SignInUser(username, password, s.Config)
    if err != nil {
        log.Printf("Authentication attempt failed for user %s: %v", username, err)
        return false
    }
    return true
}
