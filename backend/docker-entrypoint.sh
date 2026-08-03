#!/bin/sh
set -e

echo "== Application des migrations =="
./migrate

echo "== Serveur : port ${PORT:-8080} =="
exec ./server
