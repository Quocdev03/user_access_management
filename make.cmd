@echo off
setlocal

set DB_URL="mysql://uam_user:uam_password@tcp(localhost:3306)/uam_db"

if "%1"=="" goto help
if "%1"=="dev" goto dev
if "%1"=="run" goto run
if "%1"=="build" goto build
if "%1"=="migrate-up" goto migrate_up
if "%1"=="migrate-down" goto migrate_down
if "%1"=="migrate-create" goto migrate_create
if "%1"=="swagger" goto swagger
if "%1"=="test" goto test
if "%1"=="test-coverage" goto test_coverage
if "%1"=="docker-up" goto docker_up
if "%1"=="docker-down" goto docker_down

echo Unknown command: %1
goto help

:help
echo Usage: make.cmd [command]
echo.
echo Commands:
echo   dev             Run air
echo   run             Run server
echo   build           Build server (CGO_ENABLED=0)
echo   migrate-up      Run migrations up
echo   migrate-down    Run migrations down
echo   migrate-create  Create a new migration (Usage: make.cmd migrate-create create_xxx_table)
echo   swagger         Generate swagger docs
echo   test            Run tests
echo   test-coverage   Run tests with coverage and open html
echo   docker-up       Start docker compose
echo   docker-down     Stop docker compose
goto end

:dev
air
goto end

:run
go run cmd\server\main.go
goto end

:build
set CGO_ENABLED=0
go build -o bin\server.exe .\cmd\server
goto end

:migrate_up
.\migrate -path migrations -database %DB_URL% -verbose up
goto end

:migrate_down
.\migrate -path migrations -database %DB_URL% -verbose down
goto end

:migrate_create
if "%2"=="" (
    echo Error: name is required. Usage: make.cmd migrate-create create_xxx_table
    exit /b 1
)
.\migrate create -ext sql -dir migrations -seq %2
goto end

:swagger
swag init -g cmd\server\main.go
goto end

:test
go test -v .\...
goto end

:test_coverage
go test -coverprofile=coverage.out .\...
go tool cover -html=coverage.out
goto end

:docker_up
docker compose up -d
goto end

:docker_down
docker compose down
goto end

:end
endlocal
