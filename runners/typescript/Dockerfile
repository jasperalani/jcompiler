FROM node:23-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

RUN npm install -g esbuild

EXPOSE 8003
CMD ["node", "server.js"]