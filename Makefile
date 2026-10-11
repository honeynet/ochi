.PHONY: build local clean deploy-helper deploy .require-deploy-host

build:
	npm run build
	go build -o server .

local:
	./server

clean:
	rm -f server
	rm -rf public/build .routify

# Local-only deploy target; see .env.deploy.example (file is gitignored).
-include .env.deploy

REMOTE_DIR ?= /opt/ochi
HELPER := redeploy-ochi.sh

.require-deploy-host:
	@test -n "$(DEPLOY_HOST)" || (echo 'DEPLOY_HOST unset; copy .env.deploy.example to .env.deploy' >&2; exit 1)

# One-time: install scripts/$(HELPER) on the host.
deploy-helper: .require-deploy-host
	scp scripts/$(HELPER) $(DEPLOY_HOST):$(REMOTE_DIR)/$(HELPER)
	ssh $(DEPLOY_HOST) 'chmod +x $(REMOTE_DIR)/$(HELPER)'

# Build, upload to /tmp (avoids ETXTBSY), then restart the screen session.
deploy: .require-deploy-host clean build
	scp server $(DEPLOY_HOST):/tmp/ochi.new
	ssh $(DEPLOY_HOST) $(REMOTE_DIR)/$(HELPER)
