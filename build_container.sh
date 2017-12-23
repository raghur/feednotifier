#! /bin/sh
CGO_ENABLED=0 go build
docker build -t rraghur/feednotifier .
