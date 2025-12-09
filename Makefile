.PHONY: help start-backend start-frontend apk stop db-start db-stop migrate-up proto

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend

# Default target
help:
	@echo "LandR - Development Commands"
	@echo ""
	@echo "  make start-backend   - Stop, build, and start backend"
	@echo "  make start-frontend  - Stop, clear cache, and start frontend"
	@echo "  make apk             - Build release APK"
	@echo "  make stop            - Stop all servers"
	@echo "  make db-start        - Start PostgreSQL (Docker)"
	@echo "  make proto           - Generate proto files"
	@echo ""

# ============================================
# BACKEND
# ============================================
start-backend:
	@echo "🛑 Stopping previous backend..."
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
	@echo "🔨 Building backend..."
	@cd $(BACKEND_DIR) && go build -o bin/server cmd/server/main.go
	@echo "🚀 Starting backend..."
	@cd $(BACKEND_DIR) && ./bin/server

# ============================================
# FRONTEND
# ============================================
start-frontend:
	@echo "🛑 Stopping previous frontend..."
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "🧹 Clearing Metro cache..."
	@rm -rf $(FRONTEND_DIR)/.expo 2>/dev/null || true
	@rm -rf $(FRONTEND_DIR)/node_modules/.cache 2>/dev/null || true
	@echo "🚀 Starting frontend..."
	@cd $(FRONTEND_DIR) && npx expo start --clear

# ============================================
# APK BUILD
# ============================================
apk:
	@echo "🔨 Building Android APK..."
	@echo "Step 1: Cleaning previous build..."
	@rm -rf $(FRONTEND_DIR)/android 2>/dev/null || true
	@echo "Step 2: Running expo prebuild..."
	@cd $(FRONTEND_DIR) && npx expo prebuild --platform android --clean
	@echo "Step 3: Building release APK..."
	@cd $(FRONTEND_DIR)/android && ./gradlew assembleRelease
	@echo ""
	@echo "✅ APK built successfully!"
	@echo "📦 Location: $(FRONTEND_DIR)/android/app/build/outputs/apk/release/app-release.apk"

# ============================================
# STOP
# ============================================
stop:
	@echo "Stopping all servers..."
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "All servers stopped."

# ============================================
# DATABASE
# ============================================
db-start:
	@echo "Starting Docker and PostgreSQL..."
	@sudo systemctl start docker 2>/dev/null || sudo service docker start 2>/dev/null || true
	@sleep 2
	@docker start postgres_db 2>/dev/null || docker run -d --name postgres_db -e POSTGRES_USER=amityadav9314 -e POSTGRES_PASSWORD=amit8780 -e POSTGRES_DB=inkgrid -p 5432:5432 postgres:latest
	@echo "PostgreSQL started on port 5432"

db-stop:
	@docker stop postgres_db 2>/dev/null || true
	@echo "PostgreSQL stopped."

migrate-up:
	@migrate -path backend/db/migrations -database "postgres://amityadav9314:amit8780@localhost:5432/inkgrid?sslmode=disable" up

# ============================================
# PROTO
# ============================================
proto:
	@echo "Generating Go proto files..."
	@protoc --go_out=backend --go_opt=module=github.com/amityadav/landr \
	--go-grpc_out=backend --go-grpc_opt=module=github.com/amityadav/landr \
	backend/proto/auth/*.proto backend/proto/learning/*.proto
	@echo "Generating TypeScript proto files..."
	@protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto \
	--ts_proto_out=./frontend/proto/backend \
	--ts_proto_opt=esModuleInterop=true,outputServices=nice-grpc,env=browser,useExactTypes=false \
	--proto_path=./backend \
	backend/proto/auth/auth.proto backend/proto/learning/learning.proto
	@echo "Proto generation complete!"
