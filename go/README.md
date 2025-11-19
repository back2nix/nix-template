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
