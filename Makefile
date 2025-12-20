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
	@echo "  make android         - Rebuild debug APK and install (use after adding Expo packages)"
	@echo "  make android-debug   - Build debug APK only"
	@echo "  make android-install - Install debug APK on emulator/device"
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

# Expo Go (no native modules - faster, but Firebase/Google Sign-in won't work)
start-expo-dev:
	@echo "🛑 Stopping previous frontend..."
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "🔍 Type checking TypeScript..."
	@cd $(FRONTEND_DIR) && npm run tsc
	@echo "🧹 Clearing Metro cache..."
	@rm -rf $(FRONTEND_DIR)/.expo 2>/dev/null || true
	@rm -rf $(FRONTEND_DIR)/node_modules/.cache 2>/dev/null || true
	@echo "🚀 Starting Expo Go (no native modules)..."
	@cd $(FRONTEND_DIR) && npx expo start --clear

# Native Android build with hot-reload (includes Firebase, Google Sign-in)
start-native-android:
	@echo "🛑 Stopping previous frontend..."
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "🔍 Type checking TypeScript..."
	@cd $(FRONTEND_DIR) && npm run tsc
	@echo "🚀 Building & running native Android (with hot-reload)..."
	@cd $(FRONTEND_DIR) && npx expo run:android

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

# Build debug APK with native modules (use after adding new Expo packages)
android-debug:
	@echo "🔨 Building Android debug APK with native modules..."
	@cd $(FRONTEND_DIR) && npx expo prebuild --platform android --clean
	@cd $(FRONTEND_DIR)/android && ./gradlew assembleDebug
	@echo "✅ Debug APK built: $(FRONTEND_DIR)/android/app/build/outputs/apk/debug/app-debug.apk"

# Install debug APK on emulator/device
android-install:
	@echo "📲 Installing debug APK..."
	@adb install -r $(FRONTEND_DIR)/android/app/build/outputs/apk/debug/app-debug.apk
	@echo "✅ APK installed!"

# Build and install in one command
android: android-debug android-install
	@echo "🚀 Android app ready!"

# ============================================
# STOP
# ============================================
stop:
	@echo "Stopping all servers..."
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
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
	backend/proto/auth/*.proto backend/proto/learning/*.proto backend/proto/feed/*.proto
	@echo "Generating TypeScript proto files..."
	@protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto \
	--ts_proto_out=./frontend/proto/backend \
	--ts_proto_opt=esModuleInterop=true,outputServices=nice-grpc,env=browser,useExactTypes=false \
	--proto_path=./backend \
	backend/proto/auth/auth.proto backend/proto/learning/learning.proto backend/proto/feed/feed.proto
	@echo "Proto generation complete!"
