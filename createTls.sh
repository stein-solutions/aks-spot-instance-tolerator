#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-default}
SERVICE=${SERVICE:-mutating-webhook-svc}
SECRET=${SECRET:-mutating-webhook-secret}
CERTDIR=${CERTDIR:-/tmp/webhook-certs}

mkdir -p ${CERTDIR}

openssl genrsa -out ${CERTDIR}/ca.key 2048
openssl req -x509 -new -nodes -key ${CERTDIR}/ca.key -subj "/CN=admission_ca" -days 10000 -out ${CERTDIR}/ca.crt
openssl genrsa -out ${CERTDIR}/server.key 2048
openssl req -new -key ${CERTDIR}/server.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ${CERTDIR}/server.csr
openssl x509 -req -in ${CERTDIR}/server.csr -CA ${CERTDIR}/ca.crt -CAkey ${CERTDIR}/ca.key -CAcreateserial -out ${CERTDIR}/server.crt -days 10000

kubectl create secret tls ${SECRET} \
    --cert=${CERTDIR}/server.crt \
    --key=${CERTDIR}/server.key \
    --namespace=${NAMESPACE} \

