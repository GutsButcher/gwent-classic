# Gwent Backend

Go backend server for Gwent Classic game with PostgreSQL database.

## Setup

1. Install PostgreSQL and create a database:
```sql
CREATE DATABASE gwent_db;
CREATE USER gwent_user WITH PASSWORD 'gwent_password';
GRANT ALL PRIVILEGES ON DATABASE gwent_db TO gwent_user;
```

2. Run migrations:
```bash
psql -U gwent_user -d gwent_db -f migrations/001_initial_schema.sql
```

3. Copy `.env.example` to `.env` and update values:
```bash
cp .env.example .env
```

4. Install dependencies:
```bash
go mod download
```

5. Run the server:
```bash
go run cmd/server/main.go
```

## API Endpoints

### Authentication

- `POST /api/auth/register` - Register new user
  ```json
  {
    "email": "user@example.com",
    "name": "PlayerName",
    "password": "password123"
  }
  ```

- `POST /api/auth/login` - Login user
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```

- `GET /api/users/search?nameID=PlayerName#123` - Find user by name#ID

### Protected Routes

All routes under `/api/protected/*` require Authorization header:
```
Authorization: Bearer <jwt_token>
```

#### Game Management

- `POST /api/protected/games/ai` - Create a new game vs AI

- `GET /api/protected/games` - Get all active games for the user

#### Challenges

- `POST /api/protected/challenges` - Challenge another player
  ```json
  {
    "challenged_name_id": "PlayerName#123"
  }
  ```

- `GET /api/protected/challenges` - Get pending challenges

- `POST /api/protected/challenges/{id}/respond` - Accept/decline challenge
  ```json
  {
    "accept": true
  }
  ```

### WebSocket Connection

- `ws://localhost:8080/ws/game/{gameId}?token={jwt_token}` - Real-time game updates

Message format:
```json
{
  "type": "move",
  "game_id": 1,
  "user_id": 1,
  "payload": {
    "action": "play_card",
    "card": {...}
  }
}
```