docker-compose -f docker-compose-cli.yaml down
docker rm -f `docker ps -qa`
docker ps -a
rm -rf /var/cetc/production
ls /var/cetc
