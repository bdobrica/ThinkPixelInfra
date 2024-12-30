FROM redis:7.4.1-bookworm

# Copy the sentinel configuration file
COPY sentinel.conf /etc/sentinel.conf

# Copy the startup script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

# Make the script executable
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the startup script as the container's entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
