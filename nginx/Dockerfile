# checkov:skip=CKV_DOCKER_2:health check is not needed for config test stage
FROM nginx:1.29.6-alpine AS config-test
COPY nginx.conf /etc/nginx/nginx.conf
RUN nginx -t
# USER is added to satisfy trivy DS-0002; config-test stage does not run as a service
USER nginx
