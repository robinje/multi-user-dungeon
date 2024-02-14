# Multi-User Dungeon 

The goal of this project is to create a commerical quality multi-user dungeon (MUD) engine that is flexible enough that it can be used as a conventional MUD or interactive fiction game.

The current implimentation is an SSH server that allows for secure authentication and communication between the player and the server.

The engine is being written in Go. There is a user management system that is a stub which is written in JavaScript. There are some scripts written in Python.

The current objectives are:

- [x] Create the TCP server for clinet connections.
- [x] Create a text parser for user input.
- [x] Player authentication system
- [x] Impliment a database for the game.
- [x] Character creation system.
- [ ] Build a direct messaging system.
- [ ] Build game mechanics.
- [ ] Player creation system.
- [ ] Build a world creation system.
- [ ] Build a quest system.
- [ ] Build the object system.
- [ ] Build Simple Non-Player Characters (NPCs).
- [ ] Build AI Controlled NPCs
- [ ] Build dynamic conent updating system.


TODO:

- [x] Fix output formatting for the client.
- [x] Allow players to enter their name.
- [x] Display in the incoming IP address and Port on the server.
- [x] Add a help command.
- [x] Add a character list command.
- [x] Allow users to change their passwords.
- [ ] Add a message of the day (MOTD) command.
- [ ] Impliment Persistant Logging.
- [ ] Add the ability to delete characters.
- [ ] Add the ability to delete accounts.
- [ ] Add an obscentity filter.
- [ ] Graph validation of loaded rooms and exits.
- [ ] Expand the character creation process.


## Demployment

To deploy the server, you will need to have Go installed on your system. You can download it from the [Go website](https://golang.org/).

You will need to have an AWS account to deploy the server. You can sign up for an account [here](https://aws.amazon.com/).

You will need AWS credentials for that account which have sufficient permissions to create the Cognito user pool, and the IAM policies and roles.

Run the `./scripts/deploy_cognito.py` script to create the Cognito instace and the IAM policies and roles, it will also generate the `config.json` file that is needed to run the server.

Install the Go dependencies by running `go mod download`.

Start the server by running `go run .` in the root directory of the project.

