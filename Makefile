# 环境检查
ifeq ($(shell which jq),)
    $(error "jq is not installed, please install jq or set version manually.")
endif
# 配置
SERVICE_CONFIG_FILE := service.json
VERSION := $(shell jq -r '.version' $(SERVICE_CONFIG_FILE) 2>/dev/null)
ENV := $(shell jq -r '.env' $(SERVICE_CONFIG_FILE) 2>/dev/null)
GIT_COMMIT_HASH := $(shell git rev-parse --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
NEW_VERSION := $(if $(GIT_TAG),$(GIT_TAG)-$(GIT_COMMIT_HASH),$(GIT_COMMIT_HASH))
CURRENT_DIR := $(shell pwd)
OUTPUT_DIR := $(shell mkdir -p build && cd build && pwd)
APP_ID := cn.org.ki3.savt-client
ICON := icon.ico
PACKAGE_NAME := savt-client

OS_RAW := $(strip $(shell uname -s | tr '[:upper:]' '[:lower:]'))
OS := $(strip $(if $(findstring darwin,$(OS_RAW)),mac,$(OS_RAW)))
OS := $(strip $(if $(findstring cygwin_nt,$(OS_RAW)),windows,$(OS)))

ARCH_RAW := $(strip $(shell uname -m))
ARCH := $(strip $(if $(filter x86_64,$(ARCH_RAW)),amd64,$(ARCH_RAW)))
ARCH := $(strip $(if $(filter arm64 aarch64,$(ARCH)),arm64,$(ARCH)))

# 支持的操作系统和架构列表
SUPPORTED_ARCHS := arm64 amd64 arm 386
SUPPORTED_OS := windows mac linux

# 检查当前操作系统是否被支持
ifneq ($(filter $(OS),$(SUPPORTED_OS)), $(OS))
$(error Unsupported operating system: '$(OS)'. Supported operating systems: $(SUPPORTED_OS))
endif

# 检查当前架构是否被支持
ifneq ($(filter $(ARCH),$(SUPPORTED_ARCHS)), $(ARCH))
$(error Unsupported architecture: '$(ARCH)'. Supported architectures: $(SUPPORTED_ARCHS))
endif


.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  all		  Build all targets"
	@echo "  clean		Clean build artifacts"
	@echo "  build		Build binaries for the current platform and architecture"
	@echo "  package	  Package binaries for the current platform and architecture"
	@echo "  deploy	   Deploy packaged binaries for the current platform and architecture"


.PHONY: version
version:
	@echo "Debug: NEW_VERSION = $(NEW_VERSION)"
	@echo "Updating version to $(NEW_VERSION) in $(SERVICE_CONFIG_FILE)"
	@sed -i '' 's#"version": *"[^"]*"#"version": "$(NEW_VERSION)"#' $(SERVICE_CONFIG_FILE)
	@echo "Version: $(NEW_VERSION) written to $(SERVICE_CONFIG_FILE)"
.PHONY: env-test
env-test:
	@echo "Setting environment to 'test' in $(SERVICE_CONFIG_FILE)"
	@sed -i '' 's#"env": *"[^"]*"#"env": "test"#' $(SERVICE_CONFIG_FILE)
	@echo "Environment set to 'test' in $(SERVICE_CONFIG_FILE)"

.PHONY: env-production
env-production:
	@echo "Setting environment to 'production' in $(SERVICE_CONFIG_FILE)"
	@sed -i '' 's#"env": *"[^"]*"#"env": "production"#' $(SERVICE_CONFIG_FILE)
	@echo "Environment set to 'production' in $(SERVICE_CONFIG_FILE)"
.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean completed."

.PHONY: lint
lint:
	@echo "Running lint checks..."
	@chmod +x lint.sh
	@./lint.sh
	@echo "Lint checks completed."

.PHONY: build
build: lint build-$(OS)
build-mac:
	@echo "Building for macOS $(ARCH)..."
	mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/
	cp $(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/$(ARCH)/
	cd savt-client-worker && go build -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-worker-$(VERSION)-$(OS)-$(ARCH)
	cd savt-client-cli && go build -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-cli-$(VERSION)-$(OS)-$(ARCH)
	cd savt-client-gui && go build -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-gui-$(VERSION)-$(OS)-$(ARCH)
build-linux:
	@echo "Building for linux $(ARCH)..."
	mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/
	cp $(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/$(ARCH)/
	cd savt-client-worker && go build -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-worker-$(VERSION)-$(OS)-$(ARCH)
	cd savt-client-cli && go build -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-cli-$(VERSION)-$(OS)-$(ARCH)

build-windows: build-windows-arm64 build-windows-amd64
build-windows-arm64:
	@echo "Building for windows arm64 ..."
	mkdir -p $(OUTPUT_DIR)/$(OS)/arm64/
	cp icon.ico $(OUTPUT_DIR)/$(OS)/arm64/
	cp $(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/arm64/
	powershell.exe -Command "cd savt-client-worker;   go build -o ../build/windows/arm64/savt-client-worker-$(VERSION)-$(OS)-arm64.exe"
	powershell.exe -Command "cd savt-client-cli;	  go build -o ../build/windows/arm64/savt-client-cli-$(VERSION)-$(OS)-arm64.exe"
	powershell.exe -Command "cd savt-client-gui;	  go build -ldflags=\"-H windowsgui\" -o ../build/windows/arm64/savt-client-gui-$(VERSION)-$(OS)-arm64.exe"
build-windows-amd64:
	@echo "Building for windows amd64 ..."
	mkdir -p $(OUTPUT_DIR)/$(OS)/amd64/
	cp icon.ico $(OUTPUT_DIR)/$(OS)/amd64/
	cp $(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/amd64/
	powershell.exe -Command "cd savt-client-worker;   go build -o ../build/windows/amd64/savt-client-worker-$(VERSION)-$(OS)-amd64.exe"
	powershell.exe -Command "cd savt-client-cli;   	 go build -o ../build/windows/amd64/savt-client-cli-$(VERSION)-$(OS)-amd64.exe"
	powershell.exe -Command "cd savt-client-gui;	  go build -ldflags=\"-H windowsgui\" -o ../build/windows/amd64/savt-client-gui-$(VERSION)-$(OS)-amd64.exe"


.PHONY: package
package: package-$(OS)
package-mac:
	@if ! command -v create-dmg >/dev/null 2>&1; then \
		echo "Error: 'create-dmg' is not installed. Please install it using 'brew install create-dmg'."; \
		exit 1; \
	fi
	@echo "Packaging for mac $(ARCH)..."
	@mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/MacOS
	@mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/Resources
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-gui-$(VERSION)-$(OS)-$(ARCH) $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/MacOS/savt-client-gui
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-worker-$(VERSION)-$(OS)-$(ARCH) $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/MacOS/savt-client-worker
	@sed \
		-e 's/{{CFBundleExecutable}}/savt-client-gui/' \
		-e 's/{{CFBundleIdentifier}}/$(APP_ID)/' \
		-e 's/{{CFBundleName}}/savt-client/' \
		-e 's/{{CFBundleVersion}}/$(VERSION)/' \
		-e 's/{{CFBundleIconFile}}/savt-client.icns/' \
		savt-client-ci/$(OS)/Info.plist.template > $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/Info.plist
	cp savt-client.icns $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/Resources/savt-client.icns
#	iconutil -c icns -o $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/Resources/savt-client.icns $(ICON)
	cp $(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/Applications/savt-client.app/Contents/MacOS/
	# 提供 安装包
	pkgbuild --identifier $(APP_ID) \
			 --version $(VERSION) \
			 --root "${OUTPUT_DIR}/$(OS)/${ARCH}/app" \
			 --scripts ./savt-client-ci/mac/scripts/install \
			 $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).pkg
	# 提供 卸载包
	pkgbuild --identifier com.example.savt-client.uninstall \
             --version $(VERSION) \
             --scripts ./savt-client-ci/mac/scripts/uninstall \
             --nopayload \
             $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-uninstall-$(VERSION)-$(OS)-$(ARCH).pkg

	@mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).pkg $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents/
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-uninstall-$(VERSION)-$(OS)-$(ARCH).pkg $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents/
	hdiutil create -volname "savt-client" \
	               -srcfolder $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents \
	               -ov -format UDZO \
	               $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).dmg
#	create-dmg \
#	  --volname "savt-client" \
#	  --window-size 600 400 \
#	  --icon-size 128 \
#	  --icon "savt-client-$(VERSION)-$(OS)-$(ARCH).pkg" 150 150 \
#	  --icon "savt-client-uninstall-$(VERSION)-$(OS)-$(ARCH).pkg" 450 150 \
#	  --background "savt-client-ci/$(OS)/resources/background.png" \
#	  --skip-jenkins \
#	  $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).dmg \
#	  $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents/ >/dev/null 2>&1
    # 清理不必要的文件
	rm -rf $(OUTPUT_DIR)/$(OS)/$(ARCH)/dmg_contents
	rm -rf $(OUTPUT_DIR)/$(OS)/$(ARCH)/app
	@echo "Packaging completed: $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).dmg"

package-linux:
	@echo "Packaging for linux $(ARCH)..."
	mkdir -p $(OUTPUT_DIR)/$(OS)/$(ARCH)/app
	cp $(ICON) $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/
	cp savt-client-ci/$(OS)/* $(OUTPUT_DIR)/$(OS)/$(ARCH)/app
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/$(SERVICE_CONFIG_FILE) $(OUTPUT_DIR)/$(OS)/$(ARCH)/app
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-worker-* $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/savt-client-worker
	cp $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-cli-* $(OUTPUT_DIR)/$(OS)/$(ARCH)/app/savt-client-cli
	cd $(OUTPUT_DIR)/$(OS)/$(ARCH)/ && \
	mkdir -p savt-client-$(VERSION)-$(OS)-$(ARCH) && \
	mv app/* savt-client-$(VERSION)-$(OS)-$(ARCH)/ && \
	tar -czvf savt-client-$(VERSION)-$(OS)-$(ARCH).tar.gz savt-client-$(VERSION)-$(OS)-$(ARCH) && rm -rf savt-client-$(VERSION)-$(OS)-$(ARCH)


package-windows: package-windows-arm64 package-windows-amd64
package-windows-arm64:
	@sed -e 's/#define MyAppVersion ".*"/#define MyAppVersion "$(VERSION)"/' \
		 -e 's/#define ARCH ".*"/#define ARCH "arm64"/' \
		 savt-client-ci/windows/windows_template.iss > $(OUTPUT_DIR)/windows/arm64/windows_arm64.iss
	cp -r savt-client-ci/windows/* $(OUTPUT_DIR)/windows/arm64/
	CYGWIN_PATH="$(OUTPUT_DIR)" && \
	WINDOWS_PATH=$$(cygpath -w "$$CYGWIN_PATH") && \
	echo "Cygwin Path: $$CYGWIN_PATH" && \
	echo "Windows Path: $$WINDOWS_PATH" && \
	powershell.exe -Command "ISCC.exe '$$WINDOWS_PATH\\windows\\arm64\\windows_arm64.iss'"
package-windows-amd64:
	@sed -e 's/#define MyAppVersion ".*"/#define MyAppVersion "$(VERSION)"/' \
		 -e 's/#define ARCH ".*"/#define ARCH "amd64"/' \
		 savt-client-ci/windows/windows_template.iss > $(OUTPUT_DIR)/windows/amd64/windows_amd64.iss
	cp -r savt-client-ci/windows/* $(OUTPUT_DIR)/windows/amd64/
	CYGWIN_PATH="$(OUTPUT_DIR)" && \
	WINDOWS_PATH=$$(cygpath -w "$$CYGWIN_PATH") && \
	echo "Cygwin Path: $$CYGWIN_PATH" && \
	echo "Windows Path: $$WINDOWS_PATH" && \
	powershell.exe -Command "ISCC.exe '$$WINDOWS_PATH\\windows\\amd64\\windows_amd64.iss'"

.PHONY: deploy
deploy:
	@make deploy-$(OS)
deploy-mac:
	s3cmd put --recursive $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).dmg s3://ki3-frontend-static/${ENV}/sav/dist/savt-client-$(VERSION)/
deploy-linux:
	s3cmd put --recursive $(OUTPUT_DIR)/$(OS)/$(ARCH)/savt-client-$(VERSION)-$(OS)-$(ARCH).tar.gz s3://ki3-frontend-static/${ENV}/sav/dist/savt-client-$(VERSION)/

deploy-windows: deploy-windows-arm64 deploy-windows-amd64
deploy-windows-arm64:
	s3cmd put --recursive $(OUTPUT_DIR)/windows/arm64/savt-client-*.exe s3://ki3-frontend-static/${ENV}/sav/dist/savt-client-$(VERSION)/
deploy-windows-amd64:
	s3cmd put --recursive $(OUTPUT_DIR)/windows/amd64/savt-client-*.exe s3://ki3-frontend-static/${ENV}/sav/dist/savt-client-$(VERSION)/

.PHONY: flow
flow: clean build package deploy
	@echo "Flow process completed successfully!"



.PHONY: deploy_doc
deploy_doc:
	s3cmd put --recursive savt-client-doc/* s3://ki3-frontend-static/${ENV}/sav/

.PHONY: release_doc
release_doc: clean
	mkdir -p $(OUTPUT_DIR)/release
	bash savt-client-ci/update_save_client_info.sh $(VERSION) $(ENV) $(OUTPUT_DIR)/release
	s3cmd put --recursive savt-client-doc/* s3://ki3-frontend-static/${ENV}/sav/
	s3cmd put $(OUTPUT_DIR)/release/save_client_info.json s3://ki3-frontend-static/${ENV}/sav/






