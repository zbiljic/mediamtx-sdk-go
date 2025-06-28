OGEN := mise x -- ogen

.PHONY: ogen-generate
ogen-generate: tmp/mediamtx/openapi.yaml
	@$(OGEN) \
		-clean \
		-config $(PROJECT_ROOT)/tools/.ogen.yml \
		-package mediamtx \
		-target $(PROJECT_ROOT) \
		$(PROJECT_ROOT)/tmp/mediamtx/$(MEDIAMTX_VERSION)/openapi.yaml

.PHONY: ogen-clean
ogen-clean: # Cleans generated OpenAPI code
	@rm -f $(if $(VERBOSE),-v) $(PROJECT_ROOT)/oas_*_gen.go
