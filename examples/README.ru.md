# Запускаемые примеры

Запускайте команды из корня репозитория с Go 1.26.10 или новее.

- `go run ./examples/offline_transfer` собирает намерение перевода DEL и выводит сумму и байты memo.
- `go run ./examples/token_transfer` собирает ERC20 `transfer` calldata с демонстрационными адресами и разрядностью.
- `go run ./examples/read_balance` читает публичный баланс testnet. Предварительно задайте публичный EVM-адрес в `WALLET_ADDRESS`.

Пример PowerShell:

```powershell
$env:WALLET_ADDRESS = '0x1000000000000000000000000000000000000001'
go run ./examples/read_balance
```

Офлайн-примеры не используют RPC. Чтение баланса использует официальный testnet endpoint, проверяет chain ID и ограничено 15 секундами. Примеры переводов и ABI-конвертов не подписывают транзакции. Новый пример Checks подписывает только сообщение публичной фикстуры; пользовательские ключи не загружаются и транзакции не отправляются. Демонстрационные адреса не подтверждают наличие развернутого токена или владение получателем.

[English](README.md)

- `go run ./examples/envelopes` собирает неподписанные EVM-типы 0/1/2 через SDK и локальный Backend.
- `go run ./examples/transactions` вызывает все 260 ABI write builders в 20 профилях полностью офлайн.
- `go -C cosmos run ./examples/messages` проверяет и кодирует 43 публичные схемные фикстуры: 8 текущих native-маршрутов и 35 исторических/схемных.

[Установка и обновление](../docs/INSTALL.ru.md), [код каждого builder](../docs/TRANSACTIONS.ru.md), [конверты и сценарии](../docs/TRANSACTION_ENVELOPES.ru.md), [Cosmos payload](../cosmos/TRANSACTIONS.ru.md). Подписи, proofs и права в фикстурах не подтверждают исполнение бизнес-транзакций.

- `go run ./examples/checks_signing` выводит proof для публичной фикстуры и собирает calldata без RPC и отправки. [Формат подписи](../docs/EVM_CHECKS.ru.md).
