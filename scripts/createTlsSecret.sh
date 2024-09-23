#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-default}
SECRET=${SECRET:-aks-spot-instance-tolerator-webhook}
CERTDIR=${CERTDIR:-webhook-certs}

kubectl create secret tls ${SECRET} \
    --cert=${CERTDIR}/server.crt \
    --key=${CERTDIR}/server.key \
    --namespace=${NAMESPACE} \
