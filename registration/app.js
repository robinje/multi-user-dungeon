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
    var email = document.getElementById('email').value;
    var password = document.getElementById('password').value;

    var attributeList = [];

    var dataEmail = {
        Name: 'email',
        Value: email
    };

    var attributeEmail = new AmazonCognitoIdentity.CognitoUserAttribute(dataEmail);
    attributeList.push(attributeEmail);

    userPool.signUp(email, password, attributeList, null, function(err, result) {
        if (err) {
            alert(err.message || JSON.stringify(err));
            return;
        }
        var cognitoUser = result.user;
        console.log('User registration successful: ' + cognitoUser.getUsername());
    });
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
