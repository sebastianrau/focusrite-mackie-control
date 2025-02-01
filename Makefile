BUILD_DIR = build
APP_NAME = focusrite-mackie-control
APP_MAIN = cmd/monitor-control/main.go
MC_DIR = build/bin
LOG_DIR= build/log
GIT_VER=$(shell git rev-parse --short HEAD)
GIT_DATE=$(shell git log -1 --date=format:"%Y/%m/%d" --format="%ad" )

LDFLAGS=-ldflags "-X main.version=${GIT_VER}(${GIT_DATE})"

.PHONY: dut app.darwin64 app.darwinArm lint clean distclean mrproper


# Build the project
all:
	@echo "cmd:"
	@echo ""
	@echo "  app            build all app for all os"
	@echo "  app.windows    build app for win   x86"
	@echo "  app.windows64  build app for win   amd64"
	@echo "  app.darwin64   build app for osx   amd64"
	@echo "  app.darwinArm  build app for osx   arm64"
	@echo "  app.linux64    build app for linux arm64"
	@echo ""
	@echo "  lint           go linter"
	@echo ""
	@echo "  clean          remove dut binarys"
	@echo "  distclean       remove build folder"

app: app.windows64  app.darwin64 app.darwinArm app.linux64

app.windows:
	GOOS=windows GOARCH=386 go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}.exe -v ${APP_MAIN}

app.windows64:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}64.exe -v ${APP_MAIN}

app.darwin64:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-darwin -v ${APP_MAIN}

app.darwinArm:
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-darwin-arm -v ${APP_MAIN}

app.linux64:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${APP_NAME}-linux -v ${APP_MAIN}

lint:
	golint -set_exit_status $(shell go list ./...)

clean:
	-rm -f ${BUILD_DIR}/*

distclean:
	rm -rf ./build
