.PHONY: compile
.PHONY: build-image
.PHONY: build
.PHONY: deploy

ROOT_DISK_FILE=/dev/mapper/vgregolith-root
CHAIN_DATA_DIR=.volume/ethereum/geth/chaindata

node-up:
	@
watch-node:
	@watch -t '{ docker-compose top; echo '---'; sudo df -h $(ROOT_DISK_FILE); echo '---'; sudo du -h $(CHAIN_DATA_DIR); }'
