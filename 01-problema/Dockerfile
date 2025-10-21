# Dockerfile para el servidor con race conditions
FROM golang:1.21-alpine AS builder

# Instalar dependencias del sistema
RUN apk add --no-cache git

# Establecer directorio de trabajo
WORKDIR /app

# Crear go.mod
RUN go mod init problema-reservas

# Copiar código fuente
COPY . .

# Compilar la aplicación
RUN go build -o servidor main.go

# Imagen final
FROM alpine:latest

# Instalar certificados SSL
RUN apk --no-cache add ca-certificates

# Crear usuario no-root
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Establecer directorio de trabajo
WORKDIR /app

# Copiar binario desde builder
COPY --from=builder /app/servidor .

# Cambiar propietario
RUN chown -R appuser:appgroup /app

# Cambiar a usuario no-root
USER appuser

# Exponer puerto
EXPOSE 8080

# Variables de entorno por defecto
ENV SERVIDOR_ID=servidor-1
ENV PUERTO=8080

# Comando por defecto
CMD ["./servidor"]