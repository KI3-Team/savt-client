module savt-client/savt-client-cli

go 1.23.4

replace savt-client/savt-client-api => ../savt-client-api

require (
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.0
)

require github.com/natefinch/lumberjack v2.0.0+incompatible // indirect

require (
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	savt-client/savt-client-api v0.0.0-00010101000000-000000000000
)
