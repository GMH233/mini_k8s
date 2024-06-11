# Dockerfile
FROM python:latest

# Add the script to the image
COPY start_script.py /usr/src/app/start_script.py

COPY run.sh /usr/src/app/run.sh

# Set the working directory
WORKDIR /usr/src/app

# Execute the script on container startup
CMD ["sh", "run.sh"]
