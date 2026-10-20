for %%i in (proto/*.proto) do (
    protoc34 --proto_path=./proto  --go_out=./services --go_opt=paths=source_relative --go-grpc_out=./services --go-grpc_opt=paths=source_relative  %%i
)
pause;