document.getElementById('registrationForm').addEventListener('submit', function(event) {
    event.preventDefault();
    registerUser();
});

document.getElementById('passwordResetRequestForm').addEventListener('submit', function(event) {
    event.preventDefault();
    requestPasswordReset();
});

document.getElementById('passwordResetForm').addEventListener('submit', function(event) {
    event.preventDefault();
    resetPassword();
});

var poolData = {
    UserPoolId: 'YOUR_COGNITO_USER_POOL_ID',
    ClientId: 'YOUR_COGNITO_CLIENT_ID'
};

var userPool = new AmazonCognitoIdentity.CognitoUserPool(poolData);

function registerUser() {
    // ... existing registration code ...
}

function requestPasswordReset() {
    var email = document.getElementById('resetEmail').value;
    var userData = {
        Username: email,
        Pool: userPool
    };

    var cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);
    cognitoUser.forgotPassword({
        onSuccess: function (result) {
            console.log('Password reset request successful');
            document.getElementById('passwordResetForm').style.display = 'block';
        },
        onFailure: function(err) {
            alert(err.message || JSON.stringify(err));
        }
    });
}

function resetPassword() {
    var email = document.getElementById('resetEmail').value;
    var verificationCode = document.getElementById('verificationCode').value;
    var newPassword = document.getElementById('newPassword').value;
    
    var userData = {
        Username: email,
        Pool: userPool
    };

    var cognitoUser = new AmazonCognitoIdentity.CognitoUser(userData);
    cognitoUser.confirmPassword(verificationCode, newPassword, {
        onSuccess() {
            console.log('Password reset successful');
        },
        onFailure(err) {
            alert(err.message || JSON.stringify(err));
        }
    });
}
