import { CognitoUserPool, CognitoUser} from 'amazon-cognito-identity-js';

// Configuration for Cognito User Pool
const poolData = {
  UserPoolId: 'YOUR_USER_POOL_ID',  // Replace with your Cognito User Pool ID
  ClientId: 'YOUR_CLIENT_ID'  // Replace with your Cognito Client ID
};

const userPool = new CognitoUserPool(poolData);

// Registration Function
document.getElementById('registrationForm').addEventListener('submit', function(event) {
  event.preventDefault();
  
  const email = document.getElementById('email').value;
  const password = document.getElementById('password').value;
  const confirmPassword = document.getElementById('confirmPassword').value;
  
  if (password !== confirmPassword) {
    alert("Passwords do not match.");
    return;
  }
  
  const attributeList = [];
  attributeList.push(new CognitoUserAttribute({ Name: "email", Value: email }));
  
  userPool.signUp(email, password, attributeList, null, function(err, result) {
    if (err) {
      alert(err.message || JSON.stringify(err));
      return;
    }
    alert("Registration successful. Please check your email for verification.");
  });
});

// Password Reset Function
document.getElementById('resetPasswordForm').addEventListener('submit', function(event) {
  event.preventDefault();
  
  const email = document.getElementById('resetEmail').value;
  const userData = {
    Username: email,
    Pool: userPool
  };
  
  const cognitoUser = new CognitoUser(userData);
  cognitoUser.forgotPassword({
    onSuccess: function(data) {
      alert("Password reset link sent to your email.");
    },
    onFailure: function(err) {
      alert(err.message || JSON.stringify(err));
    }
  });
});

// Password Update Function
document.getElementById('updatePasswordForm').addEventListener('submit', function(event) {
  event.preventDefault();
  
  const email = document.getElementById('updateEmail').value;
  const newPassword = document.getElementById('newPassword').value;
  
  const userData = {
    Username: email,
    Pool: userPool
  };
  
  const cognitoUser = new CognitoUser(userData);
  
  cognitoUser.completeNewPasswordChallenge(newPassword, [], {
    onSuccess: function(result) {
      alert("Password updated successfully.");
    },
    onFailure: function(err) {
      alert(err.message || JSON.stringify(err));
    }
  });
});
