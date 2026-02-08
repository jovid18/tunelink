# TuneLink Web

TuneLink의 프론트엔드 애플리케이션. URL 단축 서비스의 사용자 인터페이스를 제공합니다.

## Tech Stack

- React 19 + TypeScript
- Vite 7
- TailwindCSS 4
- Axios (HTTP Client)
- Nginx (Production Serving)

## Project Structure

```
src/
├── app/
│   ├── components/    # 공통 컴포넌트
│   ├── hooks/         # 커스텀 훅
│   ├── libs/          # HTTP 클라이언트 등 유틸리티
│   ├── models/        # 데이터 모델 (URL 등)
│   ├── repositories/  # API 통신 레이어
│   ├── screens/       # 페이지 컴포넌트
│   │   └── Home/      # 메인 화면
│   └── services/      # 비즈니스 로직
├── App.tsx            # 루트 컴포넌트
└── main.tsx           # 엔트리포인트
```

## Getting Started

```bash
# 의존성 설치
npm install

# 개발 서버 실행 (기본 포트: 5173)
npm run dev

# 프로덕션 빌드
npm run build

# 빌드 결과물 미리보기
npm run preview

# 린트 실행
npm run lint
```

## Environment

API 서버 주소는 `apps/web/src/app/libs/http-client.ts`에서 설정합니다.

## Docker

Nginx 기반의 프로덕션 이미지로 빌드됩니다.

```bash
docker build -t tunelink-web .
docker run -p 80:80 tunelink-web
```
