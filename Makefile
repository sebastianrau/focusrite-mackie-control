BUILD_DIR = build

SRC_FOLDER = cmd/monitor-control/
APP_NAME = monitor-control
APP_ID = "focusrite-mackie-control.sebastianrau.github.com"

GIT_VERSION_TAG=$(shell git describe --tags)
GIT_BUILD=$(shell git rev-parse --short HEAD)
GIT_DATE=$(shell git log -1 --date=format:"%Y/%m/%d" --format="%ad" )

APP_TAGS = "build=${GIT_BUILD}(${GIT_DATE})"

.PHONY: dut app.darwin64 app.darwinArm lint clean distclean mrproper


# Build the project
all:
	@echo "cmd:"
	@echo ""
	@echo "  app            build all app for all os"
	@echo "  app.windows64  build app for win   amd64"
	@echo "  app.darwin64   build app for osx   amd64"
	@echo "  app.darwinArm  build app for osx   arm64"
	@echo "  app.linux64    build app for linux amd64"
	@echo "  app.linuxArm   build app for linux arm64"
	@echo ""
	@echo "  lint           go linter"
	@echo ""
	@echo "  clean          remove dut binarys"


app: app.windows64 app.darwin app.darwinArm app.linux64

app.windows64:
	cd ${BUILD_DIR} && GOARCH=amd64 fyne package -os windows -icon ../../icon.png --src ../${SRC_FOLDER} --appVersion ${GIT_VERSION_TAG} --release --tags ${APP_TAGS} --appID ${APP_ID} --name ${APP_NAME}

app.darwin:
	cd ${BUILD_DIR} && GOARCH=amd64 fyne package -os darwin -icon ../../icon.png --src ../${SRC_FOLDER} --appVersion ${GIT_VERSION_TAG} --release --tags ${APP_TAGS} --appID ${APP_ID} --name ${APP_NAME}
	
app.darwinArm:
	cd ${BUILD_DIR} && GOARCH=arm64 fyne package -os darwin -icon ../../icon.png --src ../${SRC_FOLDER} --appVersion ${GIT_VERSION_TAG} --release --tags ${APP_TAGS} --appID ${APP_ID} --name ${APP_NAME}.arm

app.linux64:
	cd ${BUILD_DIR} && GOARCH=amd64 fyne package -os darwin -icon ../../icon.png --src ../${SRC_FOLDER} --appVersion ${GIT_VERSION_TAG} --release --tags ${APP_TAGS} --appID ${APP_ID} --name ${APP_NAME}	

app.linuxArm:
	cd ${BUILD_DIR} && GOARCH=arm64 fyne package -os darwin -icon ../../icon.png --src ../${SRC_FOLDER} --appVersion ${GIT_VERSION_TAG} --release --tags ${APP_TAGS} --appID ${APP_ID} --name ${APP_NAME}	


streamdeck.icons:
	cd streamdeck/ && zip -vr 'Monitor Control Icons.streamDeckIconPack' com.github.sebastianraufocusrite-mackie-control.sdIconPack/ -x "*.DS_Store"

cli-lint:
	golangci-lint run  cmd/... pkg/...

clean:
	-rm -f ${BUILD_DIR}/*