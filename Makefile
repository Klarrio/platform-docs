.PHONY: build push build-no-cache serve-local serve-docker

IMAGE_NAME :=PLACEHOLDER
IMAGE_REPO := PLACEHOLDER
IMAGE_REPO_ACR :=PLACEHOLDER

# Fetch the latest version from Google Artifact Registry
# tr is used to transform the comma-separated tags into a newline-separated list
# sort -V and tail -n 1 are used to get the latest version
LATEST_VERSION := $(shell gcloud container images list-tags $(IMAGE_REPO)/$(IMAGE_NAME) --format='value(tags)' | tr ',' '\n' | sort -V | tail -n 1)

# Increment the last digit of the version
IMAGE_VERSION := $(shell echo $(LATEST_VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}')

# Allow manual override of IMAGE_VERSION via environment or make command
ifdef IMAGE_VERSION_OVERRIDE
IMAGE_VERSION := $(IMAGE_VERSION_OVERRIDE)
endif



# While building locally, you need to pass the token (e.g., make build GIT_TOKEN=yourtoken)
# While building via github workflows, it will get the GIT_TOKEN value assigned there (.github/workflows/makefile.yml)
GIT_TOKEN ?= ""
export GIT_TOKEN
# make a combination of version and the copyright statement for display on site
COPY=Copyright &copy; 2023,
COPY_VERSION=${COPY} Version: ${IMAGE_VERSION}
export COPY_VERSION

# Usage for local builds: make build GIT_TOKEN=yourtoken
build:
	docker build  --build-arg GIT_TOKEN="${GIT_TOKEN}" --build-arg COPY_VERSION="${COPY_VERSION}" -f Dockerfile -t $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION) .

build-no-cache:
	docker build --no-cache --build-arg GIT_TOKEN="${GIT_TOKEN}" --build-arg COPY_VERSION="${COPY_VERSION}" -f Dockerfile -t $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION) .

push:
	# Push to Google Artifact Registry (GAR) (if login is needed locally, uncomment the below line) 
	# gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin PLACEHOLDER
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)

	# Push to Azure Container Registry (ACR) (if login is needed locally, uncomment the below line) 

	# Tag the Docker image for ACR
	docker tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION) $(IMAGE_REPO_ACR)/$(IMAGE_NAME):$(IMAGE_VERSION)

	# Push the Docker image to ACR
	docker push $(IMAGE_REPO_ACR)/$(IMAGE_NAME):$(IMAGE_VERSION)

serve-local:
	rm -rf ~/.gilt/clone
	rm -rf repo*
	gilt overlay
	mkdocs serve

serve-docker:
	# gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin PLACEHOLDER
	@if ! docker images $(IMAGE_REPO)/$(IMAGE_NAME):$(LATEST_VERSION) | awk '{ print $$2 }' | grep -q -F $(LATEST_VERSION); then \
		docker pull $(IMAGE_REPO)/$(IMAGE_NAME):$(LATEST_VERSION); \
	fi
	docker run -p 8080:8088 --rm  $(IMAGE_REPO)/$(IMAGE_NAME):$(LATEST_VERSION)
