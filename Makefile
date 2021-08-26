.PHONY = build run

PACKAGE_NAME = github.com/pinheirolucas/discord_instants_player
BIN = ./bin/discord_instants_player

build:
	go build -o ${BIN} ${PACKAGE_NAME}

run: build
	${BIN}

test:
	go test ./...

cover:
	go test -coverprofile cp.out ./...
	go tool cover -html=cp.out

.PHONY = clean
clean:
	go clean
	rm -rf ./bin
	rm --force cp.out
	rm --force nohup.out
