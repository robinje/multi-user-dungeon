# Multi-User Dungeon Engine

The goal of this project is to create a commercial-quality multi-user dungeon (MUD) engine that is flexible enough to be used as either a conventional MUD or an interactive fiction game.

The current implementation includes an SSH server for secure authentication and communication between the player and the server. The engine is primarily written in Go. Additionally, there is a user management system stub written in JavaScript and various utility scripts written in Python.

## Current Objectives

- [x] Create the TCP server for client connections.
- [x] Create a text parser for user input.
- [x] Implement a player authentication system.
- [x] Implement a database for the game.
- [x] Implement a character creation system.
- [x] Implement a text colorization system.
- [ ] Construct the item system.
- [ ] Develop game mechanics.
- [ ] Design an ecenomic framework.
- [ ] Implement a world creation system.
- [ ] Develop simple Non-Player Characters (NPCs).
- [ ] Design and implement a quest system.
- [ ] Build a direct messaging system.
- [ ] Develop more complex Non-Player Characters (NPCs) with basic AI.
- [ ] Implement a dynamic content updating system.
- [ ] Build an interactive password change system.
- [ ] Implement a player-to-player trading system.
- [ ] Develop more complex Non-Player Characters (NPCs) with basic AI.
- [ ] Build an interactive password change system.

## TODO

- [x] Fix output formatting for the client.
- [x] Allow players to enter their name.
- [x] Display the incoming IP address and Port on the server.
- [x] Add a help command.
- [x] Add a character list command.
- [x] Allow users to change their passwords.
- [x] Expand the character creation process.
- [x] Add take item command.
- [x] Add inventory command.
- [x] Add drop item command.
- [x] Add wear item command.
- [x] Add remove item command.
- [x] Add examine item command.
- [ ] Add a Message of the Day (MOTD) command.
- [ ] Implement Persistent Logging.
- [ ] Add the ability to delete characters.
- [ ] Add the ability to delete accounts.
- [ ] Implement an obscenity filter.
- [ ] Validate graph of loaded rooms and exits.
- [ ] Load item prototypes at start.
- [ ] Create function for creating items from prototypes.
- [ ] Add look at item command.
- [ ] Improve the say commands.
- [ ] Improve the input filters
- [ ] Ensure that a message is passed when a characters is added to the game.


## Deployment

Deploying the server involves several steps, from setting up your environment to running the server. Follow these steps to ensure a smooth deployment process:

1. **Install Python**: The server and utility scripts are now written in Python. Install Python 3.7 or later from the [Python website](https://www.python.org/).

2. **Set Up AWS Account**: An AWS account is required for deploying the server components, including DynamoDB and Cognito. Sign up for an account [here](https://aws.amazon.com/) if you don't already have one.

3. **Configure AWS Credentials**: Ensure you have AWS credentials configured on your machine. These credentials should have sufficient permissions to create DynamoDB tables, a Cognito user pool, and the necessary IAM policies and roles. You can configure your credentials by using the AWS CLI and running `aws configure`.

4. **Install Required Python Packages**: Install the necessary Python packages by running:
   ```
   pip install boto3 pyyaml
   ```

5. **Deploy AWS Resources**:
   - Navigate to the `scripts` directory within the project.
   - Run the `deploy.py` script using the command:
     ```
     python deploy.py
     ```
   This script will create the Cognito user pool, DynamoDB tables, and CodeBuild project. It will also generate the `config.yml` file needed to run the server.

6. **Initialize the Database**:
   - Navigate to the directory containing the `data_loader.py` script.
   - Run the script using the command:
     ```
     python data_loader.py -r rooms.json -a archetypes.json -p prototypes.json
     ```
   This will load the initial world data into your DynamoDB tables.

7. **Start the Server**: Start the server by running the main Python script from the root directory of the project:
   ```
   python main.py
   ```

8. **Verify Deployment**: You can verify the deployment by using the viewer script:
   ```
   python viewer.py --region your-aws-region
   ```
   This will display the contents of your DynamoDB tables.

Ensure all steps are completed without errors before trying to connect to the server. If you encounter any issues during deployment, refer to the AWS documentation or the specific tool's documentation for troubleshooting advice.

## Additional Tools

- **Create Item**: To add new items to rooms, use the `create_item.py` script:
  ```
  python create_item.py
  ```
  Follow the prompts to select a room and an item prototype, and the script will create a new item in the specified room.

Remember to keep your AWS credentials secure and never commit them to version control. It's recommended to use AWS IAM roles and policies to manage permissions securely.advice.

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.
