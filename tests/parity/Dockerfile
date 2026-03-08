# Stage 1: Build GoHopper
FROM golang:1.24 AS go-build
RUN apt-get update && apt-get install -y zlib1g-dev && rm -rf /var/lib/apt/lists/*
COPY . /src
WORKDIR /src
RUN go build -o /gohopper ./cmd/gohopper

# Stage 2: Runtime with both servers
FROM eclipse-temurin:21-jre
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
RUN curl -fL -o graphhopper.jar \
    https://repo1.maven.org/maven2/com/graphhopper/graphhopper-web/11.0/graphhopper-web-11.0.jar
COPY --from=go-build /gohopper /app/gohopper
COPY core/testdata/andorra.osm.pbf /app/andorra.osm.pbf
COPY tests/parity/config.yml /app/config.yml
