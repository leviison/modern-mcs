FROM node:20.19-alpine AS web-builder
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/modern-mcs ./cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=go-builder /out/modern-mcs /modern-mcs
COPY --from=web-builder /src/web/dist /web/dist
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/modern-mcs"]
