.DEFAULT_GOAL := help

.PHONY: help
help:
	just --list

.PHONY: %
%:
	just $@
