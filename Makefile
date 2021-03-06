DOCKER_REGISTRY?=$(shell whoami)
DOCKER_TAG?=latest

.PHONY: all
all: test

.PHONY: test
test:
	bazel test //test/... --test_output=streamed

.PHONY: stress-test
stress-test:
	bazel test //test/... --test_output=streamed --runs_per_test=10

.PHONY: gofmt
gofmt:
	gofmt -w -s cmd/ pkg/

.PHONY: goimports
goimports:
	goimports -w cmd/ pkg/ test/


.PHONY: image-etcd-manager
image-etcd-manager:
	bazel build //images:*
	bazel run //images:etcd-manager
	docker tag bazel/images:etcd-manager ${DOCKER_REGISTRY}/etcd-manager:${DOCKER_TAG}

.PHONY: push-etcd-manager
push-etcd-manager: image-etcd-manager
	docker push ${DOCKER_REGISTRY}/etcd-manager:${DOCKER_TAG}

.PHONY: image-etcd-dump
image-etcd-dump:
	bazel build //images:*
	bazel run //images:etcd-dump
	docker tag bazel/images:etcd-dump ${DOCKER_REGISTRY}/etcd-dump:${DOCKER_TAG}

.PHONY: push-etcd-dump
push-etcd-dump: image-etcd-dump
	docker push ${DOCKER_REGISTRY}/etcd-dump:${DOCKER_TAG}

.PHONY: push
push: push-etcd-manager push-etcd-dump
	echo "pushed images"

.PHONY: gazelle
gazelle:
	bazel run //:gazelle
	git checkout -- vendor
	rm -f vendor/github.com/coreos/etcd/cmd/etcd
	#rm vendor/github.com/golang/protobuf/protoc-gen-go/testdata/multi/BUILD.bazel

.PHONY: dep-ensure
dep-ensure:
	dep ensure -v
	find vendor/ -name "BUILD" -delete
	find vendor/ -name "BUILD.bazel" -delete
	bazel run //:gazelle -- -proto disable
	rm -f vendor/github.com/coreos/etcd/cmd/etcd
