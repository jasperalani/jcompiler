FROM node:23-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

RUN npm cache clean --force
RUN npm uninstall typescript tsc
RUN npm install typescript

EXPOSE 8003
CMD ["node", "server.js"]