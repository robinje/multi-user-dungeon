[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

![GitHub](https://img.shields.io/badge/github-%23121011.svg?style=for-the-badge&logo=github&logoColor=white)
![Dependabot](https://img.shields.io/badge/dependabot-025E8C?style=for-the-badge&logo=dependabot&logoColor=white)
![GitHub Actions](https://img.shields.io/badge/github%20actions-%232671E5.svg?style=for-the-badge&logo=githubactions&logoColor=white)

![AWS](https://img.shields.io/badge/AWS-%23FF9900.svg?style=for-the-badge&logo=amazon-aws&logoColor=white)
![AmazonDynamoDB](https://img.shields.io/badge/Amazon%20DynamoDB-4053D6?style=for-the-badge&logo=Amazon%20DynamoDB&logoColor=white)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Python](https://img.shields.io/badge/python-3670A0?style=for-the-badge&logo=python&logoColor=ffdd54)
![JavaScript](https://img.shields.io/badge/javascript-%23323330.svg?style=for-the-badge&logo=javascript&logoColor=%23F7DF1E)
![Webpack](https://img.shields.io/badge/webpack-%238DD6F9.svg?style=for-the-badge&logo=webpack&logoColor=black)

![Linux](https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black)
![Windows](https://img.shields.io/badge/Windows-0078D6?style=for-the-badge&logo=windows&logoColor=white)

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
- [x] Add Cloudwatch Logs and Metrics.
- [x] Build an interactive password change system.
- [ ] Develop a weather and time system.
- [ ] Construct the item system.
- [ ] Create a crafting system for items.
- [ ] Develop game mechanics.
- [ ] Design an economic framework
- [ ] Build a direct messaging system.
- [ ] Develop simple Non-Player Characters (NPCs).
- [ ] Design and implement a quest system.
- [ ] Implement a dynamic content updating system.
- [ ] Implement a player-to-player trading system.
- [ ] Implement a party system for cooperative gameplay.
- [ ] Implement a magic system.
- [ ] Impliment a quest tracking system.
- [ ] Impliment a reputation system.
- [ ] Develop a conditional room description system.
- [ ] Implement a world creation system.
- [ ] Develop more complex Non-Player Characters (NPCs) with basic AI.

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
- [x] Implement Persistent Logging.
- [x] Load item prototypes at start.
- [x] Create function for creating items from prototypes.
- [x] Ensure that a message is passed when a character is added to the game.
- [x] Add a Message of the Day (MOTD) command.
- [x] Add Bloom Filter to check for existing characters names being used.
- [x] Add the ability to delete characters.
- [x] Allow starting room to be set by Archtype.
- [ ] Add look at item command.
- [ ] Implement an obscenity filter.
- [ ] Validate graph of loaded rooms and exits.
- [ ] Improve the say command.
- [ ] Improve the input filters
- [ ] Create administrative interface.
- [ ] Force Password Resets when needed.
- [ ] Add the ability to delete accounts.
- [ ] Add the ability to ban accounts.
- [ ] Add the ability to mute accounts.

## Project Overview

The engine is primarily written in Go (version 1.22) with an SSH server for secure authentication and communication between the player and the server. Additionally, there are database utility scripts written in Python (version 3.12) and various deployment scripts.

Key components:

- Go server (v1.22) for game logic and player interactions
- Python (v3.12) scripts for database management and deployment
- AWS services for database (DynamoDB) and Identity Provider (Cognito)
- CloudFormation templates for AWS resource management

## Deployment

Deploying the server involves several steps:

1. Ensure you have Go 1.22 and Python 3.12 installed.
2. Clone the repository.
3. Install the required Python packages:
   ```
   pip install -r requirements/scripts-requirements.txt
   ```
4. Set up your AWS credentials (access key ID and secret access key) in your environment variables or AWS credentials file.
5. Run the deployment script:
   ```
   python scripts/deploy.py
   ```
   This script will create the necessary AWS resources using CloudFormation.
6. Once deployment is complete, build and run the server:
   ```
   go build ./ssh_server
   ./ssh_server
   ```

## Development

- The `core/` directory contains the main game logic and types.
- The `ssh_server/` directory contains the main server implementation.
- The `database/` directory contains Python scripts for database management.
- The `scripts/` directory contains deployment and utility scripts.

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.
