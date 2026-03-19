FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
RUN set -eux; \
  for i in 1 2 3; do \
    npm ci --no-audit --no-fund && exit 0; \
    echo "npm ci failed (attempt ${i}), retrying..."; \
    sleep $((i * 5)); \
  done; \
  exit 1

COPY . .

EXPOSE 5175

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5175", "--strictPort"]
