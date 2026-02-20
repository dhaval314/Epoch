#!/bin/bash

# 1. Generate CA's Private Key and Self-Signed Certificate
# This is the "Master Key" that signs everything else.
openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca-key.pem -out ca-cert.pem -subj "/C=IN/ST=Karnataka/L=Bengaluru/O=Epoch/OU=Root/CN=EpochRootCA"

echo "CA's Self-Signed Certificate"
openssl x509 -in ca-cert.pem -noout -text

# 2. Generate Web Server's Private Key and CSR (Certificate Signing Request)
openssl req -newkey rsa:4096 -nodes -keyout server-key.pem -out server-req.pem -subj "/C=IN/ST=Karnataka/L=Bengaluru/O=Epoch/OU=Server/CN=localhost"

# 3. Sign the Server's Request with the CA's Key (Add SANs for Go 1.15+)
# Go requires "Subject Alternative Names" (SAN) for localhost.
openssl x509 -req -in server-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extfile <(printf "subjectAltName=DNS:localhost,IP:0.0.0.0,IP:127.0.0.1")

echo "Server's Signed Certificate"
openssl x509 -in server-cert.pem -noout -text

# 4. Generate Client's Private Key and Certificate (Signed by CA)
openssl req -newkey rsa:4096 -nodes -keyout client-key.pem -out client-req.pem -subj "/C=IN/ST=Karnataka/L=Bengaluru/O=Epoch/OU=Client/CN=client"
openssl x509 -req -in client-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem

echo "Client's Signed Certificate"
openssl x509 -in client-cert.pem -noout -text

# 5. Generate Worker's Private Key and Certificate (Signed by CA)
openssl req -newkey rsa:4096 -nodes -keyout worker-key.pem -out worker-req.pem -subj "/C=IN/ST=Karnataka/L=Bengaluru/O=Epoch/OU=Worker/CN=worker"
openssl x509 -req -in worker-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out worker-cert.pem

echo "Worker's Signed Certificate"
openssl x509 -in worker-cert.pem -noout -text