# Совместимость стейков после обновления 29-30.10.2026

Исходник ноды зафиксирован на commit `a46cb48d7965c1a64a495cebb22e735dfe4b9da3`. Ссылки, SHA256 исходников и извлеченного ABI находятся в [manifest](DELEGATION_UPGRADE_SOURCES.json). Это адаптация декодирования по официальным исходникам, без утверждения о фактической активации конкретного deployment.

В `Delegation` tuple `Stake` теперь содержит восемь полей:

```text
validator, delegator, token, amount, tokenId, tokenType, holdTimestamp, holdStartTime
```

`amount` остается в индексе 3, `holdTimestamp` в индексе 6. Новое поле в индексе 7 - время начала холда, не сумма. SDK не делит суммы на 1000 и возвращает исходные целые значения.

Типизированные методы `ReadGetStake`, `ReadGetStakes`, `ReadGetHoldStake`, `ReadGetHoldStakeDEL`, `ReadGetFrozenStake`, `ReadGetFrozenStakes` сохраняют поддержку зафиксированного старого ABI и принимают соответствующие новые ответы. Поддерживаются вложенные структуры и массивы. В расширенных структурах Go `HoldStartTime == nil` означает, что старый ответ не содержит поля. У нового ответа значение `0` представлено ненулевым указателем на целое 0. Эти состояния различаются.

```go
delegation, err := client.Delegation(deploymentAddress)
if err != nil {
    return err
}
result, err := delegation.ReadGetStakes(ctx, stakeIDs) // [][32]byte
if err != nil {
    return err // ошибка чтения не означает нулевой стейк
}
for _, stake := range result.Value0 {
    fmt.Println(stake.Amount.String(), stake.HoldTimestamp.String())
    if stake.HoldStartTime != nil {
        fmt.Println("hold start:", stake.HoldStartTime.String())
    }
}
```

Сигнатуры методов и selector `getStakes(bytes32[])` не изменились. Декодер проверяет полный канонический ABI-ответ, включая offset, count, длину, padding и отсутствие лишних байтов. Поврежденные данные возвращают ошибку. Он не подменяет недоступную позицию нулевой и не отключает ABI-проверки. Для чтения на конкретном блоке применяется существующий `Contract.ReadAt`.

События с `Stake` имеют разные `topic0` для старого и нового tuple. `Delegation.DecodeLog` принимает оба официальных поколения и возвращает исходные `Signature`, `Values` и `Log`. Старые события сохраняют семь полей, без выдуманного времени начала. Новые события сохраняют восемь. Нода нормализует старые события внутри hooks, но это не меняет исходный receipt. Также декодируются новые официальные события unified NFT; это поддержка формата события, а не смена маршрута транзакций.

Официальный binding `DelegationNFT` в том же commit по-прежнему использует семь полей. Его ABI и сборщики выхода не заменяются на unified `Delegation`. Существующие старые NFT-позиции требуют своего маршрута. При chain ID 75 встроенный профиль отклоняет новые delegate/hold/permit-вызовы на известный retired адрес `0xe45adfcc739a0d10ce9462b58866c9a1a06035e2` еще при построении вызова. Чтения и исторические выходы доступны. Другие адреса, сети и явно переданный ABI не перенаправляются. SDK не выполняет автоматическую миграцию, повторную обработку истории или смену владельца NFT. ERC721 `ownerOf` нельзя автоматически использовать для ERC1155.

Для новых NFT-позиций явно выбирается `client.UnifiedNFTDelegation(unifiedProxy)`. В отличие от Python, в Go нет NFT-сервиса с неявным адресом из config: вызывающее приложение выбирает deployment само. Конструктор не обращается в сеть и не меняет глобальную конфигурацию. `ReadNFTStakeAt` возвращает отдельную структуру `NFTStake` из десяти полей; это не Stake7/8. `ReadNFTStakeIDAt` и чтение позиции требуют конкретный блок. Доступны 13 официальных сборщиков delegate/hold/transfer/withdraw, включая имеющийся в ABI ERC721 permit. Отсутствующие варианты ERC1155 permit не выдумываются.

```go
unified, err := client.UnifiedNFTDelegation(unifiedProxy)
if err != nil {
    return err
}
action, err := unified.BuildDelegateNFT(validator, collection, tokenID)
if err != nil {
    return err
}
approval, err := unified.BuildERC721Approval(collection, tokenID)
if err != nil {
    return err
}
// action.To и approval operator совпадают с unified.ApprovalOperator().
// Это только данные вызовов для проверки, без подписания и отправки.
fmt.Println(action.To, approval.To, unified.ApprovalOperator())
```

`BuildERC721Approval` строит разрешение на конкретный token ID. Для ERC1155 есть явно названный `BuildERC1155ApprovalForAll` с разрешением на всю коллекцию. Оба берут operator из выбранного unified-контракта; отправка разрешения остается отдельным действием приложения. Старые позиции читаются и выводятся через `client.DelegationNFT(legacyProxy)`.

`Contract.DecodeDelegationReceipt` фильтрует логи по адресу контракта, сохраняет порядок и несколько выходов с одинаковым stake ID и разными stake index. Неизвестные события пропускаются; поврежденное известное событие возвращает ошибку без частичного результата. Исходный receipt не меняется.

Исходные профили в `abi/` и `ABIJSON("Delegation")` остаются зафиксированными артефактами старого поколения. Дополнительный официальный ABI находится в `delegation_compat/`. Адаптация применяется только к встроенному профилю `Delegation`; явно переданный `ContractWithABI` сохраняет ABI вызывающего приложения. В SDK нет кэша стоимости стейков или валидаторного агрегатора. Приложению следует отдельно проверять полноту и свежесть RPC-данных, состояние валидатора, поколение контракта и актуальность своего кэша.

Проверки используют синтетические ABI-ответы и логи, а также переданные из соседнего проекта сохраненные mainnet RPC-данные: восемь восьмипольных стейков на блоке `0x20c2060`, 16 логов (`StakeUpdated` и `StakeAmountUpdated`) и два NFT view probes для пустой позиции. Копии и SHA256 включены в manifest; эти RPC-запросы заново здесь не выполнялись. Тест сопоставляет суммы в событиях и ответе getter, отдельно сверяет поля с исходными ABI-словами. Это подтверждает декодирование образцов и отказ на поврежденных данных, но не полный жизненный цикл NFT с реальным резервом. Публичные транзакции и продакшен-деплой для этой проверки не выполняются.
