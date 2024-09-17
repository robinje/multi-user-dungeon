import { CognitoUserPool, CognitoUserAttribute } from 'amazon-cognito-identity-js';
import './style.css'; // Webpack will process your CSS

// Configuration for Cognito User Pool
const poolData = {
  UserPoolId: 'YOUR_USER_POOL_ID',  // Replace with your Cognito User Pool ID
  ClientId: 'YOUR_CLIENT_ID'  // Replace with your Cognito Client ID
};

const userPool = new CognitoUserPool(poolData);

// Registration function
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
  const attributeEmail = new CognitoUserAttribute({ Name: "email", Value: email });
  attributeList.push(attributeEmail);
  
  userPool.signUp(email, password, attributeList, null, function(err, result) {
    if (err) {
      alert(err.message || JSON.stringify(err));
      return;
    }
    alert("Registration successful. Please check your email for verification.");
  });
});
