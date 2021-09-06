# Joyride
### Dynamic DNS for docker containers

Joyride watches for containers starting and stopping via docker events seen on /var/run/docker.sock and if they have the label joyride.host.name=host.example.com it will create a dns entry pointing to the $HOSTIP of the box that is running docker.

Example:
```
HOSTIP=$(ip route get 1 | head -1 | awk '{print $7}')
docker build -t ilude/joyride .
docker run -e HOSTIP=$(HOSTIP) ilude/joyride

# pull and start an example whoami container
docker pull traefik/whoami
docker run -l joyride.host.name=whoami.example.com traefik/whoami

# test if the record was created
dig localhost#54 whoami.example.com
```

Or use the provided cross platform compatible Makefile (linux,macos,windows):
```
# starts container in daemon mode
make 

# stops running container
make down

# tail the docker logs
make logs

# starts container with terminal attached 
# so you can watch the logs
make up

# start up test whoami container
make test
```

docker-compose.yml
```
version: '2.4'
services:
  joyride:
    build:
      context: .
      dockerfile: Dockerfile
    image: ilude/joyride
    container_name: joyride
    restart: unless-stopped
    environment:
      - HOSTIP=${HOSTIP}
    ports:
      - 54:54/udp
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

docker-compose.whoami.yml
```
version: '2.4'
services:
  whoami:
    image: traefik/whoami
    container_name: whoami
    ports:
      - 80:80/tcp
    labels:
      - joyride.host.name=whoami.example.com
```
## How to use
***
Joyride is exposed and runs on port 54 so as not to conflict with local systemd-resolv system. by default it does not forward dns request to another server, instead it is designed to have specific domain request forwarded to it by your main dns server on your network

### For [pihole](https://pi-hole.net/) (or other dnsmasq based dns server)
***
On your pihole/dnsmasq server create the file /etc/dnsmasq.d/03-custom-dns-names.conf
and put something like the following into that file:
```
address=/example.com/192.168.1.2#54
```
address=/\<domain\>/\<ip address of server running joyride\>#\<port number\> 

See [dnsmasq](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html) for addtional options and details.

If you have questions, comments or suggestions, I hang out on [TechnoTim's](https://www.youtube.com/c/TechnoTimLive) discord server: [http://bit.ly/techno-tim-discord](http://bit.ly/techno-tim-discord)