# Build stage
FROM golang:1.26.10-alpine3.23 AS builder
WORKDIR /app
# O migrate é compilado aqui (só com o driver postgres) em vez de baixar o
# binário pronto: assim ele usa o mesmo Go corrigido da API e passa no Trivy.
RUN CGO_ENABLED=0 go install -trimpath -ldflags="-s -w" -tags 'postgres' \
    github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o main cmd/server/main.go

# Run stage
FROM alpine:3.23
# Fuso horário do servidor: as datas dos filtros (tempo médio) são interpretadas nele.
ENV TZ=America/Sao_Paulo
RUN apk add --no-cache tzdata ca-certificates \
    && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /go/bin/migrate ./migrate
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./migration
COPY db/seed ./seed
RUN chmod +x /app/start.sh /app/wait-for.sh && chown -R app:app /app
USER app

EXPOSE 8080
CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]
