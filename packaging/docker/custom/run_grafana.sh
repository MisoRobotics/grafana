#!/bin/bash -e
/run.sh "cfg:default.server.http_port=${PORT:=8080}" "$@"
