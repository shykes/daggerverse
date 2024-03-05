FROM spiralscout/roadrunner:2023.3.8 AS roadrunner
FROM php:8.3-cli as base

ARG UNAME=pacha
ARG UID=1000
ARG GID=1000
ARG VERSION="unstable"

ENV TERM xterm
ENV VERSION "${VERSION}"
ENV COMPOSER_ALLOW_SUPERUSER 1
ENV COMPOSER_MEMORY_LIMIT -1

WORKDIR /var/www/framework

# install packages
RUN apt update \
  && apt dist-upgrade -y \
  && apt install --no-install-recommends -y apt-utils \
  && apt install --no-install-recommends -y htop nano curl git zip unzip findutils wget \
  && apt update \
  && apt dist-upgrade -y \
  && apt install --no-install-recommends -y procps wkhtmltopdf libfreetype6-dev libjpeg62-turbo-dev \
  libpng-dev libjpeg-dev libmcrypt-dev libgd-dev jpegoptim optipng pngquant gifsicle pkg-config libssl-dev \
  libcurl4-openssl-dev zlib1g-dev libxslt-dev libicu-dev g++ libxml2-dev libpcre3-dev libzip-dev libsodium-dev \
  libonig-dev gpg gpg-agent librabbitmq-dev imagemagick libmagick++-dev \
  && apt clean \
  && rm -rf /var/lib/apt/lists/* \
  && docker-php-ext-configure gd --enable-gd --with-freetype --with-jpeg \
  && docker-php-ext-install -j$(nproc) gd pcntl iconv curl intl xml xsl mbstring bcmath sodium opcache zip sockets soap \
  && pecl channel-update pecl.php.net \
  && pecl install -of mongodb redis ast apcu amqp excimer \
  && docker-php-ext-enable mongodb opcache redis ast apcu amqp excimer \
  && rm -rf /tmp/* \
  && apt autoremove -y \
  && apt autoclean \
  && rm -rf /var/www/html

COPY --from=roadrunner /usr/bin/rr /usr/bin/rr
COPY --from=composer/composer /usr/bin/composer /usr/bin/composer

# >>> DEVELOPMENT IMAGE
FROM base as dev

ENV PHAN_DISABLE_XDEBUG_WARN 1
ENV PHAN_ALLOW_XDEBUG 0
ENV APP_DEBUG 1
ENV LOCAL_PHP_SECURITY_CHECKER_VERSION 2.0.6
ENV PATH $PATH:/home/pacha/.composer/vendor/bin

RUN apt update \
  && apt dist-upgrade -y \
  && apt install --no-install-recommends -y apache2-utils build-essential libnss3-tools mycli ca-certificates gnupg lsb-release sudo net-tools dnsutils \
  && mkdir -p /etc/apt/keyrings \
  && curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg \
  && echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list \
  && curl -1sLf 'https://dl.cloudsmith.io/public/symfony/stable/setup.deb.sh' | sudo -E bash \
  && apt update \
  && apt dist-upgrade -y \
  && apt install --no-install-recommends -y stripe symfony-cli \
  && docker-php-ext-install -j$(nproc) pdo pdo_mysql \
  && wget https://github.com/fabpot/local-php-security-checker/releases/download/v${LOCAL_PHP_SECURITY_CHECKER_VERSION}/local-php-security-checker_${LOCAL_PHP_SECURITY_CHECKER_VERSION}_linux_$(dpkg --print-architecture) -O /usr/local/bin/local-php-security-checker \
  && chmod +x /usr/local/bin/local-php-security-checker \
  && wget https://raw.githubusercontent.com/marcocesarato/PHP-Antimalware-Scanner/master/dist/scanner -O /usr/local/bin/amscanner \
  && cd

RUN mv "${PHP_INI_DIR}/php.ini-development" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;date.timezone =/date.timezone=UTC/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/display_errors = On/display_errors=0/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/display_startup_errors = On/display_startup_errors=0/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/session.use_strict_mode = 0/session.use_strict_mode=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable=1/opcache.enable=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable_cli=0/opcache.enable_cli=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable_file_override=0/opcache.enable_file_override=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/opcache.validate_timestamps=0/opcache.validate_timestamps=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.preload_user=/opcache.preload_user=www-data/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.memory_consumption=128/opcache.memory_consumption=500/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.max_accelerated_files=10000/opcache.max_accelerated_files=30000/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;realpath_cache_size = 4096K/realpath_cache_size=4096K/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;realpath_cache_size = 120/realpath_cache_ttl=600/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/upload_max_filesize = 2M/upload_max_filesize = 15M/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/post_max_size = 8M/post_max_size = 15M/g" "${PHP_INI_DIR}/php.ini" \
  && echo "zend.detect_unicode=0" >> "${PHP_INI_DIR}/php.ini" \
  && echo "catch_workers_output=1" >> "${PHP_INI_DIR}/php.ini" \
  && echo "decorate_workers_output=0" >> "${PHP_INI_DIR}/php.ini" \
  && echo "xdebug.mode=develop,debug" >> "${PHP_INI_DIR}/php.ini" \
  && echo "xdebug.discover_client_host=0" >> "${PHP_INI_DIR}/php.ini" \
  && echo "xdebug.client_host=host.docker.internal" >> "${PHP_INI_DIR}/php.ini"

RUN groupadd -g ${GID} -o ${UNAME} \
  && useradd -m -u ${UID} -g ${GID} -o -s /bin/bash ${UNAME} \
  && usermod -aG sudo ${UNAME} \
  && echo "${UNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER $UNAME

HEALTHCHECK --interval=1m --timeout=60s --start-period=1s CMD composer hc || exit 1

ENTRYPOINT ["composer"]
CMD ["server:start:dev"]
# <<< DEVELOPMENT IMAGE

# >>> DEVELOPMENT CMD IMAGE
FROM dev as dev-cmd

HEALTHCHECK --interval=1m --timeout=3s --start-period=1s CMD if pgrep -x "php" > /dev/null; then exit 0; else exit 1; fi
# <<< DEVELOPMENT CMD IMAGE

# >>> PRODUCTION IMAGE
FROM base as prod

ARG GIT_COMMIT_HASH="NOT SET ON BUILD"
ARG GITHUB_RUN_NUMBER="NOT SET ON BUILD"

ENV APP_ENV prod
ENV APP_DEBUG 0
ENV GIT_COMMIT_HASH "${GIT_COMMIT_HASH}"
ENV GITHUB_RUN_NUMBER "${GITHUB_RUN_NUMBER}"

RUN mv "${PHP_INI_DIR}/php.ini-production" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;date.timezone =/date.timezone=UTC/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/display_errors = On/display_errors=0/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/display_startup_errors = On/display_startup_errors=0/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/session.use_strict_mode = 0/session.use_strict_mode=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable=1/opcache.enable=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable_cli=1/opcache.enable_cli=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.enable_file_override=0/opcache.enable_file_override=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/opcache.validate_timestamps=0/opcache.validate_timestamps=1/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.preload_user=/opcache.preload_user=www-data/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.memory_consumption=128/opcache.memory_consumption=500/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;opcache.max_accelerated_files=10000/opcache.max_accelerated_files=30000/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;realpath_cache_size = 4096K/realpath_cache_size=4096K/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/;realpath_cache_size = 120/realpath_cache_ttl=600/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/upload_max_filesize = 2M/upload_max_filesize = 15M/g" "${PHP_INI_DIR}/php.ini" \
  && sed -i "s/post_max_size = 8M/post_max_size = 15M/g" "${PHP_INI_DIR}/php.ini" \
  && echo "zend.detect_unicode=0" >> "${PHP_INI_DIR}/php.ini" \
  && echo "catch_workers_output=1" >> "${PHP_INI_DIR}/php.ini" \
  && echo "decorate_workers_output=0" >> "${PHP_INI_DIR}/php.ini" \
  && CAPITAL_APP_ENV=$(/bin/bash -c 'echo "${APP_ENV^}"') \
  && sed -i "s/;opcache.preload=/opcache.preload=\/var\/www\/framework\/var\/cache\/${APP_ENV}\/App_Kernel${CAPITAL_APP_ENV}Container.preload.php/g" "${PHP_INI_DIR}/php.ini"

# copy all
COPY . /var/www
COPY framework/.env /var/www/framework/.env

RUN rm -rf /var/www/framework/features \
  && cp /var/www/framework/config/packages/${APP_ENV}/* /var/www/framework/config/packages/ \
  && find /var/www/framework/config/packages/* -type d -exec rm -rf {} + \
  && mkdir -p /var/www/framework/var/cache \
  && mkdir -p /var/www/framework/var/log \
  && rm -rf /var/www/framework/var/cache/* \
  && rm -rf /var/www/framework/var/log/* \
  && composer install -o -a --no-dev \
  && composer dump-autoload -o -a --no-dev \
  && rm -rf /var/www/framework/var/cache/* \
  && rm -rf /var/www/framework/var/log/* \
  && chown -R root:www-data /var/www/framework/var/ \
  && chmod -R 777 /var/www/framework/var/ \
  && rm /var/www/framework/.env || echo ".env already deleted"
# <<< PRODUCTION IMAGE

# >>> PRODUCTION API IMAGE
FROM prod as api

HEALTHCHECK --interval=1m --timeout=3s --start-period=5s CMD composer hc || exit 1

ENTRYPOINT ["composer"]
CMD ["server:start"]
# <<< PRODUCTION API IMAGE

# >>> PRODUCTION CMD IMAGE
FROM prod as cmd

HEALTHCHECK --interval=1m --timeout=3s --start-period=5s CMD if pgrep -x "php" > /dev/null; then exit 0; else exit 1; fi

ENTRYPOINT ["bin/console"]
CMD ["about"]
# <<< PRODUCTION CMD IMAGE

# >>> CI IMAGE
FROM prod as ci

RUN apt update \
    && apt dist-upgrade -y \
    && apt install --no-install-recommends -y sudo

RUN groupadd -g ${GID} -o ${UNAME} \
  && useradd -m -u ${UID} -g ${GID} -o -s /bin/bash ${UNAME} \
  && usermod -aG sudo ${UNAME} \
  && echo "${UNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER $UNAME

HEALTHCHECK --interval=1m --timeout=3s --start-period=5s CMD composer hc || exit 1
# <<< CI IMAGE
