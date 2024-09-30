#!/bin/bash
NAMESPACE=${NAMESPACE:-"automq-operator"}
HELM_OPTS=${HELM_OPTS:-""}
helm upgrade --install automq-operator charts/automq-operator --namespace "${NAMESPACE}" --create-namespace ${HELM_OPTS}
