FROM oven/bun:1 AS dev
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install
EXPOSE 5173
CMD ["bun", "run", "dev", "--host"]

FROM oven/bun:1 AS builder
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install
COPY . .
RUN bun run build

FROM node:20-slim AS prod
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
RUN npm install --omit=dev
EXPOSE 3000
CMD ["node", "build"]
