#!/usr/bin/bash
set -euo pipefail

for agent in node-agent cluster-agent cluster-checks; do
    kubectl label nodes --selector benchmark.datadoghq.com/agent=$agent node-role.kubernetes.io/$agent=$agent
done

for variant in baseline comparison; do
    kubectl label nodes --selector benchmark.datadoghq.com/variant=$variant node-role.kubernetes.io/$variant=$variant
done
