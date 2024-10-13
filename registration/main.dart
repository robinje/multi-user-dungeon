import 'package:flutter/material.dart';
import 'package:amazon_cognito_identity_dart_2/cognito.dart';

void main() {
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Cognito Email Verification',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: EmailVerificationScreen(),
    );
  }
}

class EmailVerificationScreen extends StatefulWidget {
  @override
  _EmailVerificationScreenState createState() => _EmailVerificationScreenState();
}

class _EmailVerificationScreenState extends State<EmailVerificationScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  String _message = '';

  final userPool = CognitoUserPool(
    const String.fromEnvironment('USER_POOL_ID'),
    const String.fromEnvironment('CLIENT_ID'),
  );

  Future<void> _signUp() async {
    if (_formKey.currentState!.validate()) {
      try {
        final signUpResult = await userPool.signUp(
          _emailController.text,
          'tempPassword123!',  // This is a temporary password
          userAttributes: [
            AttributeArg(name: 'email', value: _emailController.text),
          ],
        );

        setState(() {
          _message = 'Verification email sent. Please check your inbox.';
        });
      } catch (e) {
        setState(() {
          _message = 'Error: ${e.toString()}';
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Email Verification'),
      ),
      body: Padding(
        padding: EdgeInsets.all(16.0),
        child: Form(
          key: _formKey,
          child: Column(
            children: <Widget>[
              TextFormField(
                controller: _emailController,
                decoration: InputDecoration(labelText: 'Email'),
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return 'Please enter your email';
                  }
                  return null;
                },
              ),
              SizedBox(height: 20),
              ElevatedButton(
                onPressed: _signUp,
                child: Text('Send Verification Email'),
              ),
              SizedBox(height: 20),
              Text(_message),
            ],
          ),
        ),
      ),
    );
  }
}