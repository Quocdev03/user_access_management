@echo off
setlocal EnableExtensions

set "DB_URL=mysql://uam_user:uam_password@tcp(localhost:3306)/uam_db"
set "SWAG_CMD="

if "%~1"=="" goto help
if /I "%~1"=="help" goto help
if /I "%~1"=="dev" goto dev
if /I "%~1"=="run" goto run
if /I "%~1"=="build" goto build
if /I "%~1"=="migrate-up" goto migrate_up
if /I "%~1"=="migrate-down" goto migrate_down
if /I "%~1"=="migrate-create" goto migrate_create
if /I "%~1"=="swagger" goto swagger
if /I "%~1"=="swagger-install" goto swagger_install
if /I "%~1"=="test" goto test
if /I "%~1"=="test-coverage" goto test_coverage
if /I "%~1"=="docker-up" goto docker_up
if /I "%~1"=="docker-down" goto docker_down

echo Unknown command: %~1
goto help

:help
echo Usage: make.cmd [command]
echo.
echo Commands:
echo   dev              Run air (hot reload)
echo   run              Run server
echo   build            Build server (CGO_ENABLED=0)
echo   migrate-up       Run migrations up
echo   migrate-down     Run migrations down
echo   migrate-create   Create migration (make.cmd migrate-create create_xxx_table)
echo   swagger          Generate docs/docs.go + swagger.json/yaml
echo   swagger-install  Install swag CLI via go install
echo   test             Run tests
echo   test-coverage    Run tests with coverage HTML
echo   docker-up        docker compose up -d
echo   docker-down      docker compose down
goto end

:dev
air
goto end

:run
go run cmd\server\main.go
goto end

:build
set CGO_ENABLED=0
if not exist bin mkdir bin
go build -o bin\server.exe .\cmd\server
goto end

:migrate_up
if exist migrate.exe (
    .\migrate.exe -path migrations -database "%DB_URL%" -verbose up
) else if exist migrate (
    .\migrate -path migrations -database "%DB_URL%" -verbose up
) else (
    migrate -path migrations -database "%DB_URL%" -verbose up
)
goto end

:migrate_down
if exist migrate.exe (
    .\migrate.exe -path migrations -database "%DB_URL%" -verbose down
) else if exist migrate (
    .\migrate -path migrations -database "%DB_URL%" -verbose down
) else (
    migrate -path migrations -database "%DB_URL%" -verbose down
)
goto end

:migrate_create
if "%~2"=="" (
    echo Error: name is required. Usage: make.cmd migrate-create create_xxx_table
    exit /b 1
)
if exist migrate.exe (
    .\migrate.exe create -ext sql -dir migrations -seq %~2
) else if exist migrate (
    .\migrate create -ext sql -dir migrations -seq %~2
) else (
    migrate create -ext sql -dir migrations -seq %~2
)
goto end

:swagger
call :resolve_swag
if errorlevel 1 (
    echo swag not found. Run: make.cmd swagger-install
    exit /b 1
)
"%SWAG_CMD%" init -g cmd\server\main.go -o docs --parseDependency --parseInternal
if errorlevel 1 exit /b 1
echo Swagger generated: docs\docs.go, docs\swagger.json, docs\swagger.yaml
goto end

:swagger_install
go install github.com/swaggo/swag/cmd/swag@latest
if errorlevel 1 exit /b 1
echo Installed swag. Ensure %%USERPROFILE%%\go\bin is on PATH.
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

:resolve_swag
where swag >nul 2>&1
if not errorlevel 1 (
    set "SWAG_CMD=swag"
    exit /b 0
)
if exist "%USERPROFILE%\go\bin\swag.exe" (
    set "SWAG_CMD=%USERPROFILE%\go\bin\swag.exe"
    exit /b 0
)
for /f "delims=" %%i in ('go env GOPATH 2^>nul') do set "GOPATH_VAL=%%i"
if defined GOPATH_VAL if exist "%GOPATH_VAL%\bin\swag.exe" (
    set "SWAG_CMD=%GOPATH_VAL%\bin\swag.exe"
    exit /b 0
)
exit /b 1

:end
endlocal
