# ====== CONFIG ======
DIST_DIR       := dist
CLIENT_DIR     := $(DIST_DIR)/client
SERVER_DIR     := $(DIST_DIR)/server
LINUX_DIR      := $(DIST_DIR)/linux

CLIENT_BIN     := client
SERVER_BIN     := server
LINUX_BIN      := server-linux-amd64
LINUX_TAR      := $(DIST_DIR)/$(LINUX_BIN).tar.gz

MAIN_CLIENT    := ./cmd/client/main.go
MAIN_SERVER    := ./cmd/server/main.go

.PHONY: all build build-client build-server build-server-linux dist-server-linux run run-client run-server clean test

# ====== DEFAULT ======
all: build

# ====== BUILD BOTH ======
build: build-client build-server

# ====== BUILD CLIENT ======
build-client:
	@echo "Creating client directory..."
	@mkdir -p $(CLIENT_DIR)

	@echo "Copying lib files to client..."
	@cp -r lib/* $(CLIENT_DIR)/ 2>/dev/null || true

	@echo "Copying assets to client..."
	@cp -r assets $(CLIENT_DIR)/

	@echo "Building client binary with CGO..."
	@CGO_ENABLED=1 go build -o $(CLIENT_DIR)/$(CLIENT_BIN) $(MAIN_CLIENT)

	@echo "Client build completed: $(CLIENT_DIR)/$(CLIENT_BIN)"

# ====== BUILD SERVER (native) ======
build-server:
	@echo "Creating server directory..."
	@mkdir -p $(SERVER_DIR)

	@echo "Building server binary..."
	@CGO_ENABLED=0 go build -o $(SERVER_DIR)/$(SERVER_BIN) $(MAIN_SERVER)

	@echo "Server build completed: $(SERVER_DIR)/$(SERVER_BIN)"

# ====== BUILD SERVER FOR LINUX ======
build-server-linux:
	@echo "Creating linux build directory..."
	@mkdir -p $(LINUX_DIR)

	@echo "Building server binary for Linux amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(LINUX_DIR)/$(LINUX_BIN) $(MAIN_SERVER)

	@echo "Linux server build completed: $(LINUX_DIR)/$(LINUX_BIN)"

# ====== CREATE TAR.GZ FOR LINUX SERVER ======
dist-server-linux: build-server-linux
	@echo "Creating tar.gz archive..."
	@tar -czf $(LINUX_TAR) -C $(LINUX_DIR) $(LINUX_BIN)
	@echo "Archive created: $(LINUX_TAR)"
	@ls -lh $(LINUX_TAR)

# ====== TEST BUILD ======
test: dist-server-linux
	@echo "Testing linux binary..."
	@file $(LINUX_DIR)/$(LINUX_BIN)
	@echo "Extract test:"
	@mkdir -p $(DIST_DIR)/test_extract
	@tar -xzf $(LINUX_TAR) -C $(DIST_DIR)/test_extract
	@ls -la $(DIST_DIR)/test_extract/
	@rm -rf $(DIST_DIR)/test_extract

# ====== RUN BOTH ======
run: build
	@echo "Starting server..."
	@$(SERVER_DIR)/$(SERVER_BIN) &
	@sleep 1
	@echo "Starting client..."
	@cd $(CLIENT_DIR) && ./$(CLIENT_BIN)

# ====== RUN CLIENT ONLY ======
run-client: build-client
	@echo "Starting client..."
	@cd $(CLIENT_DIR) && ./$(CLIENT_BIN)

# ====== RUN SERVER ONLY ======
run-server: build-server
	@echo "Starting server..."
	@$(SERVER_DIR)/$(SERVER_BIN)

# ====== CLEAN ======
clean:
	@echo "Cleaning dist directory..."
	@rm -rf $(DIST_DIR)
	@echo "Clean done."
