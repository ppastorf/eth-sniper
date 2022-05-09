.PHONY: compile
.PHONY: build-image
.PHONY: build
.PHONY: deploy

ROOT_DISK_FILE=/dev/mapper/vgregolith-root
CHAIN_DATA_DIR=.volume/ethereum/geth/chaindata

build:
	@go build -o bin/sniper cmd/main.go
run-test:
	@bin/sniper config_test.yaml
watch-node:
	@watch -t '{ docker-compose top; echo '---'; sudo df -h $(ROOT_DISK_FILE); echo '---'; sudo du -h $(CHAIN_DATA_DIR); }'
