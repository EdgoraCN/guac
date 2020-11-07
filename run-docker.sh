docker rm -f guac
docker run -d --name guac --restart=always -p 4567:4567 \
-e GUACD=192.168.2.120:4822 \
edgora/guac:1.3.2