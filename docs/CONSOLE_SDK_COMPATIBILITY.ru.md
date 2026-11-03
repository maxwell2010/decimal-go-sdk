# MintCandy Console: границы Go SDK, 2026-11-03

Сверка выполнена с локальным снимком `D:/my_codex/OneDriveProjects/ChatGPT/CandyConsole/go-console`: README, inventory/routes.json, frontend-functions.json и исходниками web/src. Перечни 22 маршрутов и 177 экспортов не считаются подтверждением функционального паритета. Снимок `sdk` внутри консоли не изменялся. Основной SDK содержит также локальные изменения после публичного preview.12; их нельзя считать автоматически установленными в консоль.

| Требование | Доступные API и границы |
|---|---|
| Неподписанные операции | `NativeTransfer`, контрактные `Build*` возвращают `Call`; `Client.Prepare` возвращает `Prepared.Transaction`, From и Fee без signer. `ContractWithABI` позволяет явно выбрать проверенную схему. Не вызывать Send/Broadcast из миграционного backend |
| WASM | `GOOS=js GOARCH=wasm go build .` прошел локально для основного пакета. Это компиляция библиотеки, не проверка работы в браузере. Нет готового `syscall/js`-моста, JSON-протокола или адаптера EIP1193/MetaMask. Их нужно реализовать и отдельно протестировать |
| EIP712 Permit без ключей | `PermitTypedData` и `NFTPermitTypedData` строят typed data офлайн; `SplitSignature` разбирает переданную подпись. Подписывает браузерный кошелек. `CheckedPermit`/`CheckedNFTPermit` требуют KeySigner и не подходят как серверный маршрут получения ключей. Браузерный путь должен отдельно проверить chain, owner, nonce, фактический DOMAIN_SEPARATOR и сроки, как это делают checked-методы |
| Комиссии | `Prepare`/`FeeQuote` различают EstimatedFee, ReservedFee, RequiredBalance; `WaitReceipt`/`Mined.ActualFee` отражают подтвержденный receipt. Для dynamic fee без BaseFeeEstimate оценка может быть nil. ReservedFee не выдавать за фактически списанную комиссию |
| Точные суммы и Max | `ParseUnits`/`FormatUnits`, `big.Int`, сырые единицы. Готового универсального Max-планировщика нет. Для DEL нужно учитывать fee reserve и повторную оценку; для ERC20 gas оплачивается отдельным балансом DEL. Не использовать float64/JS Number и не считать balance-reserve формальной гарантией успеха |
| Stake7/8 | Встроенный `Delegation` поддерживает шесть getter-вариантов, frozen tuples/arrays, HoldStartTime отдельно от HoldTimestamp; nil означает отсутствие поля в legacy. Ошибки не подменяются нулем. `ReadAt` фиксирует блок |
| Unified NFT | `UnifiedNFTDelegation`, отдельный `NFTStake` tuple10, `ReadNFTStakeAt`/`ReadNFTStakeIDAt`. Старые позиции доступны через `DelegationNFT`. Приложение хранит origin/адрес/сеть позиции; SDK не сканирует оба контракта и не угадывает origin |
| NFT quantity/reserve/hold | `ERC1155.ReadBalanceOf/ReadBalanceOfBatch`, `ERC721.ReadOwnerOf`, `ReadGetReserve` обоих профилей и NFTStake дают отдельные исходные поля. ReadGetReserve возвращает Token/Amount/ReserveType. Универсального нормализатора quantity, индивидуального резерва, доступного остатка и активных HOLD нет; семантику Amount на экземпляр/количество нельзя выдумывать по одному имени поля |
| Reserve-free токены | Есть явный versioned TestnetReservelessToken20261029 и balance getter. Он ограничен testnet и не заменяет автоматически mainnet ABI. Баланс сам по себе не устанавливает наличие резерва |
| minTotalSupply | У Token есть `ReadMinTotalSupply`, но это getter токена. В проверенных ContractCenter/TestnetContractCenter20261029 нет прямого getter minTotalSupply. Контрактный источник лимита из Contract Center нужно подтвердить точной схемой/маршрутом; нельзя подставить значение из токена или константу вместо него |
| Allowance | `PlanERC20Action`, `PlanERC20Allowance` разделяют approval/action, привязывают spender к action.To; второй поддерживает resetExisting. Это данные плана, не автоматическое разрешение. После подтверждения approval нужно заново оценить действие. Generic Multicall не разрешается как ERC20 spender |
| Делегирование/новый HOLD | `Delegation.BuildDelegateByPermit`, BuildDelegateHold, BuildDelegateHoldByPermit, BuildDelegateHoldDEL и обычные сборщики доступны по закрепленному ABI. Для conversion есть versioned reserve builders. Точный Permit workflow зависит от конкретного token/deployment, а не от наличия слова Permit в ABI |
| Существующий стейк в HOLD | `Delegation.BuildHold(validator, token, amount, previousHold, holdTimestamp)` строит hold(address,address,uint256,uint256,uint256). Не прикреплять новый approve автоматически: это изменение уже существующего стейка. Консоль должна выбрать правильный origin и передать подтвержденные параметры позиции |

## Специализированный SafeMultisend

В `web/src/safeMultisend.js` зафиксированы chain75, адрес `0x9b0658e219636215f7f78a88c241ef3bbff447a9` и runtime hash `0x7784c7a41662304b6f6c183f2169a9b6844a1a2f192021d8f3d1ccbdacfc821e`. Это сведения из локального снимка, не свежая проверка развернутого кода.

Его `sendToken(address,address[],uint256[],bytes)` и `sendTokenWithPermit(address,address[],uint256[],bytes,uint256,uint8,bytes32,bytes32)` совпадают с интерфейсом `RestrictedMultisend`. Передавать эти адрес, chain ID и hash явно через `RestrictedMultisendDeployment`; перед использованием выполнять `Verify`, а не доверять произвольному spender из API. `BuildToken` и `BuildTokenWithPermit` строят calldata. Prepare-методы используют RPC-чтения и симуляцию/проверки, без автоматической отправки.

`BuildMultisendDEL` использует другой маршрут через общий Multicall и не является заменой SafeMultisend консоли. `BuildMultisendERC20` через общий Multicall намеренно отключен. Runtime hash не проверялся в публичной сети в рамках этой сверки.

## Что реализовать в следующем этапе

1. Ограниченный WASM-мост для Call, неподписанной транзакции и typed data с целыми числами в строках, browser-side проверкой account/chain и EIP1193-тестами. Ключи и signer на сервер не передаются.
2. Общий браузерный preflight Permit без KeySigner, с nonce/domain/deadline и тестами смены аккаунта/сети между подготовкой и подписью.
3. Подтвержденные адаптеры NFT quantity/reserve/hold и origin, источник лимитов Contract Center, integer Max с раздельными estimate/reserve/receipt. Неподтвержденные поля возвращают явную ошибку/unknown.
4. Прикладные allowance/Permit-планы для каждого проверенного действия и отдельные тесты, что hold существующего стейка не создает approve.

Прикладные кеши, платежи за описания, media, индексная агрегация и UI остаются в консоли. Никаких реальных подписей, approvals, публичных транзакций или изменений production при сверке не выполнялось.

Существующие примеры: [конверты транзакций](TRANSACTION_ENVELOPES.md), [все pinned builders](TRANSACTIONS.md), [unified NFT](UNIFIED_NFT_EXAMPLES.md), [исполняемый NFT-пример](../examples/unified_nft/main.go), [совместимость стейков](DELEGATION_UPGRADE.ru.md).
