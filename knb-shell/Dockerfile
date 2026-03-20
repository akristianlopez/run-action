# --- ÉTAPE 1 : Build du Frontend (Vite) ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY ./web/package*.json ./
RUN npm install
COPY ./web/ ./
RUN npm run build

# --- ÉTAPE 2 : Build du Backend (Go) ---
FROM golang:1.25.5-alpine AS backend-builder
WORKDIR /app

# Installation des outils nécessaires
RUN apk add --no-cache git ca-certificates

# Copie des fichiers de définition des modules
# On les copie un par un pour être sûr de ne pas échouer si go.sum n'existe pas
COPY go.mod ./
COPY go.sum* ./

# Configuration du proxy Go (souvent utile pour éviter les erreurs de timeout/réseau)
ENV GOPROXY=https://proxy.golang.org,direct

# Téléchargement des dépendances
# IMPORTANT : Assurez-vous d'avoir lancé 'go mod tidy' sur votre machine avant le build
RUN go mod download

# Copie du reste du code source
COPY . .

# Compilation statique
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o knb-shell main.go

# --- ÉTAPE 3 : Image finale d'exécution ---
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copie du binaire
COPY --from=backend-builder /app/knb-shell .

# Copie des fichiers web compilés
COPY --from=frontend-builder /app/web/dist ./web/dist

# Configuration des variables d'environnement
ENV PORT=8080
ENV CONSUL_ADDR=consul-server:8500
ENV SERVICE_NAME=app
ENV SERVICE_DOMAIN=wosa.local
ENV CONFIG_PATH=knb/services/shell
ENV SERVICE_KIND=swarm

# Exposition du port
EXPOSE 8080

# Lancement
ENTRYPOINT ["./knb-shell"]