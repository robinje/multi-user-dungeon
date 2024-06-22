module github.com/robinje/multi-user-dungeon

go 1.21

replace github.com/robinje/multi-user-dungeon/mud => ./mud

replace github.com/robinje/multi-user-dungeon/database_loader => ./database_loader

require go.etcd.io/bbolt v1.3.10

require golang.org/x/sys v0.4.0 // indirect
