module github.com/1RAFTIK1/linkpulse-dashboard

go 1.26.5

replace github.com/1RAFTIK1/linkpulse-contracts => ../linkpulse-contracts

require (
	github.com/1RAFTIK1/linkpulse-contracts v0.0.0-00010101000000-000000000000
	github.com/coder/websocket v1.8.15
	google.golang.org/grpc v1.82.0
)

require (
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
