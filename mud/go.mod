module github.com/robinje/multi-user-dungeon/mud

go 1.22

replace github.com/robinje/multi-user-dungeon/core => ../core

require (
	github.com/robinje/multi-user-dungeon/core v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.24.0
)

require (
	github.com/aws/aws-sdk-go v1.54.15 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	go.etcd.io/bbolt v1.3.10 // indirect
	golang.org/x/sys v0.21.0 // indirect
)
