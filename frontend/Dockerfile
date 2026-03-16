FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM node:22-alpine

WORKDIR /app

ENV NODE_ENV=production

COPY --from=builder /app/package.json /app/package-lock.json ./
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/.svelte-kit ./.svelte-kit
COPY --from=builder /app/build ./build

EXPOSE 4173

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:4173/resources || exit 1

CMD ["npm", "run", "preview", "--", "--host", "0.0.0.0", "--port", "4173"]
