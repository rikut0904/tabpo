FRONTEND_DIR := frontend

.PHONY: install up build/win build/mac
	
install:
	cd $(FRONTEND_DIR) && npm i

up:
	cd $(FRONTEND_DIR) && npm run build
	wails dev

build/win:
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform windows/amd64 -clean
	mkdir -p build/win-amd64
	mv build/bin/tabpo.exe build/win-amd64/tabpo.exe

build/mac:
	cd $(FRONTEND_DIR) && npm run build
	wails build -platform darwin/arm64 -clean
	mkdir -p build/mac-arm64
	mv build/bin/tabpo.app build/mac-arm64/tabpo.app
