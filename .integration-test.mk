# Integration Test Quick Reference
# Include this in your shell or add to main Makefile

# Convenience aliases that forward to the real Makefile
it-up:
	@$(MAKE) -f Makefile.integration-test integration-test-up

it-down:
	@$(MAKE) -f Makefile.integration-test integration-test-down

it-setup:
	@$(MAKE) -f Makefile.integration-test integration-test-setup

it-force-finish:
	@$(MAKE) -f Makefile.integration-test integration-test-force-finish

it-inspect:
	@$(MAKE) -f Makefile.integration-test integration-test-inspect

it-full:
	@$(MAKE) -f Makefile.integration-test integration-test-full

it-clean:
	@$(MAKE) -f Makefile.integration-test integration-test-clean

it-help:
	@$(MAKE) -f Makefile.integration-test help
