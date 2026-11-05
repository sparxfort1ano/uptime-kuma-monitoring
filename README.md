# Uptime Kuma Resilient Monitoring Stack

Локальный отказоустойчивый контур мониторинга для продакшн-сервиса **Uptime Kuma** с автоматизацией жизненного цикла и конфигурации через **Dagger.io (Go SDK)**. Проект разделен на два независимых инфраструктурных слоя (`docker-compose`), объединенных общей виртуальной сетью Docker.

## Технологический стек

| Компонент | Технология |
| :--- | :--- |
| **Оркестрация и CI/CD** | Dagger.io v0.21.11 (Go SDK), Docker 29.2.1, Docker Compose v5.0.2 |
| **Целевой сервис (App)** | Uptime Kuma 2.5.7 |
| **База данных** | MariaDB 11.4 |
| **Сбор и хранение метрик** | Prometheus v3.12.0 |
| **Визуализация** | Grafana 13.1.7 |
| **Экспортеры метрик** | mysqld-exporter v0.14.0, cAdvisor v0.60.5, node-exporter v1.12.0 |
| **Автоматизация и тесты** | Bash (`sh`), `curl`, `jq` |

## Архитектура

Инфраструктура разделена на два независимых слоя Docker Compose, которые взаимодействуют через общую сеть `monitoring_net`:

* **Слой 1 — Приложение и БД (`docker-compose.app.yml`):**
  * **Uptime Kuma:** основной веб-сервис мониторинга доступности внешних ресурсов. Работает в связке с внешней БД MariaDB.
  * **MariaDB:** реляционная БД с персистентным хранением данных в именованном томе `mariadb-data` и проверкой готовности движка InnoDB (`healthcheck.sh --connect --innodb_initialized`).
  * **MariaDB Exporter (`mysqld-exporter`):** сайдкар-агент для экспорта внутренних метрик СУБД в Prometheus. Учетная запись экспортера с минимально необходимыми привилегиями (`MAX_USER_CONNECTIONS 3`) создается автоматически при деплое через `scripts/init-exporter.sh`.

* **Слой 2 — Наблюдаемость (`docker-compose.monitoring.yml`):**
  * **Prometheus:** собирает метрики состояния контейнеров, хоста, БД и самой Uptime Kuma (аутентификация по API-ключу через механизм Docker Compose Secrets). Конфигурация упаковывается в образ через `Dockerfile.monitoring` для полной совместимости с изолированной средой выполнения Dagger (Docker-out-of-Docker).
  * **Grafana:** визуализирует состояние контура. Автоматическое подключение Data Source (Prometheus) и импорт JSON-дашбордов с динамической подстановкой `UID` выполняются через HTTP API (`scripts/configure-grafana.sh`).
  * **cAdvisor & Node Exporter:** собирают метрики утилизации ресурсов контейнеров (CPU, RAM, сеть, диск) и хост-машины.

Реализованы **End-to-End Orchestration (`deploy-all`)** с валидацией состояния всех сервисов и **Chaos Testing (`scripts/chaos-test.sh`)**. Кластер Galera (3 ноды + HAProxy) в рамках данного решения не реализовывался.

## Структура проекта

```text
.
├── .dagger/                        # Dagger-модуль на Go
│   └── main.go                     # Функции DeployApp, DeployMonitoring, ConfigureGrafana, DeployAll, ChaosTest
├── grafana/
│   └── dashboards/                 # JSON-модели дашбордов (Uptime Kuma, MariaDB, cAdvisor, Node Exporter)
├── scripts/
│   ├── init-exporter.sh            # Автоматическая инициализация пользователя БД для mysqld-exporter
│   ├── configure-grafana.sh        # Провижининг Data Source и дашбордов через Grafana HTTP API
│   └── chaos-test.sh               # Скрипт Chaos-тестирования Uptime Kuma
├── docker-compose.app.yml          # Слой 1: Uptime Kuma, MariaDB, mysqld-exporter
├── docker-compose.monitoring.yml   # Слой 2: Prometheus, Grafana, cAdvisor, node-exporter
├── Dockerfile.monitoring           # Сборка кастомного образа Prometheus с конфигурацией
├── prometheus.yml                  # Конфигурация таргетов сбора метрик Prometheus
├── .env.app.example                # Шаблон переменных окружения и секретов для Слоя 1
├── .env.monitoring.example         # Шаблон переменных окружения и секретов для Слоя 2
├── AI_REVIEW.md                    # Отчет об использовании AI-ассистентов и исправленных ошибках моделей
└── README.md                       # Документация проекта
```

## Локальный запуск и деплой

### Предварительные требования
* **ОС:** Linux, macOS или Windows 10/11 (WSL2 с Docker Desktop / Docker Engine)
* **Утилиты:** `docker` (29.2.1+), `docker compose` (v5.0.2+), `dagger` CLI (v0.21.11+), `jq`, `curl`

Установка `jq` (необходим для работы скрипта Chaos-тестирования на хосте):
```bash
sudo apt update && sudo apt install -y jq
```

### Пошаговая инструкция запуска

**1. Подготовка файлов окружения (`.env`):**

Скопируйте шаблоны конфигурационных файлов и заполните переменные в `.env.app` и `.env.monitoring` (пароли БД, данные для экспортера и учетную запись администратора Grafana):
```bash
cp .env.app.example .env.app
cp .env.monitoring.example .env.monitoring
```

**2. Запуск Слоя 1 (Приложение и БД) и получение API-ключа Uptime Kuma:**

Выполните команду Dagger для развертывания базы данных и сервиса Uptime Kuma:
```bash
dagger call deploy-app --docker-sock=/var/run/docker.sock
```

Для сбора метрик Uptime Kuma в Prometheus необходимо сгенерировать API-ключ:
1. Откройте веб-интерфейс Uptime Kuma: `http://localhost:3001` и создайте учетную запись администратора.
2. Перейдите в **Settings (Настройки) → API Keys (Ключи API)** и создайте новый ключ.
3. Впишите полученный ключ в файл `.env.monitoring` в переменную `KUMA_API_KEY=...` (для наглядности дашбордов рекомендуется также добавить 1–2 тестовых HTTP/DNS монитора в интерфейсе Uptime Kuma).

**3. Запуск Слоя 2 (Мониторинг) и автоконфигурация Grafana:**

Поднимите стек наблюдаемости и накатите дашборды через HTTP API:
```bash
dagger call deploy-monitoring --docker-sock=/var/run/docker.sock
dagger call configure-grafana --docker-sock=/var/run/docker.sock
```

**Альтернативный вариант (End-to-End Orchestration одной командой):**

Если `KUMA_API_KEY` уже прописан в `.env.monitoring`, всю платформу целиком (включая валидацию health-эндпоинтов всех систем) можно развернуть одной командой:
```bash
dagger call deploy-all --docker-sock=/var/run/docker.sock
```

## Доступные сервисы и проверка работоспособности

| Сервис | URL | Описание |
| :--- | :--- | :--- |
| **Uptime Kuma** | `http://localhost:3001` | Веб-интерфейс сервиса мониторинга |
| **Grafana** | `http://localhost:3000` | Панель дашбордов (логин/пароль из `.env.monitoring`) |
| **Prometheus** | `http://localhost:9090/targets` | Статус всех подключенных экспортеров (`1/1 UP`) |

В разделе **Dashboards** в Grafana автоматически создаются 4 дашборда:
1. **Uptime Kuma SLA / Latency / Certs** — статус сервиса и мониторов (с цветовой индикацией `ONLINE` / `DOWN`).
2. **Galera / MariaDB Overview** — статус СУБД, QPS, активные соединения, трафик и файловые дескрипторы.
3. **cAdvisor Dashboard** — потребление CPU, RAM и сетевого I/O по каждому контейнеру контура.
4. **Node Exporter Full** — метрики хост-машины.

## Chaos Testing

Для проверки отказоустойчивости и реакции мониторинга реализован скрипт `scripts/chaos-test.sh`.
Поскольку штатные команды `docker stop` и `docker kill` расцениваются демоном Docker как преднамеренная остановка пользователем и не активируют политику `restart: unless-stopped`, тест разделен на два независимых этапа:

1. **Проверка деградации (`docker stop`):** останавливает контейнер `uptime-kuma`, дожидается обновления метрики `up{job="uptime-kuma"} == 0` в Prometheus API и делает паузу 15 секунд для визуального контроля красного статуса `DOWN` на дашборде Grafana, после чего возвращает контейнер в строй.
2. **Проверка самовосстановления (`restart-policy`):** выжидает более 10 секунд активной работы контейнера (обязательное условие активации мониторинга рестартов в Docker Engine), убивает рабочие процессы приложения внутри контейнера в обход `PID 1` (`dumb-init`), проверяет автоматический инкремент счетчика `.RestartCount` демоном Docker и возврат метрики `up == 1`.

**Запуск Chaos-теста:**
```bash
sh ./scripts/chaos-test.sh
```