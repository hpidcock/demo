FROM node

WORKDIR /app
ADD . .

RUN npm install
RUN npm run build

ENV HOST 0.0.0.0

ENTRYPOINT [ "npm", "run", "start" ]
