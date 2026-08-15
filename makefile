FRONTEND_DIR := frontend
WAILS_VERSION := v2.14.0

.PHONY: init install up build

init:
	@case "$$(uname -s)" in \
		Darwin) $(MAKE) init/mac ;; \
		Linux) $(MAKE) init/linux ;; \
		MINGW*|MSYS*|CYGWIN*) $(MAKE) init/win ;; \
		*) echo "未対応のOSです: $$(uname -s)"; exit 1 ;; \
	esac
	make install

init/mac:
	@command -v go >/dev/null || (echo "Goが必要です: https://go.dev/dl/"; exit 1)
	@command -v npm >/dev/null || (echo "Node.js/npmが必要です: https://nodejs.org/"; exit 1)
	go install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)
	cd $(FRONTEND_DIR) && npm install
	@echo "macOSの初期化が完了しました。"

init/linux:
	@command -v go >/dev/null || (echo "Goが必要です: https://go.dev/dl/"; exit 1)
	@command -v npm >/dev/null || (echo "Node.js/npmが必要です: https://nodejs.org/"; exit 1)
	sudo apt-get update
	@if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then \
		sudo apt-get install -y pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev; \
	else \
		sudo apt-get install -y pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev; \
	fi
	go install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)
	@echo "Linuxの初期化が完了しました。"

init/win:
	powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/init-windows.ps1
	
install:
	cd $(FRONTEND_DIR) && npm i
up:
	cd $(FRONTEND_DIR) && npm run build
	wails dev

build:
	@case "$$(uname -s)" in \
		Darwin) $(MAKE) build/mac ;; \
		Linux) $(MAKE) build/linux ;; \
		MINGW*|MSYS*|CYGWIN*) $(MAKE) build/win ;; \
		*) echo "未対応のOSです: $$(uname -s)"; exit 1 ;; \
	esac

build/win:
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform windows/amd64 -clean
	mkdir -p build/win-amd64
	rm -f build/win-amd64/tabpo.exe
	mv build/bin/tabpo.exe build/win-amd64/tabpo.exe
	@echo "Windowsのビルドが完了しました。"

build/mac:
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform darwin/arm64 -clean
	mkdir -p build/mac-arm64
	rm -rf build/mac-arm64/tabpo.app
	mv build/bin/tabpo.app build/mac-arm64/tabpo.app
	@echo "macOSのビルドが完了しました。"

build/linux:
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform linux/amd64 -clean
	mkdir -p build/linux-amd64
	rm -f build/linux-amd64/tabpo
	mv build/bin/tabpo build/linux-amd64/tabpo
	@echo "Linuxのビルドが完了しました。"
