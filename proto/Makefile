all: bmc.pb.go config.pb.go

%.pb.go: %.proto
	protoc --go_out=plugins=grpc:. $?
