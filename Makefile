export VERSION ?= $(shell (scripts/version.sh 2>/dev/null || echo "dev") | sed -e 's/^v//g')
export REVISION ?= $(shell git rev-parse --short HEAD || echo "unknown")
export BRANCH ?= $(shell git show-ref | grep "$(REVISION)" | grep -v HEAD | awk '{print $$2}' | sed 's|refs/remotes/origin/||' | sed 's|refs/heads/||' | sort | head -n 1)
export BUILT ?= $(shell date -u +%Y-%m-%dT%H:%M:%S%z)
export LATEST_STABLE_TAG := $(shell git -c versionsort.prereleaseSuffix="-rc" tag -l "v*.*.*" --sort=-v:refname | awk '!/rc/' | head -n 1)
export CGO_ENABLED ?= 0
export GOPATH ?= ".go/"
export BUILD_PLATFORMS ?= -osarch 'linux/amd64 linux/386 linux/arm linux/arm64'

export goJunitReport ?= $(GOPATH)/bin/go-junit-report
MOCKERY_VERSION = 1.1.0
MOCKERY = .tmp/mockery-$(MOCKERY_VERSION)
gox ?= $(GOPATH)/bin/gox

RELEASE_INDEX_GEN_VERSION ?= master
releaseIndexGen ?= .tmp/release-index-gen-$(RELEASE_INDEX_GEN_VERSION)

PKG := $(shell go list .)
PKGs := $(shell go list ./... | grep -vE "^/vendor/")

GO_LDFLAGS := -X $(PKG).VERSION=$(VERSION) \
              -X $(PKG).REVISION=$(REVISION) \
              -X $(PKG).BRANCH=$(BRANCH) \
              -X $(PKG).BUILT=$(BUILT) \
              -s -w

.PHONY: compile
compile:
	go build \
			-o build/fargate \
			-ldflags "$(GO_LDFLAGS)" \
			./cmd/fargate

.PHONY: compile_all
compile_all: $(gox)
	# Building project in version $(VERSION) for $(BUILD_PLATFORMS)
	$(gox) $(BUILD_PLATFORMS) \
			-ldflags "$(GO_LDFLAGS)" \
			-output="build/fargate-{{.OS}}-{{.Arch}}" \
			./cmd/fargate

export testsDir = ./.tests

.PHONY: tests
tests: $(testsDir) $(goJunitReport)
	@./scripts/tests.sh normal

.PHONY: tests_race
tests_race: $(testsDir) $(goJunitReport)
	@./scripts/tests.sh race

.PHONY: lint
lint: OUT_FORMAT ?= colored-line-number
lint: LINT_FLAGS ?=
lint:
	@golangci-lint run ./... --out-format $(OUT_FORMAT) $(LINT_FLAGS)

$(testsDir):
	# Preparing tests output directory
	@mkdir -p $@

.PHONY: fmt
fmt:
	# Fixing project code formatting...
	@go fmt $(PKGs) | awk '{if (NF > 0) {if (NR == 1) print "Please run go fmt for:"; print "- "$$1}} END {if (NF > 0) {if (NR > 0) exit 1}}'

.PHONY: mocks
mocks: $(MOCKERY)
	# Removing existing mocks
	@find * -name "mock_*.go" -delete
	# Generating new mocks
	@$(MOCKERY) -recursive -all -inpkg -dir ./

.PHONY: check_mocks
check_mocks:
	# Checking if mocks are up-to-date
	@$(MAKE) mocks
	# Checking the differences
	@git --no-pager diff --compact-summary --exit-code -- ./helpers/service/mocks \
		$(shell git ls-files | grep 'mock_' | grep -v 'vendor/') && \
		echo "Mocks up-to-date!"

.PHONY: prepare_ci_image
prepare_ci_image: CI_IMAGE ?= fargate-ci-image
prepare_ci_image: CI_REGISTRY ?= ""
prepare_ci_image:
	$(MAKE) prepare_image IMAGE_NAME=$(CI_IMAGE) IMAGE_PATH=ci CI_REGISTRY=$(CI_REGISTRY)

.PHONY: prepare_ssh_service_image
prepare_ssh_service_image: SSH_SERVICE_IMAGE ?= fargate-ssh-service-image
prepare_ssh_service_image: CI_REGISTRY ?= ""
prepare_ssh_service_image:
	$(MAKE) prepare_image IMAGE_NAME=$(SSH_SERVICE_IMAGE) IMAGE_PATH=ssh_service CI_REGISTRY=$(CI_REGISTRY)

.PHONY: prepare_image
prepare_image: CI_REGISTRY ?= ""
prepare_image:
	# Builiding the $(IMAGE_NAME) image
	@docker build \
		--pull \
		--no-cache \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg ALPINE_VERSION=$(ALPINE_VERSION) \
		-t $(IMAGE_NAME) \
		-f dockerfiles/$(IMAGE_PATH)/Dockerfile dockerfiles/$(IMAGE_PATH)/
ifneq ($(CI_REGISTRY),)
	# Pushing the $(IMAGE_NAME) image to $(CI_REGISTRY)
	@docker login --username $${CI_REGISTRY_USER} --password $${CI_REGISTRY_PASSWORD} $(CI_REGISTRY)
	@docker push $(IMAGE_NAME)
	@docker logout $(CI_REGISTRY)
else
	# No CI_REGISTRY value, skipping image push
endif

.PHONY: release_s3
release_s3: CI_COMMIT_REF_NAME ?= $(BRANCH)
release_s3: CI_COMMIT_SHA ?= $(REVISION)
release_s3: S3_BUCKET ?=
release_s3:
	@$(MAKE) index_file
ifneq ($(S3_BUCKET),)
	@$(MAKE) sync_s3_release S3_URL="s3://$(S3_BUCKET)/$(CI_COMMIT_REF_NAME)"
ifeq ($(shell git describe --exact-match --match $(LATEST_STABLE_TAG) >/dev/null 2>&1; echo $$?), 0)
	@$(MAKE) sync_s3_release S3_URL="s3://$(S3_BUCKET)/latest";
endif
	@$(MAKE) release_gitlab
endif

.PHONY: sync_s3_release
sync_s3_release: S3_URL ?=
sync_s3_release:
	# Syncing with $(S3_URL)
	@aws s3 sync build "$(S3_URL)" --acl public-read

.PHONY: remove_s3_release
remove_s3_release: CI_COMMIT_REF_NAME ?= $(BRANCH)
remove_s3_release: S3_BUCKET ?=
remove_s3_release:
ifneq ($(S3_BUCKET),)
	@aws s3 rm "s3://$(S3_BUCKET)/$(CI_COMMIT_REF_NAME)" --recursive
endif

.PHONY: release_gitlab
release_gitlab: export CI_COMMIT_TAG ?=
release_gitlab: export CI_PROJECT_URL ?=
release_gitlab:
ifneq ($(CI_COMMIT_TAG),)
	# Saving as GitLab release at $(CI_PROJECT_URL)/-/releases
	@$(shell ./scripts/gitlab_release.sh)
endif

.PHONY: index_file
index_file: export CI_COMMIT_REF_NAME ?= $(BRANCH)
index_file: export CI_COMMIT_SHA ?= $(REVISION)
index_file: $(releaseIndexGen)
	# generating index.html file
	@$(releaseIndexGen) -working-directory build/ \
                        -project-version $(VERSION) \
                        -project-git-ref $(CI_COMMIT_REF_NAME) \
                        -project-git-revision $(CI_COMMIT_SHA) \
                        -project-name "GitLab Runner - Custom Executor's AWS Fargate driver" \
                        -project-repo-url "https://gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate" \
                        -gpg-key-env GPG_KEY \
                        -gpg-password-env GPG_PASSPHRASE

.PHONY: check_modules
check_modules:
	@git diff go.sum > /tmp/gosum-$${CI_JOB_ID}-before
	@go mod tidy
	@git diff go.sum > /tmp/gosum-$${CI_JOB_ID}-after
	@diff -U0 /tmp/gosum-$${CI_JOB_ID}-before /tmp/gosum-$${CI_JOB_ID}-after

$(MOCKERY): OS_TYPE ?= $(shell uname -s)
$(MOCKERY): DOWNLOAD_URL = "https://github.com/vektra/mockery/releases/download/v$(MOCKERY_VERSION)/mockery_$(MOCKERY_VERSION)_$(OS_TYPE)_x86_64.tar.gz"
$(MOCKERY):
	# Installing $(DOWNLOAD_URL) as $(MOCKERY)
	@mkdir -p $(shell dirname $(MOCKERY))
	@curl -sL "$(DOWNLOAD_URL)" | tar xz -O mockery > $(MOCKERY)
	@chmod +x "$(MOCKERY)"

$(goJunitReport):
	# Installing go-junit-report
	@go get github.com/jstemmer/go-junit-report

$(gox):
	# Installing gox
	@go get github.com/mitchellh/gox

$(releaseIndexGen): OS_TYPE ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
$(releaseIndexGen): DOWNLOAD_URL = "https://storage.googleapis.com/gitlab-runner-tools/release-index-generator/$(RELEASE_INDEX_GEN_VERSION)/release-index-gen-$(OS_TYPE)-amd64"
$(releaseIndexGen):
	# Installing $(DOWNLOAD_URL) as $(releaseIndexGen)
	@mkdir -p $(shell dirname $(releaseIndexGen))
	@curl -sL "$(DOWNLOAD_URL)" -o "$(releaseIndexGen)"
	@chmod +x "$(releaseIndexGen)"
