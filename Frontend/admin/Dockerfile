FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_APP_BASE_PATH=/routerease/admin/
ARG VITE_API_BASE_URL=/routerease/dns
ARG VITE_OIDC_AUTHORITY
ARG VITE_OIDC_CLIENT_ID=routerease-admin
ENV VITE_APP_BASE_PATH=$VITE_APP_BASE_PATH VITE_API_BASE_URL=$VITE_API_BASE_URL VITE_OIDC_AUTHORITY=$VITE_OIDC_AUTHORITY VITE_OIDC_CLIENT_ID=$VITE_OIDC_CLIENT_ID
RUN npm run build
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD wget -q -O /dev/null http://localhost:8080/routerease/admin/ || exit 1
