FROM oven/bun:1 AS build
WORKDIR /src
COPY web/vue/package.json web/vue/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/vue/ .
RUN bunx --bun vite build

FROM nginx:1.27-alpine
COPY web/vue/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 8080
