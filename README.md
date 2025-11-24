# Telex API

The **Telex API** is a scalable Go-based backend service that integrates multiple data sources and technologies such as PostgreSQL, Redis, MinIO, MongoDB, Firebase, RabbitMQ, TypeSense, and Stripe. It also supports background jobs, external service integration, and real-time notifications.


## 🚀 Features

- PostgreSQL + GORM for relational data
- Redis caching
- MinIO for object storage
- MongoDB support
- TypeSense for search indexing     <--No longer in use -->
- Elasticsearch integration
- Stripe for payments
- RabbitMQ for async task queues
- Firebase for push notifications
- Cron jobs for background tasks
- Real-time communication via Centrifugo
- Configurable through environment variables or config file
- Built-in database migrations and seeding


## Prerequisites

- Go >= 1.24
- PostgreSQL
- Redis
- MinIO
- MongoDB
- RabbitMQ
- TypeSense         <--No longer in use -->
- Elasticsearch
- Firebase (Service Account Key)
- Stripe account (for API key)
- Centrifugo

Make sure all services are installed and running locally or in Docker containers.


## Setup

1. Clone this repo.

```sh

    git clone https://github.com/telexorg/telex_be.git
```

2. Change your active directory in the project.

```sh

   cd telex_be
```

3. Copy `app-sample.env` into `app.env`.
   Swap out values if necessary. If you're using the Docker environment, remember to set any network host (e.g localhost) in your .env file to the name of the corresponding service in the `docker-compose.yaml` file.

4. Install Go dependencies.

```sh
    go mod tidy
```

5. Run the app.

```sh
    go run main.go
```

6. Confirm that the API is running by visiting `http://localhost:{port}`.

7. Get to work :).



## 🤝 Contributing

1. Fork the repository

2. Create your feature branch: `git checkout -b feature/your-feature`.

3. Commit your changes: `git commit -am 'Add some feature'`.
   

5. Push to the branch: `git push origin feature/your-feature`.

6. Open a pull request.
