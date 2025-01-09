#!/bin/bash

# Exit on error
set -euo pipefail

# Check MySQL connection
echo "Checking MySQL connection..."
while ! mysql -u"${WORDPRESS_DB_USER}" -p"${WORDPRESS_DB_PASSWORD}" -h"${WORDPRESS_DB_HOST}" -e "SELECT 1;"; do
    echo "MySQL is unavailable - sleeping..."
    sleep 1
done

# Check if the database exists
echo "Checking if database '${WORDPRESS_DB_NAME}' exists..."
if ! mysql -u"${WORDPRESS_DB_USER}" -p"${WORDPRESS_DB_PASSWORD}" -h"${WORDPRESS_DB_HOST}" -e "USE ${WORDPRESS_DB_NAME}; SHOW TABLES LIKE 'wp_options';" | grep -q 'wp_options'; then

    echo "Database does not exist. Creating database..."
    #wp db create --allow-root --path=/opt/wordpress

    echo "Installing WordPress..."
    wp core install \
        --allow-root \
        --path=/opt/wordpress \
        --url=${WORDPRESS_SITE_URL} \
        --title=${WORDPRESS_TITLE} \
        --admin_user=${WORDPRESS_ADMIN_USER} \
        --admin_password=${WORDPRESS_ADMIN_PASSWORD} \
        --admin_email=${WORDPRESS_ADMIN_EMAIL}

    echo "WordPress installation completed."
else
    echo "Database '${WORDPRESS_DB_NAME}' already exists. Skipping setup."
fi


# Start PHP-FPM
php-fpm8.2 -D

# Execute CMD
exec "$@"
