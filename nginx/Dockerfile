# checkov:skip=CKV_DOCKER_2:health check is not needed for config test stage
FROM nginx:1.29.6-alpine AS config-test
COPY . /etc/nginx/
RUN nginx -t
# USER is added to satisfy trivy DS-0002; config-test stage does not run as a service
USER nginx
