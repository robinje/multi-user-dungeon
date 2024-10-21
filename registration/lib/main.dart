import 'package:flutter/material.dart';
import 'package:amazon_cognito_identity_dart_2/cognito.dart';
import 'package:crypto/crypto.dart';
import 'dart:convert';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Cognito Email Verification',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: const EmailVerificationScreen(),
    );
  }
}

class EmailVerificationScreen extends StatefulWidget {
  const EmailVerificationScreen({super.key});

  @override
  State<EmailVerificationScreen> createState() =>
      _EmailVerificationScreenState();
}

class _EmailVerificationScreenState extends State<EmailVerificationScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  String _message = '';

  final userPool = CognitoUserPool(
    const String.fromEnvironment('USER_POOL_ID'),
    const String.fromEnvironment('CLIENT_ID'),
  );

  String calculateSecretHash(String username) {
    final clientSecret = const String.fromEnvironment('CLIENT_SECRET');
    final clientId = const String.fromEnvironment('CLIENT_ID');
    final key = utf8.encode(clientSecret);
    final message = utf8.encode(username + clientId);
    final hmac = Hmac(sha256, key);
    final digest = hmac.convert(message);
    return base64.encode(digest.bytes);
  }

  Future<void> _signUp() async {
    if (_formKey.currentState!.validate()) {
      try {
        final secretHash = calculateSecretHash(_emailController.text);
        await userPool.signUp(
          _emailController.text,
          'tempPassword123!', // This is a temporary password
          userAttributes: [
            AttributeArg(name: 'email', value: _emailController.text),
          ],
          validationData: [
            AttributeArg(name: 'SECRET_HASH', value: secretHash),
          ],
        );

        setState(() {
          _message = 'Verification email sent. Please check your inbox.';
        });
      } catch (e) {
        setState(() {
          if (e is CognitoClientException) {
            _message = 'Error: ${e.message}';
          } else {
            _message = 'An unexpected error occurred. Please try again.';
          }
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Email Verification'),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Form(
          key: _formKey,
          child: Column(
            children: <Widget>[
              TextFormField(
                controller: _emailController,
                decoration: const InputDecoration(labelText: 'Email'),
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return 'Please enter your email';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 20),
              ElevatedButton(
                onPressed: _signUp,
                child: const Text('Send Verification Email'),
              ),
              const SizedBox(height: 20),
              Text(_message),
            ],
          ),
        ),
      ),
    );
  }
}