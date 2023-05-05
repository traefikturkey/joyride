# syntax=docker/dockerfile:1.3-labs 
FROM ruby:3-alpine as base

LABEL maintainer="Mike Glenn <mglenn@ilude.com>"
LABEL ilude-project=joyride

RUN \
  mkdir -p /app /etc \
  && { \
  echo 'install: --no-document'; \
  echo 'update: --no-document'; \
  } >> /etc/gemrc \
  && apk add --no-cache \
  bash \
  dnsmasq \
  shadow \
  tzdata \
  && rm -rf \
    /usr/lib/ruby/gems/*/cache/* \
    /root/.gem/ \
    /var/cache/apk/* \
    /tmp/* \
    /var/tmp/*

ENV APP /app
ENV GEM_HOME /gems

WORKDIR $APP

COPY --chmod=755 <<-"EOF" /usr/local/bin/docker-entrypoint.sh
#!/bin/sh
set -e

exec $@
EOF

EXPOSE 54/udp

ENTRYPOINT ["docker-entrypoint.sh"]
CMD [ "bundle", "exec", "ruby", "joyride.rb" ]

##############################
# Begin builder
##############################
FROM base AS builder

RUN apk --no-cache add \
  build-base \
  git

COPY ./app/Gemfile $APP
#COPY ./app/Gemfile.lock $APP
RUN bundle install

##############################
# Begin production 
##############################
FROM base AS production

COPY --from=builder ${GEM_HOME} ${GEM_HOME}
COPY ./app $APP
