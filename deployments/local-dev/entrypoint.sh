#!/bin/sh

swag init -g cmd/url-shortener/main.go
cd /app && air -c .air.toml
