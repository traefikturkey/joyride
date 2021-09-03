FROM ruby:3.0.2-alpine3.14

LABEL maintainer="Mike Glenn <mglenn@ilude.com>"
LABEL ilude-project=joyride

RUN \
  mkdir -p /app /usr/local/etc \
  && { \
  echo 'install: --no-document'; \
  echo 'update: --no-document'; \
  } >> /usr/local/etc/gemrc \
  && apk add --no-cache \
    bash \
    dnsmasq \
    tzdata \
  && update-ca-certificates \
  && gem install bundler:1.17.3 \
  && rm -rf \
    /usr/lib/ruby/gems/*/cache/* \
    /root/.gem/ \
    /var/cache/apk/* \
    /tmp/* \
    /var/tmp/*

EXPOSE 53/udp


   