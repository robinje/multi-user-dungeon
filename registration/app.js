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

function registerPlayer() {
  const email = document.getElementById("email").value;
  const password = document.getElementById("password").value;
  const givenName = document.getElementById("givenName").value;
  const familyName = document.getElementById("familyName").value;

  const attributeList = [
    new AmazonCognitoIdentity.CognitoUserAttribute({ Name: "email", Value: email }),
    new AmazonCognitoIdentity.CognitoUserAttribute({ Name: "given_name", Value: givenName }),
    new AmazonCognitoIdentity.CognitoUserAttribute({ Name: "family_name", Value: familyName })
  ];

  userPool.signUp(email, password, attributeList, null, (err, result) => {
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
  const email = document.getElementById("email").value;
  const confirmationCode = document.getElementById("confirmationCode").value;

  const userData = {
    Username: email,
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

// Add password validation function
function validatePassword(password) {
  const minLength = 8;
  const hasLowerCase = /[a-z]/.test(password);
  const hasUpperCase = /[A-Z]/.test(password);
  const hasNumber = /\d/.test(password);
  const hasSymbol = /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password);

  return password.length >= minLength && hasLowerCase && hasUpperCase && hasNumber && hasSymbol;
}

// Add event listener for password input
document.getElementById("password").addEventListener("input", function (event) {
  const password = event.target.value;
  const isValid = validatePassword(password);
  
  const feedbackElement = document.getElementById("passwordFeedback");
  if (isValid) {
    feedbackElement.textContent = "Password meets requirements";
    feedbackElement.style.color = "green";
  } else {
    feedbackElement.textContent = "Password must be at least 8 characters long and include lowercase, uppercase, number, and symbol";
    feedbackElement.style.color = "red";
  }
});