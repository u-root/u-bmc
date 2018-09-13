all: bmc.pb.go

bmc.pb.go: bmc.proto
	protoc --go_out=plugins=grpc:. $?
