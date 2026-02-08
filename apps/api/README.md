# TuneLink API

TuneLink의 백엔드 API 서버. URL 단축, 리다이렉트, 클릭 카운팅 기능을 제공합니다.

## Tech Stack

- Go 1.24
- Gin (HTTP Framework)
- GORM (ORM, MySQL)
- Redis (Cache + Click Counter)

## Project Structure

```
├── cmd/api/              # 엔트리포인트 (main.go)
├── internal/
│   ├── adapter/
│   │   ├── in/http/      # HTTP 핸들러 (인바운드 어댑터)
│   │   └── out/
│   │       ├── cache/
│   │       │   ├── redis/ # Redis 캐시 어댑터
│   │       │   └── noop/  # 캐시 비활성화 시 Noop 어댑터
│   │       └── persistence/
│   │           └── mysql/ # MySQL 리포지토리 어댑터
│   ├── application/
│   │   ├── url/          # URL 유스케이스 (비즈니스 로직)
│   │   └── sync/         # 클릭 수 동기화 서비스
│   ├── domain/url/       # 도메인 엔티티 및 리포지토리 인터페이스
│   └── infrastructure/   # 설정 로드 (DB, Redis 연결)
├── Dockerfile
├── .env.example
└── go.mod
```

Hexagonal Architecture (포트 & 어댑터 패턴)를 따릅니다.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | 헬스 체크 |
| POST | `/api/urls` | 단축 URL 생성 |
| GET | `/r/{shortUrl}` | 원본 URL로 리다이렉트 |

## Getting Started

```bash
# 환경 변수 설정
cp .env.example .env
# .env 파일에 DB 정보 입력

# 서버 실행
go run cmd/api/main.go
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_HOST` | Yes | - | MySQL 호스트 |
| `DB_PORT` | Yes | - | MySQL 포트 |
| `DB_USER` | Yes | - | MySQL 사용자 |
| `DB_PASSWORD` | Yes | - | MySQL 비밀번호 |
| `DB_NAME` | Yes | - | 데이터베이스 이름 |
| `REDIS_HOST` | No | `localhost` | Redis 호스트 |
| `REDIS_PORT` | No | `6379` | Redis 포트 |
| `PORT` | No | `8080` | 서버 포트 |
| `BASE_URL` | No | `http://localhost:8080` | 단축 URL 생성 시 기본 도메인 |

## Docker

```bash
docker build -t tunelink-api .
docker run -p 8080:8080 --env-file .env tunelink-api
```

## Key Features

- **Cache-Aside Pattern**: Redis를 활용한 URL 조회 캐싱 (TTL: 1h)
- **Click Batch Sync**: Redis INCR로 클릭 수 카운팅 후 백그라운드에서 DB 동기화 (10초 간격)
- **Graceful Shutdown**: 서버 종료 시 진행 중인 요청 완료 대기 후 최종 클릭 수 동기화
- **Graceful Degradation**: Redis 장애 시 DB로 자동 폴백
