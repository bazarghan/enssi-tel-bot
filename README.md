# Enssi - Your Personal Telegram Language Tutor 🤖

An intelligent Telegram bot designed to make language learning effective and engaging through structured courses, interactive quizzes, and a smart Spaced Repetition System (SRS).

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Go Version](https://img.shields.io/badge/go-1.21+-blue)
![License](https://img.shields.io/badge/license-MIT-yellow)
![Docker](https://img.shields.io/badge/Docker-Powered-blue?logo=docker)

---

## 📖 Table of Contents

- [About The Project](#about-the-project)
- [✨ Key Features](#-key-features)
- [🛠️ Tech Stack & Architecture](#️-tech-stack--architecture)
- [🚀 Local Development Setup](#-local-development-setup)
  - [Prerequisites](#prerequisites)
  - [Installation & Setup](#installation--setup)
- [🔧 Development & Tooling](#-development--tooling)
  - [Makefile Commands](#makefile-commands)
- [📂 Project Structure](#-project-structure)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)
- [📞 Contact](#-contact)

---

## About The Project

Enssi is a sophisticated Telegram bot built with Go, designed to offer a seamless and interactive language learning experience. It helps users learn new vocabulary through structured courses, reinforces learning with quizzes, and uses a Spaced Repetition System (SRS) to ensure long-term retention.

The project is architected for scalability and maintainability, featuring a clean separation of concerns, containerized services, and a robust set of development tools.

---

## ✨ Key Features

-   📚 **Structured Courses**: Learn vocabulary in organized, topic-based blocks.
-   🧠 **Interactive Quizzes**: Test your knowledge after each learning module to solidify understanding.
-   📅 **Spaced Repetition System (SRS)**: The bot intelligently schedules review sessions based on your performance, maximizing memory retention.
-   🏆 **Achievement System**: Stay motivated by unlocking achievements and tracking your progress with unique, progressively-revealed visuals.
-   🎧 **Rich Media Content**: Learn with images and audio pronunciations for a multi-sensory experience.
-   ⚙️ **Admin Panel**: Built-in tools for administrators to view bot statistics and broadcast messages to all users.
-   ⛓️ **Version-Controlled Migrations**: Database schema changes are managed through versioned SQL files, ensuring consistency across all environments.
-   🌱 **Configurable Data Seeding**: Easily populate the database with initial data (courses, words) using a simple make command.
-   🔒 **Concurrency Safe**: Handles simultaneous user interactions reliably, preventing data races and ensuring a smooth user experience.
-   🐳 **Containerized**: Fully containerized with Docker and Docker Compose for easy setup and deployment.

---

## 🛠️ Tech Stack & Architecture

### Tech Stack

-   **Backend**: [Go](https://golang.org/)
-   **Database**: [PostgreSQL](https://www.postgresql.org/)
-   **ORM**: [GORM](https://gorm.io/)
-   **Database Migrations**: [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
-   **Telegram API**: [Telebot (v4)](https://github.com/tucnak/telebot)
-   **Configuration**: [Viper](https://github.com/spf13/viper)
-   **Logging**: [Zap](https://github.com/uber-go/zap)
-   **Dependency Injection**: [Google Wire](https://github.com/google/wire)
-   **Containerization**: [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

### Architecture

This project follows the principles of **Clean Architecture**, ensuring a clear separation of concerns and making the codebase modular, testable, and independent of external frameworks.

-   **`Domain`**: The core of the application. It contains the business logic and entities (e.g., User, Course, Word) and is completely independent of any other layer.
-   **`Usecase`**: Orchestrates the flow of data between the `Domain` and the outer layers. It contains the application-specific business rules.
-   **`Adapter`**: The outermost layer. It connects the application to external technologies like the Telegram API, PostgreSQL database, and other services.

---

## 🚀 Local Development Setup

Follow these instructions to get a local copy up and running for development and testing purposes.

### Prerequisites

-   [Go](https://golang.org/doc/install/) (version 1.21 or higher)
-   [Docker](https://docs.docker.com/get-docker/)
-   [Docker Compose](https://docs.docker.com/compose/install/)
-   [make](https://www.gnu.org/software/make/)

### Installation & Setup

1.  **Clone the repository:**
    ```sh
    git clone [https://github.com/2000ostd/enssi-tel-bot.git](https://github.com/2000ostd/enssi-tel-bot.git)
    cd enssi-tel-bot
    ```

2.  **Create an environment file:**
    Copy the example `.env` file. This file is ignored by Git to keep your secrets safe.
    ```sh
    cp .env.example .env
    ```

3.  **Configure your `.env` file:**
    Open the `.env` file and add your credentials.
    ```env
    # Your Telegram bot token from BotFather
    ENSSI_TELEGRAM_TOKEN=your_telegram_bot_token

    # Credentials for the PostgreSQL database
    # These should match the POSTGRES_* variables in docker-compose.yml
    ENSSI_DATABASE_USER=postgres
    ENSSI_DATABASE_PASSWORD=your_super_secret_password
    ENSSI_DATABASE_NAME=enssi_bot_db
    ENSSI_DATABASE_HOST=localhost
    ENSSI_DATABASE_PORT=5432
    ```
    > **Important**: For local development, `ENSSI_DATABASE_HOST` should be `localhost`. The `db` hostname is for communication between containers.

4.  **Start the Database Service:**
    Launch the PostgreSQL database using Docker Compose. This runs the database in the background.
    ```sh
    docker-compose up -d db
    ```

5.  **Run Database Migrations:**
    Apply the latest database schema. This creates all the necessary tables. You only need to run this once initially and whenever there are new migration files.
    ```sh
    make migrate-up
    ```

6.  **Seed the Database:**
    Populate the database with the initial set of courses, words, and achievements from the configuration files.
    ```sh
    make seed
    ```

7.  **Run the Bot Application:**
    You can now start the bot locally. It will connect to the Dockerized database you set up.
    ```sh
    make run-bot
    ```

8.  **Start interacting with your bot!**
    Find your bot on Telegram and send the `/start` command.

---

## 🔧 Development & Tooling

The `Makefile` provides several commands to streamline the development process.

### Makefile Commands

-   `make help`: Displays a list of all available commands.

**Local Development:**
-   `make run-bot`: Runs the bot application locally without Docker.
-   `make run-worker`: Runs the background worker locally without Docker.
-   `make build`: Compiles and builds the binaries for both the bot and worker.
-   `make test`: Runs all Go tests in the project.
-   `make wire`: Generates the dependency injection files using Google Wire.
-   `make clean`: Removes build artifacts and cleans the test cache.

**Database Management:**
-   `make db-reset`: ⚠️ **Destructive!** Resets the dev database by dropping all tables, re-migrating, and re-seeding.
-   `make seed`: Populates the database with initial data from config files.
-   `make migrate-up`: Applies all available 'up' database migrations.
-   `make migrate-down`: Reverts the last 'down' database migration.
-   `make migrate-create name=<name>`: Creates new up/down migration files.

**Docker Management:**
-   `make docker-up`: Starts all services (bot, worker, db) using `docker-compose`.
-   `make docker-down`: Stops and removes all running containers.
-   `make docker-build`: Builds the Docker images for the bot and worker.

---

## 📂 Project Structure

The project's structure is organized according to Clean Architecture principles:

```
.
├── cmd/                 # Main application entrypoints (bot, worker, seeder, migrate)
├── configs/             # Configuration files (config.yaml, seeder.config.yaml)
├── internal/
│   ├── adapter/         # Adapters for external services (PostgreSQL, Telegram)
│   ├── domain/          # Core business logic and entities
│   ├── platform/        # Supporting platform code (DI, config loading, DB connection)
│   ├── seeder/          # Logic for database data population
│   └── usecase/         # Application-specific business rules (Commands & Queries)
├── migrations/          # Version-controlled SQL migration files
├── pkg/                 # Reusable packages (e.g., imagekit, tgmarkdown)
├── assets/              # Static assets like images and course data
├── test/                # Integration and end-to-end tests
├── Dockerfile           # Multi-stage Dockerfile for building the application
├── docker-compose.yml   # Docker Compose configuration for all services
└── Makefile             # Helper commands for development
```

---

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---

## 📞 Contact

Project Link: [https://github.com/2000ostd/enssi-tel-bot](https://github.com/2000ostd/enssi-tel-bot)

