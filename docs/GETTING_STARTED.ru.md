# Начало работы

[Установка и обновление одной командой](INSTALL.ru.md) | [Все ABI write-примеры](TRANSACTIONS.ru.md) | [Типы 0/1/2](TRANSACTION_ENVELOPES.ru.md) | [Cosmos-сообщения](../cosmos/TRANSACTIONS.ru.md)

Требуется Go 1.26.10 или новее. Версия для предварительного использования: `v0.1.0-preview.8`.

## Установка

Выполните в новом каталоге. У вложенного Cosmos-модуля отдельный тег `cosmos/v0.1.0-preview.8`; команда Go выбирает его по пути модуля и обычному номеру версии ниже.

```sh
go mod init example.com/decimal-demo
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.8
# Необязательный самостоятельный Cosmos-модуль:
go get github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.8
```
## Сборка вызовов без сети

Сохраните полную программу как `main.go` и выполните `go run .`. Адреса здесь тестовые примеры, а не рекомендуемые развертывания. Builders кодируют намерение без подписи, получения nonce или отправки. Нулевой Client используется только для сборки контрактных вызовов; для чтения и подготовки подключайтесь через `Dial`.

```go
package main

import (
    "fmt"
    "math/big"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    recipient, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
    if err != nil { panic(err) }
    tokenAddress, err := decimal.ParseAddress("0x2000000000000000000000000000000000000002")
    if err != nil { panic(err) }
    amount, err := decimal.ParseUnits("1.25", 18)
    if err != nil { panic(err) }
    native, err := decimal.NativeTransfer(recipient, amount, "invoice-123")
    if err != nil { panic(err) }

    // Offline builder only: no RPC backend is configured.
    builder := new(decimal.Client)
    token, err := builder.Token(tokenAddress)
    if err != nil { panic(err) }
    // Explicit raw token units; discover decimals separately for a real token.
    transfer, err := token.BuildTransfer(recipient, big.NewInt(125))
    if err != nil { panic(err) }
    fmt.Println(native.Value, string(native.Data))
    fmt.Printf("token=%s calldata=%x\n", transfer.To, transfer.Data)
}
```
## Чтение баланса

Эта отдельная программа `main.go` выполняет только запросы чтения к официальному testnet RPC из конфигурации. Нужен интернет, ключ и средства не нужны. `Dial` проверяет chain ID.

```go
package main

import (
    "context"
    "fmt"
    "time"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    client, err := decimal.Dial(ctx, decimal.Testnet())
    if err != nil { panic(err) }
    defer client.Close()
    address, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
    if err != nil { panic(err) }
    balance, err := client.Balance(ctx, address)
    if err != nil { panic(err) }
    fmt.Println(decimal.FormatUnits(balance, 18), "DEL")
}
```
## Подготовка и отправка

`client.Prepare(ctx, from, call, decimal.TxOptions{...})` выполняет RPC-проверки и оценку газа/комиссии, возвращая неподписанную транзакцию. Подписи и отправки нет. `client.Send(ctx, signer, call, opts)` подписывает и отправляет транзакцию; его нужно вызывать явно. Успешная подготовка не доказывает успешное исполнение бизнес-операции. В режиме только чтения не вызывайте Send или Cosmos Broadcast. Примеров отправки здесь нет.

| TxOptions.TxType | Формат | Ограничения |
|---|---|---|
| 0, по умолчанию | Legacy gas price | Без access list; GasPrice задается явно или запрашивается у RPC. |
| 1 | EIP-2930 | Явные access list и gas price; нужна поддержка сети. |
| 2 | EIP-1559 | Явные FeeCap и TipCap; нужна поддержка сети. BaseFeeEstimate - явный параметр оценки, а не гарантия комиссии. |

Все комиссии задаются целыми числами в базовых единицах. До отдельно разрешенной отправки задайте `MaxFee` и соответствующие ограничения gas price. Оценка комиссии отличается от резервируемого максимума; проверяйте `Prepared.Fee`. Построение формата и локальные тесты не подтверждают его приемку каждым развертыванием Decimal.

## Необязательные Cosmos-кодеки

После установки вложенного модуля запустите эту отдельную программу. Она намеренно упаковывает пустое историческое сообщение для демонстрации схемы и классификации маршрута. Это не валидная бизнес-транзакция. Для заполненных сообщений вызывайте `ValidateBasic`; порядок подписантов, роли и допуск целевой сети проверяются отдельно.

```go
package main

import (
    "fmt"
    cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"
)

func main() {
    message := cosmos.NewMsgSendCoin()
    packed, err := cosmos.Pack(message)
    if err != nil { panic(err) }
    fmt.Println(packed.TypeUrl)
    fmt.Println(cosmos.PinnedNativeRoute(packed.TypeUrl)) // false
}
```

В EVM-пакете 524 ABI-обёртки, включая чтение и административные методы. Это не 524 типа бизнес-транзакций. В Cosmos доступны 43 зафиксированные схемы сообщений; ante текущего зафиксированного node допускает только восемь как самостоятельные native-запросы. Наличие кодека не означает поддержку исполнения.

Ограничения ABI и результаты локальных проверок: [ADDITIONAL_ABI_EVIDENCE.md](ADDITIONAL_ABI_EVIDENCE.md), [VALIDATION.md](VALIDATION.md). Во время реализации и аудита реальные транзакции в сеть не отправлялись.
