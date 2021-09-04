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
  && ln -snf /etc/localtime /usr/share/zoneinfo/$TZ && echo $TZ > /etc/timezone \
  && rm -rf \
    /usr/lib/ruby/gems/*/cache/* \
    /root/.gem/ \
    /var/cache/apk/* \
    /tmp/* \
    /var/tmp/*

WORKDIR /app

COPY ./app/Gemfile /app/Gemfile
RUN bundle install

COPY *-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/*-entrypoint.sh 

COPY ./app /app

EXPOSE 53/udp

ENTRYPOINT ["docker-entrypoint.sh"]
CMD [ "bundle", "exec", "ruby", "joyride.rb" ]
   