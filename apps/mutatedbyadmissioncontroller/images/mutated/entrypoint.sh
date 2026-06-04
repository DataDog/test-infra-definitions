#!/bin/sh

printf 'DD_DOGSTATSD_URL:   %s\n' "${DD_DOGSTATSD_URL:-❌ not set}"
printf 'DD_TRACE_AGENT_URL: %s\n' "${DD_TRACE_AGENT_URL:-❌ not set}"
printf 'DD_ENTITY_ID:       %s\n' "${DD_ENTITY_ID:-❌ not set}"
printf 'DD_ENV:             %s\n' "${DD_ENV:-❌ not set}"
printf 'DD_SERVICE:         %s\n' "${DD_SERVICE:-❌ not set}"
printf 'DD_VERSION:         %s\n' "${DD_VERSION:-❌ not set}"
printf '\n'

ls -la /var/run/datadog

sleep infinity
