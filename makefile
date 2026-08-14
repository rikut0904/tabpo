FRONTEND_DIR := frontend

.PHONY: install up
	
install:
	cd $(FRONTEND_DIR) && npm i

up:
	cd $(FRONTEND_DIR) && npm run build
	wails dev
