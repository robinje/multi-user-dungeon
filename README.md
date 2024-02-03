# Multi-User Dungeon 

The goal of this project is to create a commerical quality multi-user dungeon (MUD) engine that is flexible enough that it can be used as a conventional MUD or interactive fiction game.

The current implimentation is an SSH server that allows for secure authentication and communication between the player and the server.

The engine is being written in Go. There is a user management system that is a stub which is written in JavaScript. There are some scripts written in Python.

The current objectives are:

- [x] Create the TCP server for clinet connections.
- [x] Create a text parser for user input.
- [x] Player authentication system
- [ ] Impliment a database for the game.
- [ ] Player creation system.
- [ ] Character creation system.
- [ ] Build a private messaging system.
- [ ] Build game mechanics.
- [ ] Build a combat system.

TODO:

- [x] Fix output formatting for the client.
- [x] Allow players to enter their name.
- [x] Display in the incoming IP address and Port on the server.
- [x] Add a help command.
- [ ] Add a message of the day (MOTD) command.
- [ ] Add a player list command.
- [ ] Impliment Persistant Logging.
- [ ] Allow users to change their passwords.