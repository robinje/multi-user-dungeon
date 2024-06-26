import * as AmazonCognitoIdentity from "amazon-cognito-identity-js";

let config;

// Load the configuration file
fetch('../mud/config.json')
  .then(response => response.json())
  .then(data => {
    config = data;
    initializeCognito();
  })
  .catch(error => console.error('Error loading config:', error));

let userPool;

function initializeCognito() {
  const poolData = {
    UserPoolId: config.UserPoolId,
    ClientId: config.UserPoolClientId,
  };

  userPool = new AmazonCognitoIdentity.CognitoUserPool(poolData);
}

document.getElementById("registrationForm").addEventListener("submit", function (event) {
  event.preventDefault();
  registerPlayer();
});

document.getElementById("confirmationForm").addEventListener("submit", function (event) {
  event.preventDefault();
  confirmRegistration();
});

document.getElementById("passwordResetRequestForm").addEventListener("submit", function (event) {
  event.preventDefault();
  requestPasswordReset();
});

document.getElementById("passwordResetForm").addEventListener("submit", function (event) {
  event.preventDefault();
  resetPassword();
});

function registerPlayer() {
  const username = document.getElementById("username").value;
  const email = document.getElementById("email").value;
  const password = document.getElementById("password").value;

  const attributeList = [
    new AmazonCognitoIdentity.CognitoUserAttribute({ Name: "email", Value: email }),
    new AmazonCognitoIdentity.CognitoUserAttribute({ Name: "preferred_username", Value: username }),
  ];

  userPool.signUp(username, password, attributeList, null, (err, result) => {
    if (err) {
      alert(err.message || JSON.stringify(err));
      return;
    }
    const cognitoUser = result.user;
    console.log("Player registration successful. Username: " + cognitoUser.getUsername());
    document.getElementById("confirmationForm").style.display = "block";
    document.getElementById("registrationForm").style.display = "none";
  });
}

function confirmRegistration() {
  const username = document.getElementById("username").value;
  const confirmationCode = document.getElementById("confirmationCode").value;

  const userData = {
    Username: username,
    Pool: userPool,
  };

  const cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);

  cognitoUser.confirmRegistration(confirmationCode, true, (err, result) => {
    if (err) {
      alert(err.message || JSON.stringify(err));
      return;
    }
    console.log("Registration confirmed. Result: " + result);
    alert("Registration confirmed successfully. You can now log in.");
  });
}

function requestPasswordReset() {
  const username = document.getElementById("resetUsername").value;
  const userData = {
    Username: username,
    Pool: userPool,
  };

  const cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);
  cognitoUser.forgotPassword({
    onSuccess: function () {
      console.log("Password reset request successful");
      document.getElementById("passwordResetForm").style.display = "block";
    },
    onFailure: function (err) {
      alert(err.message || JSON.stringify(err));
    },
  });
}

function resetPassword() {
  const username = document.getElementById("resetUsername").value;
  const verificationCode = document.getElementById("verificationCode").value;
  const newPassword = document.getElementById("newPassword").value;

  const userData = {
    Username: username,
    Pool: userPool,
  };

  const cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);
  cognitoUser.confirmPassword(verificationCode, newPassword, {
    onSuccess() {
      console.log("Password reset successful");
      alert("Password reset successful. You can now log in with your new password.");
    },
    onFailure(err) {
      alert(err.message || JSON.stringify(err));
    },
  });
}