# Decimal Go SDK

Самостоятельный Go-модуль `github.com/maxwell2010/decimal-go-sdk`, созданный по исходникам официальной ноды, JS SDK и Go SDK с учетом сценариев наших Python и C# SDK. Требуется Go 1.26.10 или новее. Текущая публичная версия - `v0.1.0-preview.5`, предварительная версия без заявления о полном соответствии развернутым контрактам.

Разработчики: [MintCandy](https://mintcandy.ru/) и [@Maxwell2019](https://t.me/Maxwell2019). Лицензия MIT.

[English README](README.md) | [Начало работы](docs/GETTING_STARTED.ru.md) | [Примеры](examples/README.ru.md) | [Изменения](CHANGELOG.md)

В пакете 11 зафиксированных ABI-профилей и 300 сгенерированных методов: 157 сборщиков транзакций и 143 метода чтения с типизированными аргументами и результатами. Это число контрактных методов, включая административные, а не количество пользовательских сценариев. [Каталог API](docs/API.md), [матрица операций](docs/OPERATION_MATRIX.md), [первоисточники и хеши](docs/sources.json).

Полный сетевой parity пока не подтвержден. В текущих официальных исходниках есть несовместимые поколения ABI. Из 93 активных write-методов JS для 55 найден хотя бы один соответствующий сборщик по зафиксированному ABI; 35 требуют явно переданного ABI развернутого контракта, два являются нативными переводами, один - смешанной рассылкой с отключенным небезопасным ERC20-маршрутом. Это классификация покрытия, не обещание одинакового поведения всех вариантов.

Дополнительно доступны девять явно выбираемых versioned testnet-профилей: 224 метода системных контрактов, Safe/factory/MultiSend и резервных/безрезервных токенов. Адреса, блоки, codehash и хеши ABI записаны в [системном manifest](docs/deployment-sources.json), [Safe manifest](docs/additional-deployment-sources.json) и [token manifest](docs/token-deployment-sources.json). ABI-интеграция охватывает все 524 метода; это не проверка исполнения в сети.

## Совместимость с сетью без отправки транзакций

[Матрица совместимости](docs/COMPATIBILITY.ru.md) отдельно показывает локальные тесты ABI, ответы getters, исторические события Checks и ограниченные eth_call-симуляции. Для обеих сетей сохранены блоки и исходные RPC-хеши. Настроенный mainnet-адрес Checks требует проверки deployment: наличие initialized proxy не подтверждено. Адреса по умолчанию не менялись.

Для RPC-хеша блока используйте `Client.BlockIdentity`: `types.Header.Hash()` на Decimal может отличаться. `ChecksCreationHistory` и `ChecksCreationReceipt` читают настоящие хеши создания без кошелька. [API и пример](docs/CHECKS_HISTORY.ru.md).

## Что реализовано

- DEL: перевод с memo, сжигание переводом на нулевой адрес, рассылка DEL.
- Токены: ERC20 transfer/approve/transferFrom, создание резервных и безрезервных токенов, buy/sell с exact-input/exact-output, convert с permit и без, обновление через `updateDetails` текущего ABI. Mint безрезервного токена использует его отдельный deployment ABI.
- Стейкинг: delegate/hold/withdraw/transfer/reset/complete, permit-варианты, обычные, held и frozen стейки. Подтвержденные индексы frozen можно читать пакетно одним RPC-вызовом на фиксированном блоке. Зрелая заявка не становится завершенной автоматически.
- NFT: ERC721/1155, single/batch transfer, approvals, mint, reserve, burn, коллекции, чтение metadata и состояний, отдельный DelegationNFT. Методы старых и новых поколений не подменяются друг другом.
- Валидаторы: кандидат, validator, pause/unpause, metadata, статусы, penalty/reward/owner и другие методы ABI.
- Чеки: nonce, создание, permit и погашение по ABI официального JS SDK; подпись EIP-191 для конкретного получателя, восстановление подписанта и проверенная офлайн-сборка proofs. [Формат и пример](docs/EVM_CHECKS.ru.md).
- Safe: EIP-712 hash/sign, проверка ожидаемых подписантов, сортировка и отклонение повторов, сборка исполнения по deployment ABI, factory proxy с явным initializer. Весовые owners и threshold проверяет приложение по развернутому Safe.
- Bridge/GasCenter: сборщики по явно переданному deployment ABI; целочисленные разрядности берутся из него. Передаются явные proof, nonce, fee и deadline.
- RPC/HTTP: контексты, блоки, балансы, код, логи, receipt, подписки через WebSocket backend; ограниченные HTTP-ответы и пагинация, произвольные относительные REST-маршруты, проверка авторитетности withdrawal index.

Legacy Cosmos protobuf-транзакции из `go-smart-node/sdk/tx` - другой протокол. Они не реализованы как EVM-вызовы и не входят в заявленное покрытие этого модуля. IPFS-загрузка и contract verification тоже не являются транзакциями и не включены.

Для Cosmos добавлен [отдельный модуль](cosmos/README.md) с 43 схемными конструкторами сообщений и транзакционными envelope. Его тесты запускаются отдельно. Общая команда для обоих модулей: `pwsh -File tools/check.ps1 -Race -Vulnerabilities`. Точные границы реализации и сетевой приемки указаны в README Cosmos.

В Cosmos теперь доступны gRPC и source-backed `ValidateBasic` для всех 43 схем. Stateful-роли проверяет целевая нода; локальная валидация не является авторизацией. [Статические правила и differential-проверки](docs/COSMOS_VALIDATION.md).

Для EVM добавлены полные uint256-веса Safe, contract signatures/ERC-1271, reset allowance и явные proxy/beacon loaders с проверкой кода на фиксированном блоке. Опциональная sender-restricted рассылка - расширение нашего проекта по C# contract evidence, а не официальный Decimal default; адрес/chain/runtime hash задаются явно. Подробности в [SAFE_WORKFLOWS.md](docs/SAFE_WORKFLOWS.md). Неполученные официальные ABI и проверенные пути поиска перечислены в [ADDITIONAL_ABI_EVIDENCE.md](docs/ADDITIONAL_ABI_EVIDENCE.md).

## Установка и обновление одной постоянной командой

В существующем Go-проекте:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

Команда устанавливает либо обновляет **сразу EVM и Cosmos** до опубликованной версии из [latest.json](latest.json), включая preview. Номер релиза в команде менять не нужно. Обновление происходит только при вашем запуске. Флаг `-print` показывает выбранную команду без изменения проекта, `-evm-only` / `-cosmos-only` выбирают один модуль. В новом проекте сначала выполните `go mod init example.com/myapp`. [Полная инструкция по установке, обновлению и откату](docs/INSTALL.ru.md).

[Все 260 ABI write builders](docs/TRANSACTIONS.ru.md) | [EVM-форматы 0/1/2](docs/TRANSACTION_ENVELOPES.ru.md) | [Все 43 Cosmos-сообщения](cosmos/TRANSACTIONS.ru.md). Примеры выполняются офлайн и не подтверждают исполнение в сети.

## Установка фиксированной версии

Требуется Go 1.26.10 или новее. В Go-модуле приложения выполните:

```sh
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.5
```

Независимый Cosmos-модуль подключается отдельно при необходимости:

```sh
go get github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.5
```

Go установит зависимости автоматически. На [странице релиза](https://github.com/maxwell2010/decimal-go-sdk/releases/tag/v0.1.0-preview.5) доступны архивы ZIP/TAR.GZ и `SHA256SUMS`. Для использования SDK не нужно генерировать ABI-обертки. Полные примеры кода приведены в [руководстве](docs/GETTING_STARTED.ru.md) и [каталоге примеров](examples/README.ru.md).

Для разработки клонируйте репозиторий. Сам модуль проверяется командами:

```sh
go test ./...
go vet ./...
go run ./cmd/decimal-inspect -network testnet
```

Последняя команда только читает сеть; кошелек ей не нужен. `Dial` проверяет chain ID: mainnet 75, testnet 202020. Дочерние системные контракты разрешаются через `ResolveSystemContract`, без наследования адресов другой сети.

## Сборка и отправка

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
client, err := decimal.Dial(ctx, decimal.Testnet())
if err != nil { return err }
defer client.Close()

// Секрет поступает от приложения. Не записывайте его в код или логи.
signer, err := decimal.NewKeySigner(privateKeyFromSecretStore)
if err != nil { return err }
defer signer.Close()
to, err := decimal.ParseAddress(recipientAddress)
if err != nil { return err }
amount, err := decimal.ParseUnits("0.01", 18)
if err != nil { return err }
call, err := decimal.NativeTransfer(to, amount, "invoice-123")
if err != nil { return err }

// Сначала оценка. Ни подписи, ни отправки здесь нет.
prepared, err := client.Prepare(ctx, signer.Address(), call, decimal.TxOptions{})
if err != nil { return err }
fmt.Println(prepared.Fee.EstimatedFee, prepared.Fee.ReservedFee)

// Следующая строка действительно отправляет транзакцию.
sent, err := client.Send(ctx, signer, call, decimal.TxOptions{})
if err != nil {
    // При SubmissionUnknownError сверяйте sent.Hash: исход отправки неизвестен.
    return err
}
mined, err := client.WaitReceipt(ctx, sent.Hash, time.Second)
if err != nil { return err } // revert не считается успешным receipt
fmt.Println(mined.ActualFee) // nil, если RPC не вернул effectiveGasPrice
```

Импорты примера: `context`, `fmt`, `time` и `decimal "github.com/maxwell2010/decimal-go-sdk"`. Пример намеренно не содержит ключа, адреса получателя или автоматического запуска.

`SignerFromMnemonic(words, passphrase, path)` использует BIP39/BIP32; путь по умолчанию `m/44'/60'/0'/0/0`. Создавайте signer один раз на серию подписей. Он реализует интерфейс `Signer`; для аппаратного кошелька или внешнего хранилища можно передать свою реализацию. Клиент проверяет, что signer не изменил отправителя или intent.

## Стейкинг и токены

```go
address, err := client.ResolveSystemContract(ctx, "delegation")
if err != nil { return err }
staking, err := client.Delegation(address)
if err != nil { return err }
call, err := staking.BuildDelegateDEL(amountWei, validator)
if err != nil { return err }
// call можно оценить, симулировать и только затем явно отправить.

token, err := client.Token(tokenAddress)
if err != nil { return err }
transfer, err := token.BuildTransfer(recipient, amountRaw)
if err != nil { return err }
_ = transfer
```

Все контрактные amounts - целые базовые единицы. Не используйте `float64`. `PlanERC20Action` возвращает отдельный approve при нехватке allowance. Дождитесь его успеха и заново оцените основную операцию. Для токенов, требующих сначала обнулить существующий allowance, приложение должно выполнить этот дополнительный шаг явно. `CheckedPermit` сверяет nonce и `DOMAIN_SEPARATOR`; версию и deadline задает приложение.

По умолчанию используется legacy EIP-155. Явный `TxOptions.TxType=1` выбирает EIP-2930, `TxType=2` - EIP-1559 с `FeeCap`/`TipCap`. Для EIP-1559 резерв считается по fee cap; без явного `BaseFeeEstimate` фактическая эффективная оценка неизвестна. Поддержку формата целевой сетью нужно проверить; автоматического переключения нет.

`FrozenWithdrawalsBatch(ctx, wallet, indexes, block)` проверяет до 500 известных индексов за вызов. Для deployment без batch getter есть `FrozenWithdrawals`. Индексы должны поступать из полного авторитетного источника. `API.WithdrawalIndex` отклоняет stale/unavailable/incomplete; пустой авторитетный список остается пустым списком.

## ABI конкретного развертывания

```go
contract, err := client.ContractWithABI(deploymentAddress, verifiedABI)
if err != nil { return err }
call, err := contract.BuildReservelessMint(recipient, amountRaw)
if err != nil { return err }
```

Для остальных эталонных сценариев есть `BuildReferenceOperation(name, value, args...)`: имя берется из матрицы JS, аргументы идут в порядке **ABI**, не JS-wrapper. Для overload используйте `Build(fullSignature, value, args...)`. Неизвестные методы и неправильные разрядности отклоняются до RPC. `API.DeploymentABI` читает ABI официального compact endpoint, но применение через `ContractWithABI` остается явным решением приложения.

## Границы проверки

Тесты без сети проверяют wire-format JSON-RPC, подпись и calldata, nonce при конкурентной отправке, пределы комиссий, неизвестный исход отправки, receipt/revert, ABI tuple и integer bounds, события, API-маршруты и Safe-вектор из C#/официального JS. Сетевые результаты записаны в [VALIDATION.md](docs/VALIDATION.md). Проверок реальной отправки здесь нет.

Перед публикацией потребуется выбрать публичный module path. Статус лицензирования импортированных ABI описан в `THIRD_PARTY_*`; в частности, JS snapshot объявляет ISC, но отдельного LICENSE в нем нет.
