# Joyride
## Dynamic DNS for docker containers

Joyride watches for containers starting and stopping via docker events seen on /var/run/docker.sock and if they have the label joyride.host.name=host.example.com it will create a dns entry pointing to the $HOSTIP of the box that is running docker.

Example:
```
HOSTIP=$(ip route get 1 | head -1 | awk '{print $7}')
docker pull ghcr.io/ilude/joyride:latest
docker run -e HOSTIP=$(HOSTIP) ghcr.io/ilude/joyride

# pull and start an example whoami container
docker pull traefik/whoami
docker run -l joyride.host.name=whoami.example.com traefik/whoami

# test if the record was created
dig localhost#54 whoami.example.com
```

docker-compose.yml
```
version: '2.4'
services:
  joyride:
    image: ghcr.io/ilude/joyride:latest
    restart: unless-stopped
    environment:
      # run the following before docker-compose
      # export HOSTIP=$(ip route get 1 | head -1 | awk '{print $7}')
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
Joyride is exposed and runs on port 54 so as not to conflict with local systemd-resolv system. by default it does not forward dns request to another server, instead it is designed to have specific domain request forwarded to it by your main dns server on your network
***
### [PiHole](https://pi-hole.net/) / [dnsmasq](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html) : Customize and run the following command on your pihole server
```bash
echo "server=/example.com/192.168.1.2#54" | sudo tee -a /etc/dnsmasq.d/03-custom-dns-names.conf
```
#### example.com - replac with your domain
#### 192.168.1.2 - replace with the ip address of the server where joyride is running
***
If you have questions, comments or suggestions, We hang out on [TechnoTim's](https://www.youtube.com/c/TechnoTimLive) [discord server](http://bit.ly/techno-tim-discord)

[Joyride](https://github.com/traefikturkey/joyride) is a [TraefikTurkey Project](https://github.com/traefikturkey) © 2024
