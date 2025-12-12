module savt-client/savt-client-worker

go 1.24.0

replace savt-client/savt-client-api => ../savt-client-api

require (
	github.com/go-resty/resty/v2 v2.16.2
	github.com/gopacket/gopacket v1.5.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	golang.org/x/net v0.39.0
	google.golang.org/grpc v1.69.0
	google.golang.org/protobuf v1.35.2
	savt-client/savt-client-api v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
)
