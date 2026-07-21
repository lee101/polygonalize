FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY . .
RUN make build

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /src/bin/polygonalize-server /app/bin/polygonalize-server
COPY --from=build /src/web /app/web
ENV PORT=8080
EXPOSE 8080
CMD ["./bin/polygonalize-server"]

