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
- [ ] Construct the item system.
- [ ] Develop game mechanics.
- [ ] Design an ecenomic framework.
- [ ] Implement a world creation system.
- [ ] Develop simple Non-Player Characters (NPCs).
- [ ] Design and implement a quest system.
- [ ] Build a direct messaging system.
- [ ] Develop more complex Non-Player Characters (NPCs) with basic AI.
- [ ] Implement a dynamic content updating system.
- [ ] Implement a player-to-player trading system.
- [ ] Create a crafting system for items.
- [ ] Develop a weather and time system.
- [ ] Implement a party system for cooperative gameplay.
- [ ] Implement a magic system.


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
- [x] Ensure that a message is passed when a characters is added to the game.
- [x] Add a Message of the Day (MOTD) command.
- [ ] Add the ability to delete characters.
- [ ] Add the ability to delete accounts.
- [ ] Implement an obscenity filter.
- [ ] Validate graph of loaded rooms and exits.
- [ ] Add look at item command.
- [ ] Improve the say commands.
- [ ] Improve the input filters
- [ ] Create administrative interface.



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

