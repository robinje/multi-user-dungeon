module github.com/robinje/multi-user-dungeon/database

go 1.22

replace github.com/robinje/multi-user-dungeon/core => ../core

require github.com/robinje/multi-user-dungeon/core v0.0.0-00010101000000-000000000000

require golang.org/x/crypto v0.24.0 // indirect

require (
	github.com/google/uuid v1.6.0 // indirect
	go.etcd.io/bbolt v1.3.10 // indirect
	golang.org/x/sys v0.21.0 // indirect
)
