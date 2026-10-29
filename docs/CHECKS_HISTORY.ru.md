# Чтение существующих событий EVM Checks

[English](CHECKS_HISTORY.md) | [Подпись](EVM_CHECKS.ru.md)

Кошелёк, пароль и отправка транзакции не нужны. Декодер использует закреплённый официальный JS ABI: CreatedCheck содержит один topic и 128 байт данных, CreatedCheckDetails - один topic и 192 байта. Проверяются адрес, точное ABI-кодирование, отсутствие Removed и идентификаторы блока/транзакции. Неизвестные числовые статусы сохраняются без предположений о погашении.

```sh
go run ./cmd/checks-inspect -network testnet -blocks 2000
go run ./cmd/checks-inspect -network mainnet -blocks 2000
```

Верхний блок фиксируется один раз; его можно задать через `-to`. Команда сравнивает исходный RPC block.hash до и после запроса. Пересчитанный types.Header.Hash() не используется: на Decimal он может отличаться от исходного hash. `-address` задаёт deployment явно. Максимум одного запроса - 2000 блоков и 10000 событий; полного сканирования сети нет.

Методы SDK: `client.ChecksCreationHistory(ctx, address, from, to)` и `client.ChecksCreationReceipt(ctx, address, txHash)`. Для готового receipt есть `ChecksCreationsFromReceipt`; отдельные логи читают `DecodeChecksCreationLog` и `DecodeChecksCreationDetailsLog`. Receipt должен быть успешным, а адреса, tx hash, block hash, высота и индекс транзакции должны согласоваться. Возвращаемые буферы копируются. События сортируются; повторные позиции логов отклоняются, одинаковые checkHash в разных исторических позициях не удаляются.

На 2026-10-29 в testnet на блоках 22475438..22477437 найдены шесть событий. В mainnet на блоках 34334454..34336453 событий не найдено - это результат только выбранного окна. [Testnet](compatibility/checks-testnet.json), [mainnet](compatibility/checks-mainnet.json).

Один реальный receipt проверен отдельно по официальному RPC с сопоставлением исходного block.hash: транзакция `0xd670470a9c382456cd9102e7b9a99ecf5a42de07bfa1ad8388615b912c649518`, checkHash `0x58d605b116ba5b5833da55ad0b89b2b20564f982d4afe2e586bfc8256fca1d4c`. [Сохранённый receipt](compatibility/checks-testnet-receipts.json).

Офлайн-чтение сохранённых публичных данных:

```sh
go run ./examples/checks_history
```

Статус CreatedCheck относится ко времени события. Он не доказывает текущее наличие средств, срок, права, отсутствие погашения/отзыва или финальность. Чужой публичный receipt не даёт контроля над подписантом. Формула выпуска хеша не выдумывается; сведения разных поколений контрактов не смешиваются. При этой проверке транзакции не отправлялись и пользовательские секреты не загружались.

## Полная программа чтения известного receipt

Это только чтение официального testnet RPC. Хеш транзакции взят из публичного evidence выше. Ключ, подпись и отправка не нужны. Совпадение RPC block hash подтверждает согласованность ответа провайдера, а не финальность и не текущую погашаемость. Для нового приложения сначала установите SDK; сохраните код как main.go и запустите go run .

```go
package main

import (
    "context"
    "fmt"
    "math/big"
    "time"
    "github.com/ethereum/go-ethereum/common"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()
    cfg := decimal.Testnet()
    client, err := decimal.Dial(ctx, cfg)
    if err != nil { panic(err) }
    defer client.Close()
    tx := common.HexToHash("0xd670470a9c382456cd9102e7b9a99ecf5a42de07bfa1ad8388615b912c649518")
    rows, err := client.ChecksCreationReceipt(ctx, cfg.Checks, tx)
    if err != nil { panic(err) }
    for _, row := range rows {
        block, err := client.BlockIdentity(ctx, new(big.Int).SetUint64(row.Log.BlockNumber))
        if err != nil { panic(err) }
        if block.Hash != row.Log.BlockHash { panic("receipt/RPC block identity mismatch") }
        fmt.Println(row.CheckHash.Hex(), row.Signer.Hex(), block.Number, block.Hash.Hex())
    }
}
```
