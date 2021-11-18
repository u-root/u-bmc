# gRPC definitions for u-bmc

When updating the .proto file, make sure to update and commit the updated Go
implementation. The reason for this is to make it trivial to include the
protocol definitions without having to do a pre-build step in Go.

1. Make sure you have `protoc` installed. On Debian this can be installed via
`apt-get install protobuf-compiler`.

2. Install `protoc-gen-go` by running:
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
```

3. Run `task protogen` to update the .pb.go file(s).

