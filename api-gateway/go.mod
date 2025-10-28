module github.com/sonuudigital/microservices/api-gateway

go 1.25.3

require (
	github.com/joho/godotenv v1.5.1
	github.com/sonuudigital/microservices/gen v0.0.0-00010101000000-000000000000
	github.com/sonuudigital/microservices/shared v0.0.0-20251022201705-f3843615e342
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sonuudigital/microservices/gen => ../gen
