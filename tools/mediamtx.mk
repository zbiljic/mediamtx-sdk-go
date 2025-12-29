# renovate: datasource=github-releases depName=bluenviron/mediamtx
MEDIAMTX_VERSION ?= v1.15.6

tmp/mediamtx: tmp
	mkdir -p $@

tmp/mediamtx/$(MEDIAMTX_VERSION): tmp/mediamtx
	mkdir -p $@

tmp/mediamtx/$(MEDIAMTX_VERSION)/openapi.yaml: tmp/mediamtx/$(MEDIAMTX_VERSION)
	echo "Downloading MediaMTX '$(MEDIAMTX_VERSION)' openapi.yaml"
	curl -sSL \
		https://github.com/bluenviron/mediamtx/raw/refs/tags/$(MEDIAMTX_VERSION)/apidocs/openapi.yaml \
		--output $(PROJECT_ROOT)/$@

.PHONY: tmp/mediamtx/openapi.yaml
tmp/mediamtx/openapi.yaml: tmp/mediamtx/$(MEDIAMTX_VERSION)/openapi.yaml
