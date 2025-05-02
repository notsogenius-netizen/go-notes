# Notes Management System

## 📦 System Components

### 1. API Gateway (GraphQL)
- **Entry point** for all client requests
- **Translates** GraphQL to gRPC calls
- **Port**: `8080`

### 2. Notes Service (gRPC)
- **Core business logic** for notes CRUD
- **Streaming notifications** for real-time updates  
- **Port**: `50051`

## 🚀 Quick Start

### Prerequisites
```bash
# Install protobuf compiler
brew install protobuf  # macOS
sudo apt install protobuf-compiler  # Linux

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

# 1. Start Notes Service (new terminal)
```bash
cd notes-service
go run cmd/server/main.go
```
# 2. Start API Gateway (new terminal) 
```bash
cd api-gateway
go run cmd/server/server.go
```
# 3. Access GraphQL Playground
open http://localhost:8080