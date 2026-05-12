# ====== CONFIG ======
DIST_DIR       := dist
CLIENT_DIR     := $(DIST_DIR)\client
SERVER_DIR     := $(DIST_DIR)\server

CLIENT_BIN     := client.exe
SERVER_BIN     := server.exe

MAIN_CLIENT    := ./cmd/client/main.go
MAIN_SERVER    := ./cmd/server/main.go

.PHONY: all build build-client build-server run run-client run-server clean

# ====== DEFAULT ======
all: build

# ====== BUILD BOTH ======
build: build-client build-server

# ====== BUILD CLIENT ======
build-client:
	@echo Creating client directory...
	@cmd /c "if not exist $(CLIENT_DIR) mkdir $(CLIENT_DIR)"

	@echo Copying lib files to client...
	@cmd /c "xcopy lib $(CLIENT_DIR)\ /E /I /Y > nul"

	@echo Copying assets to client...
	@cmd /c "xcopy assets $(CLIENT_DIR)\assets\ /E /I /Y > nul"

	@echo Building client binary with CGO...
	@cmd /c "set CGO_ENABLED=1 && go build -o $(CLIENT_DIR)\$(CLIENT_BIN) $(MAIN_CLIENT)"

	@echo Client build completed.

# ====== BUILD SERVER ======
build-server:
	@echo Creating server directory...
	@cmd /c "if not exist $(SERVER_DIR) mkdir $(SERVER_DIR)"

	@echo Copying lib files to server...
	@cmd /c "xcopy lib $(SERVER_DIR)\ /E /I /Y > nul"

	@echo Building server binary...
	@go build -o $(SERVER_DIR)\$(SERVER_BIN) $(MAIN_SERVER)

	@echo Server build completed.

build-server-linux:
	@echo Creating server directory...
	@cmd /c "if not exist $(SERVER_DIR) mkdir $(SERVER_DIR)"

	@echo Building server binary for Linux...
	@cmd /c "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o $(SERVER_DIR)\server_linux $(MAIN_SERVER)"

	@echo Server Linux build completed.

# ====== RUN BOTH ======
run: build
	@echo Starting server in new window...
	@start "Game Server" cmd /c "$(SERVER_DIR)\$(SERVER_BIN)"

	@echo Starting client in new window...
	@start "Game Client" cmd /c "$(CLIENT_DIR)\$(CLIENT_BIN)"

# ====== RUN CLIENT ONLY ======
run-client: build-client
	@echo Starting client...
	@start "Game Client" cmd /c "$(CLIENT_DIR)\$(CLIENT_BIN)"

# ====== RUN SERVER ONLY ======
run-server: build-server
	@echo Starting server...
	@start "Game Server" cmd /c "$(SERVER_DIR)\$(SERVER_BIN)"

# ====== CLEAN ======
clean:
	@echo Cleaning dist directory...
	-@cmd /c "rmdir /S /Q $(DIST_DIR) > nul 2>&1"
