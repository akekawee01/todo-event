FROM oven/bun:1 AS build
WORKDIR /src
COPY web/onboarding/package.json web/onboarding/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/onboarding/ .
RUN bunx --bun vite build

FROM nginx:1.27-alpine
COPY web/onboarding/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 8080
