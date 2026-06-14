FROM node:20.11.1-alpine3.19 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20.11.1-alpine3.19 AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY package.json tsconfig.json tsconfig.build.json ./
COPY src ./src
RUN npm run build

FROM node:20.11.1-alpine3.19 AS runtime
ENV NODE_ENV=production
ENV PORT=8080
WORKDIR /app
RUN addgroup -S -g 10001 tradeops && adduser -S -D -H -u 10001 -G tradeops tradeops
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY --from=build /app/dist ./dist
USER 10001:10001
EXPOSE 8080
CMD ["node", "dist/index.js"]
