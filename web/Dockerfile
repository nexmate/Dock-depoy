FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
# npm ci rejects lockfiles made on Windows (missing linux-only optional deps)
RUN npm install --no-audit --no-fund

FROM deps AS build
COPY . .
# Rewrites to the API are resolved at build time.
ARG API_URL=http://api:8080
ENV API_URL=$API_URL NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
