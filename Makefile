# OPD — Makefile

BIN     := opd
SRC_DIR := ./cli

ifeq ($(OS),Windows_NT)
    BIN         := opd.exe
    INSTALL_DIR := $(USERPROFILE)\AppData\Local\Programs\opd
    INSTALL_CMD := powershell -Command "New-Item -ItemType Directory -Force -Path '$(INSTALL_DIR)' | Out-Null; Copy-Item '$(SRC_DIR)/$(BIN)' '$(INSTALL_DIR)/$(BIN)' -Force"
else
    INSTALL_DIR := /usr/local/bin
    INSTALL_CMD := install -m 0755 $(SRC_DIR)/$(BIN) $(INSTALL_DIR)/$(BIN)
endif

.PHONY: build install clean

build:
	cd $(SRC_DIR) && go mod tidy && go build -o $(BIN) .

install: build
	$(INSTALL_CMD)
	@echo "Installed opd to $(INSTALL_DIR)"

clean:
	rm -f $(SRC_DIR)/opd $(SRC_DIR)/opd.exe
