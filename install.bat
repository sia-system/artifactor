@echo off
set GOARCH=amd64
set GOOS=linux
go build
move artifactor ../../config-registry/services/artifactor/docker-content/bin