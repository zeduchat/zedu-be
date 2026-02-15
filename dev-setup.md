# Zedu Backend - Docker Setup

This guide explains how to run the Zedu backend application and its dependencies using Docker and Docker Compose.

### Prerequisites

  Docker ≥ 24.x
  Docker Compose ≥ 2.x
  make (optional, for automating commands)

### Configuration
  .air.toml for hot reload
  config.dev.json contains Centrifugo configuration.
  Default variables exists in dev Compose file

### Make Commands

  - Build and start all services
    - make start:dev

  - Stop all services and remove volumes
    - make dev-clean

### Running without make

  - Build and start all services
     - docker compose -f docker-compose.dev.yml up --build

  - Stop all services and remove volumes
    - docker compose -f docker-compose.dev.yml down -v

### To inspect app logs

  - docker exec -it telex-backend sh
  - cat logs/app.log