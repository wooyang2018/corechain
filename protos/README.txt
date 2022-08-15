protoc --go_out=protos --go_opt=paths=source_relative -I ./protos protos/chainbft.proto
protoc --go_out=protos --go_opt=paths=source_relative -I ./protos protos/permission.proto
protoc --go_out=protos --go_opt=paths=source_relative -I ./protos protos/contract.proto
protoc --go_out=protos --go_opt=paths=source_relative -I ./protos protos/proposal.proto
protoc --go_out=protos --go_opt=paths=source_relative -I ./protos protos/status.proto

protoc --go_out=protos --go_opt=paths=source_relative --go-grpc_out=protos --go-grpc_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false -I ./protos protos/event_service.proto
protoc --go_out=protos --go_opt=paths=source_relative --go-grpc_out=protos --go-grpc_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false -I ./protos protos/syscall_service.proto
protoc --go_out=protos --go_opt=paths=source_relative --go-grpc_out=protos --go-grpc_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false -I ./protos protos/network_service.proto
protoc --go_out=protos --go_opt=paths=source_relative --go-grpc_out=protos --go-grpc_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false -I ./protos protos/chain_service.proto