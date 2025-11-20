```bash
just setup          # Первый запуск - установит зависимости
just dev-all        # Запустить все сервисы
just restart        # Перезапустить все
just kill-all       # Остановить все
```

```bash
just test                          # Все тесты
just grpc-test                     # Тест gRPC
just http-test-greeter-via-gateway # Тест через Gateway
just http-test-greeter-direct      # Тест напрямую
```

```bash
just build          # Собрать все через Nix
just build-gateway  # Только Gateway
just build-greeter  # Только Greeter
```

```bash
just info      # Показать инфу о проекте
just clean     # Очистка
just fmt       # Форматирование кода
just ps        # Показать запущенные процессы
```


# Frontend Federation

## Как понять, что ты накосячил во фронте

Спроси себя:
- **Могу ли я удалить модуль News и всё остальное продолжит работать?** Если нет - плохо
- **Могу ли я откатить один модуль на старую версию?** Если нет - плохо
- **Могу ли я разрабатывать модули разными командами независимо?** Если нет - плохо
- **Могу ли я запустить модуль в изоляции для тестов?** Если нет - плохо

## Правильный подход

Модули общаются только через:
- **Явные контракты** (минимальный набор props)
- **События** (не знают друг о друге)
- **Shared модуль** (только для критичных общих вещей)
- **Backend API** (каждый со своим, токен в куках)

Принцип: модуль должен **деградировать gracefully** - если другой модуль сломался, он просто не показывает свою часть, но не падает весь сайт.


# Docker hack

Можно не собирать образ, а просто запустить его в alpine примонтировав /nix/store

```bash
nix build .#gateway
```

File: Dockerfile.minimal
```Dockerfile
FROM alpine:latest
WORKDIR /app
CMD ["echo", "Ready. Please provide a command to run, for example, a path from /nix/store."]
```


```bash
docker build -t gateway-base -f Dockerfile.minimal .

docker run --rm -it \
         --name my-gateway \
         -p 8080:8080 \
         -v /nix/store:/nix/store:ro \
         gateway-base \
         "${GATEWAY_PATH}/bin/start-gateway"
```
