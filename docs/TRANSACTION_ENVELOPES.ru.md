# Транзакционные конверты и ручные сценарии

[English](TRANSACTION_ENVELOPES.md)

## Запускаемые неподписанные конверты

Из корня репозитория:

```sh
go run ./examples/envelopes
go test ./examples/envelopes
```

[Исходник](../examples/envelopes/main.go) вызывает настоящий `Client.Prepare` с локальным `Backend`. Сетевого транспорта и signer нет. Методы отправки и получения receipt завершаются panic, если их случайно вызвать. Все три случая независимо используют nonce 7. `Prepare` не резервирует nonce: это альтернативные примеры сериализации, а не три готовые к совместной отправке транзакции.

| Тип | Явные параметры | Результат на фикстурах |
|---|---|---|
| 0 Legacy | `TxType: types.LegacyTxType`, рекомендованная цена 10 wei | Gas 23100, оценка комиссии 210000 wei, резерв 231000 wei |
| 1 EIP-2930 | `TxType: types.AccessListTxType`, `AccessList`, `GasPrice: 10` | Та же комиссия фикстуры, chain ID 202020 и access list сохранены |
| 2 EIP-1559 | `TxType: types.DynamicFeeTxType`, `FeeCap: 20`, `TipCap: 2`, `BaseFeeEstimate: 5` | Эффективная оценка цены 7 wei, оценка комиссии 147000 wei, резерв 462000 wei |

Backend возвращает оценку gas 21000 во всех случаях. Стандартный множитель SDK 11000 базисных пунктов даёт лимит 23100. В реальной сети access list меняет intrinsic gas; фикстура не моделирует исполнение. Резерв динамической комиссии использует cap, оценка - `min(cap, base estimate + tip)`. Без `BaseFeeEstimate` оценка динамической комиссии остаётся неизвестной. Все суммы заданы целыми базовыми единицами.

Тест проверяет тип, nonce, отсутствие подписи, gas, комиссии, chain ID, access list и бинарную сериализацию. Сборка не подтверждает поддержку typed-транзакций конкретным развертыванием Decimal. Тип 0 выбран по умолчанию, типы 1 и 2 включаются явно. Другие EVM-типы `Prepare` отклоняет.

## API ручных сценариев

Ниже указаны реальные публичные helpers. Они не запускают автоматическую отправку. Адреса и поколения deployment ABI задаются явно; офлайн-сборка не подтверждает права, балансы или исполнение. [Все ABI builders](API.md), [versioned-каталог](token-deployment-catalog.json), [границы приёмки](ACCEPTANCE.md).

| Сценарий | Публичный API | Граница |
|---|---|---|
| Перевод DEL | `NativeTransfer(to, value, memo)` | `value` в wei, memo записывается в data. [Пример](../examples/offline_transfer/main.go). |
| Сжигание DEL | `NativeTransfer(common.Address{}, value, "")` | Намерение перевода на нулевой адрес; отправки нет. |
| Рассылка DEL | `client.BuildMultisendDEL([]decimal.Recipient{...})` | Один получатель - native transfer; несколько - настроенный Multicall с общей суммой DEL. ERC20 allowance произвольному Multicall запрещён. |
| ERC20 transfer | `client.Token(address)`, затем `token.BuildTransfer(recipient, amount)` | Проверяйте decimals настоящего токена. [Пример](../examples/token_transfer/main.go). |
| Сброс approval | `client.PlanERC20Allowance(ctx, token, owner, spender, amount, action, true)` | Читает allowance. `AllowancePlan.Approvals` может содержать reset в ноль и целевой approval. Каждый шаг требует отдельного успешного receipt; `Action` остаётся отдельным. |
| ERC20 permit | `PermitTypedData(name, version, chainID, token, permit)`, `key.SignTypedData(ctx, typed)`, `SplitSignature(bytes)` | Только подпись. `client.CheckedPermit(ctx, key, token, name, version, permit)` дополнительно читает nonce/domain. Owner, spender, nonce, deadline должны соответствовать контракту. |
| Подпись Safe | `NewSafeTransaction(call, nonce)`, `tx.Hash(chainID, safe)`, `tx.Sign(ctx, key, chainID, safe)` | Подпись не подтверждает полномочия. `ReadBigSafePolicy` и `PackBigWeightedSafeSignatures` проверяют веса/threshold/nonce; `BuildExecution(contract, signatures)` требует проверенный ABI. |
| EVM Checks | `EVMCheckSignerFromPassword`, `SignEVMCheckRedemption`, `VerifyEVMCheckRedemption`, `checks.BuildSignedRedemption` | Подтверждённый JS-профиль с EIP-191 и привязкой к redeemer. Настоящий checkHash задаётся явно. [Помощник и пример](EVM_CHECKS.ru.md); это не Cosmos/RLP и не универсальная подпись всех поколений контрактов. |
| Историческая Cosmos-подпись чека | Вложенный модуль: `cosmos.IssueCheck(ctx, chainID, denom, amount, nonce, dueBlock, passphrase, issuerKey, sign)` | Другой протокол, не подпись EVM Checks. Используйте детерминированные тестовые данные и callback; примерам не нужны пользовательские секреты. |

Порядок approval, весовые/контрактные подписи Safe и ограниченная рассылка описаны в [Safe workflows](SAFE_WORKFLOWS.md). Protobuf-конструкторы и исторические маршруты - в [Cosmos README](../cosmos/README.md). Ни один пример здесь не вызывает `Send` или `Broadcast`, не загружает пользовательский ключ и не пишет в сеть.
