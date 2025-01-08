#!/bin/sh

wget http://10.62.210.66:9900/dip/upx-server-docker-tools-linux-amd64
chmod +x upx-server-docker-tools-linux-amd64
systemctl stop docker-tools.service
mv upx-server-docker-tools-linux-amd64 docker-tools
systemctl start docker-tools.service
