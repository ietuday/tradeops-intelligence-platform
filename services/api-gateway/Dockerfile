FROM node:20.11.1-alpine3.19 AS deps
WORKDIR /app
COPY package.json ./
RUN npm install

FROM node:20.11.1-alpine3.19 AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY package.json tsconfig.json ./
COPY src ./src
RUN npm run build

FROM node:20.11.1-alpine3.19 AS runtime
ENV NODE_ENV=production
ENV PORT=8080
WORKDIR /app
RUN addgroup -S tradeops && adduser -S tradeops -G tradeops
COPY package.json ./
RUN npm install --omit=dev && npm cache clean --force
COPY --from=build /app/dist ./dist
USER tradeops
EXPOSE 8080
CMD ["node", "dist/index.js"]