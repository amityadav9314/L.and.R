.PHONY: help build build-backend build-frontend start start-backend start-frontend stop clean clean-backend clean-frontend verify-frontend run-android

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
APP_NAME := server.exe

# Default target
help:
	@echo "Learn and Revise - Development Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build backend and install frontend dependencies"
	@echo "  make start          - Start both backend and frontend servers in new windows"
	@echo "  make stop           - Stop all running servers"
	@echo "  make clean          - Clean build artifacts"
	@echo ""

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	cd $(BACKEND_DIR) && go build -o bin/$(APP_NAME) cmd/server/main.go

build-frontend:
	@echo "Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && npm install
	@echo "Running type check..."
	cd $(FRONTEND_DIR) && npm run tsc

# Start
start:
	@echo "Starting servers in separate windows..."
	start "LandR Backend" cmd /c "cd $(BACKEND_DIR) && go run cmd/server/main.go"
	start "LandR Frontend" cmd /c "cd $(FRONTEND_DIR) && npm start"

start-backend:
	@echo "Starting backend..."
	cd $(BACKEND_DIR) && go run cmd/server/main.go

start-frontend:
	@echo "Starting frontend..."
	cd $(FRONTEND_DIR) && npm start -- -w

# Stop
stop:
	@echo "Stopping servers..."
	@taskkill /F /IM $(APP_NAME) 2>nul || echo Backend not running
	@taskkill /F /IM node.exe 2>nul || echo Node processes not running (Warning: This might kill other Node processes)
	@echo "Note: You might need to manually close the opened terminal windows."

# Clean
clean: clean-backend clean-frontend

clean-backend:
	@echo "Cleaning backend..."
	cd $(BACKEND_DIR) && del /Q bin\$(APP_NAME) 2>nul || echo No backend artifacts

clean-frontend:
	@echo "Cleaning frontend..."
	cd $(FRONTEND_DIR) && rmdir /S /Q node_modules 2>nul || echo No frontend modules

# Proto
proto:
	@echo "Generating Go proto files..."
	protoc --go_out=backend --go_opt=module=github.com/amityadav/landr \
	--go-grpc_out=backend --go-grpc_opt=module=github.com/amityadav/landr \
	backend/proto/auth/*.proto backend/proto/learning/*.proto
	@echo "Generating TypeScript proto files..."
	protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto \
	--ts_proto_out=./frontend/proto/backend \
	--ts_proto_opt=esModuleInterop=true,outputServices=nice-grpc,env=browser,useExactTypes=false \
	--proto_path=./backend \
	backend/proto/auth/auth.proto backend/proto/learning/learning.proto
	@echo "Proto generation complete!"

# Database
migrate-up:
	migrate -path backend/db/migrations -database "postgres://amityadav9314:amit8780@localhost:5432/inkgrid?sslmode=disable" up

# Verify frontend TypeScript
verify-frontend:
	@echo "Type checking frontend..."
	cd $(FRONTEND_DIR) && npm run tsc
	@echo "Frontend type check complete!"

# Run Android app
run-android:
	cd $(FRONTEND_DIR) && npm run android
