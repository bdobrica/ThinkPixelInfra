# Use a lightweight Debian base image
FROM debian:bookworm-slim

# Set environment variables for non-interactive installations
ENV DEBIAN_FRONTEND=noninteractive

# Update and install required packages
RUN apt-get update && apt-get install -y \
    nginx \
    mariadb-client \
    php-fpm \
    php-mysql \
    php-cli \
    php-curl \
    php-gd \
    php-mbstring \
    php-xml \
    php-zip \
    php-bcmath \
    php-intl \
    curl \
    tar \
    unzip \
    && apt-get clean

# Set up directories
RUN mkdir -p /var/www/html /opt/wordpress /etc/nginx/conf.d && \
    chown -R www-data:www-data /var/www/html /opt/wordpress

# Install WordPress CLI
RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && \
    chmod +x wp-cli.phar && mv wp-cli.phar /usr/local/bin/wp

# Download the latest WordPress
RUN wp core download --allow-root --path=/opt/wordpress --locale=en_US

# Remove themes and plugins
RUN rm -rf /opt/wordpress/wp-content/themes/* && \
    rm -rf /opt/wordpress/wp-content/plugins/*

# Copy Nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

ENV WORDPRESS_DB_NAME=wordpress
ENV WORDPRESS_DB_USER=wordpress
ENV WORDPRESS_DB_PASSWORD=wordpress
ENV WORDPRESS_DB_HOST=mysql
ENV WORDPRESS_DB_CHARSET=utf8mb4
ENV WORDPRESS_SITE_URL=http://www.thinkpixel.io
ENV WORDPRESS_SITE_TITLE=ThinkPixel
ENV WORDPRESS_ADMIN_USER=thinkpixel
ENV WORDPRESS_ADMIN_PASSWORD=thinkpixel
ENV WORDPRESS_ADMIN_EMAIL=thinkpixel@thinkpixel.io

# Set up wp-config.php
RUN wp config create --allow-root --path=/opt/wordpress \
    --dbname="${WORDPRESS_DB_NAME}" \
    --dbuser="${WORDPRESS_DB_USER}" \
    --dbpass="${WORDPRESS_DB_PASSWORD}" \
    --dbhost="${WORDPRESS_DB_HOST}" \
    --dbcharset="${WORDPRESS_DB_CHARSET}" \
    --dbprefix=wp_ \
    --extra-php <<PHP
define('WP_DEBUG', true);
define('FS_METHOD', 'direct');
define('WP_DEFAULT_THEME', 'thinkpixel-ui');
PHP

# Expose ports for Nginx
EXPOSE 80

# Add entrypoint script to start all services
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Copy theme
COPY thinkpixel-ui /opt/wordpress/wp-content/themes/thinkpixel-ui

# Set entrypoint
ENTRYPOINT ["entrypoint.sh"]

# Default command
CMD ["nginx", "-g", "daemon off;"]
