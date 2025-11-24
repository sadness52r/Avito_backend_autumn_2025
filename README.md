# Avito_backend_autumn_2025

Backend service for assigning pull request reviewers to team members.

## Quick Start

### Installation

1. **Clone the repository**
   ```bash
   https://github.com/sadness52r/Avito_backend_autumn_2025.git
   cd Avito_backend_autumn_2025

2. **Run services**
    ```bash
    docker-compose up --build

3. **Check this!**
    ```bash
    curl http://localhost:8080/<your_endpoint>

**Important:** Create your own .env file (like in .env.example) to set database configuration. Good luck!

**Problems that I felt when wrote the code:** 
1. Some errors (invalid request, internal server error and etc.) did not notice into openapi.yaml. I decided to write my own errors on the cases.
2. Setting the environment of db. I created .env file to save the secrets of the db.
3. I decided to create the pr_reviewers table to get easy access to members who can check reviews of someone. This solution helped refuse from some fields of tables.
4. First, I forgot that it is need if the available candidates less than 2 need to assign 0 or 1. Fixed this!

**Integration tests:**
First of all you need to set your test env (check /tests/.env.example) and run app:

    ```bash
    docker-compose up --build

To run tests you need enter:

    ```bash
    cd tests
    make test-env
    make test-all
