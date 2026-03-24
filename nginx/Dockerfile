# checkov:skip=CKV_DOCKER_2:health check is not needed for config test stage
FROM nginx:1.29.6-alpine AS config-test
COPY . /etc/nginx/
RUN nginx -t
# USER is added to satisfy trivy DS-0002; nginx -t requires root so this must come after RUN
USER nginx

# checkov:skip=CKV_DOCKER_2:health check is managed by ECS task definition
FROM nginx:1.29.6-alpine
RUN chown -R nginx:nginx /var/cache/nginx /run
USER nginx
COPY . /etc/nginx/
