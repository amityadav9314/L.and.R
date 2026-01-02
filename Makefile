.PHONY: help build test-agent start-backend start-frontend desktop deploy-desktop apk aab stop db-start db-stop migrate-up proto prod

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
DESKTOP_DIR := frontend/desktop
DEPLOY_DIR := /var/www/landr/desktop

# Default target
help:
	@echo "LandR - Development Commands"
	@echo ""
	@echo "  make build           - Build all"
	@echo "  make start-backend   - Stop, build, and start backend"
	@echo "  make start-frontend  - Stop, clear cache, and start mobile frontend (Expo)"
	@echo "  make desktop         - Type check and start desktop frontend (Vite dev server)"
	@echo "  make deploy-desktop  - Build and deploy desktop to production (/var/www/landr/desktop)"
	@echo "  make apk             - Build release APK"
	@echo "  make aab             - Build AAB for Google Play"
	@echo "  make android         - Rebuild debug APK and install (use after adding Expo packages)"
	@echo "  make android-debug   - Build debug APK only"
	@echo "  make android-install - Install debug APK on emulator/device"
	@echo "  make stop            - Stop all servers"
	@echo "  make prod            - Deploy to production (desktop + backend + frontend)"
	@echo "  make test-agent      - Run agent test with mocked Tavily/SerpApi"
	@echo "  make db-start        - Start PostgreSQL (Docker)"
	@echo "  make proto           - Generate proto files"
	@echo ""

# ============================================
# BUILD (compile both backend and frontend)
# ============================================
build:
	@echo "🔨 Building backend..."
	@cd $(BACKEND_DIR) && go build ./...
	@echo "✅ Backend compiled successfully"
	@echo "🔍 Type checking mobile frontend..."
	@cd $(FRONTEND_DIR) && npx tsc --noEmit --skipLibCheck 2>/dev/null || echo "⚠️  Mobile frontend has type warnings (non-blocking)"
	@echo "🔍 Type checking desktop frontend..."
	@cd $(DESKTOP_DIR) && npx tsc -p tsconfig.app.json --noEmit
	@echo "✅ Desktop type check passed"
	@echo ""
	@echo "🎉 Build complete!"

# ============================================
# TESTING (Agent with mocked search APIs)
# ============================================
test-agent:
	@echo "🧪 Running Agent Test (Mocked Search)..."
	@echo "   Uses: REAL Groq, REAL DB, MOCKED Tavily/SerpApi"
	@cd $(BACKEND_DIR) && go test -v -timeout 15m ./internal/adk/feedagent/...
	@echo "✅ Test complete!"

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
	@echo "Starting frontend..."
	cd $(FRONTEND_DIR) && npm install && npm start -- -w


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

desktop:
	@echo "🛑 Stopping previous desktop frontend..."
	@lsof -ti:5173 | xargs -r kill -9 2>/dev/null || true
	@echo "🔍 Type checking Desktop frontend..."
	@cd $(DESKTOP_DIR) && npx tsc -p tsconfig.app.json --noEmit
	@echo "🚀 Starting Desktop frontend..."
	@cd $(DESKTOP_DIR) && npm run dev

deploy-desktop:
	@echo "📦 Installing Desktop dependencies..."
	@cd $(DESKTOP_DIR) && npm install
	@echo "🏗️  Building Desktop for production..."
	@cd $(DESKTOP_DIR) && npm run build
	@echo "📦 Deploying to $(DEPLOY_DIR)..."
	@sudo mkdir -p $(DEPLOY_DIR)
	@sudo rm -rf $(DEPLOY_DIR)/*
	@sudo cp -r $(DESKTOP_DIR)/dist/* $(DEPLOY_DIR)/
	@sudo chown -R www-data:www-data $(DEPLOY_DIR)
	@echo "✅ Desktop deployed successfully!"
	@echo "🌐 Access at: https://landr.aky.net.in (from desktop browser)"

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

# Build AAB (Android App Bundle) for Google Play - PRODUCTION SIGNED
aab:
	@echo "🔨 Building Android App Bundle (AAB) for Google Play..."
	@echo "Step 1: Cleaning previous build..."
	@rm -rf $(FRONTEND_DIR)/android 2>/dev/null || true
	@echo "Step 2: Running expo prebuild..."
	@cd $(FRONTEND_DIR) && npx expo prebuild --platform android --clean
	@echo "Step 3: Applying production signing config..."
	@echo "" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "// Production Signing Configuration" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "def keystorePropertiesFile = rootProject.file('../keystore/keystore.properties')" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "def keystoreProperties = new Properties()" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "if (keystorePropertiesFile.exists()) { keystoreProperties.load(new FileInputStream(keystorePropertiesFile)) }" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "android.signingConfigs.create('release') {" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "    storeFile file(keystoreProperties['LANDR_UPLOAD_STORE_FILE'])" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "    storePassword keystoreProperties['LANDR_UPLOAD_STORE_PASSWORD']" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "    keyAlias keystoreProperties['LANDR_UPLOAD_KEY_ALIAS']" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "    keyPassword keystoreProperties['LANDR_UPLOAD_KEY_PASSWORD']" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "}" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "android.buildTypes.release.signingConfig = android.signingConfigs.release" >> $(FRONTEND_DIR)/android/app/build.gradle
	@echo "Step 4: Building release AAB..."
	@cd $(FRONTEND_DIR)/android && ./gradlew bundleRelease
	@echo ""
	@echo "✅ AAB built successfully (PRODUCTION SIGNED)!"
	@echo "📦 Location: $(FRONTEND_DIR)/android/app/build/outputs/bundle/release/app-release.aab"
	@echo "🚀 Upload this file to Google Play Console"

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
	@lsof -ti:5173 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "All servers stopped."

# ============================================
# PRODUCTION DEPLOY
# ============================================
prod:
	@echo "🚀 Deploying to production..."
	@echo "Step 1: Stopping all servers..."
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:5173 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:8081 | xargs -r kill -9 2>/dev/null || true
	@echo "Step 2: Deploying desktop..."
	@nohup $(MAKE) deploy-desktop > desktop.log 2>&1 &
	@sleep 5
	@echo "Step 3: Starting backend..."
	@nohup $(MAKE) start-backend > backend.log 2>&1 &
	@sleep 3
	@echo "Step 4: Starting frontend (Expo)..."
	@nohup $(MAKE) start-frontend > frontend.log 2>&1 &
	@echo ""
	@echo "✅ Production deployment started!"
	@echo "📋 Logs:"
	@echo "   - Desktop:  tail -f desktop.log"
	@echo "   - Backend:  tail -f backend.log"
	@echo "   - Frontend: tail -f frontend.log"

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
# PROTO - Generates for: Go backend, Mobile TS, Desktop TS
# ============================================
proto:
	@echo "🔧 Generating Go proto files (backend)..."
	@protoc --go_out=backend --go_opt=module=github.com/amityadav/landr \
	--go-grpc_out=backend --go-grpc_opt=module=github.com/amityadav/landr \
	backend/proto/auth/*.proto backend/proto/learning/*.proto backend/proto/feed/*.proto
	@echo "✅ Go proto files generated"
	
	@echo "🔧 Generating TypeScript proto files (mobile)..."
	@protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto \
	--ts_proto_out=./frontend/proto/backend \
	--ts_proto_opt=esModuleInterop=true,outputServices=nice-grpc,env=browser,useExactTypes=false \
	--proto_path=./backend \
	backend/proto/auth/auth.proto backend/proto/learning/learning.proto backend/proto/feed/feed.proto
	@echo "✅ Mobile TypeScript proto files generated"
	
	@echo "🔧 Generating TypeScript proto files (desktop)..."
	@protoc --plugin=./frontend/node_modules/.bin/protoc-gen-ts_proto \
	--ts_proto_out=./frontend/desktop/src/proto/backend \
	--ts_proto_opt=esModuleInterop=true,outputServices=nice-grpc,env=browser,useExactTypes=false \
	--proto_path=./backend \
	backend/proto/auth/auth.proto backend/proto/learning/learning.proto backend/proto/feed/feed.proto
	@echo "✅ Desktop TypeScript proto files generated"
	
	@echo ""
	@echo "🎉 Proto generation complete! Generated for:"
	@echo "   - Go backend:    backend/pkg/pb/"
	@echo "   - Mobile (RN):   frontend/proto/backend/"
	@echo "   - Desktop (Web): frontend/desktop/src/proto/backend/"
