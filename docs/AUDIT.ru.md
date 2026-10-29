# Сопоставление источников и принятые решения

Проверено 29.10.2026. Ветки master сверены по сети через `git ls-remote`. SHA, пути исходных ABI и SHA-256 нормализованных интерфейсов находятся в [sources.json](sources.json). Контексты чатов «на PIP» и «C SDK», исходники Python и документы/тесты C# читались как дополнительный материал. Незакоммиченные файлы этих SDK не редактировались.

| Источник | Зафиксированная версия | Назначение |
|---|---|---|
| [go-smart-node](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/) | `9e6c6d7` | Контрактные ABI, registry, chain/fee поведение |
| [dsc-js-sdk](https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/) | `6790d35` | Сценарии пользователя, динамическая загрузка deployment ABI, Checks/Multicall |
| [dsc-go-sdk](https://bitbucket.org/decimalteam/dsc-go-sdk/src/3ef4a089b6020889e60783c2026df5725fb960e2/) | `3ef4a08` | Существующие Go функции и организация отправки |

## Подтвержденные пробелы официального Go SDK

В каталоге `decimalevm/methods` есть 58 экспортированных функций, включая вспомогательные методы сборки/подписи. В активном JS EVM API 93 write entry point. Прежний подсчет 94 включал закомментированный `mintNFT`. Числа не означают количество уникальных протокольных транзакций. Все строки сравнения доступны в [OPERATION_MATRIX.md](OPERATION_MATRIX.md).

По исходникам Go нет именованных методов `CreateTokenReserveless`, `MintTokenReserveless`, отдельных обновлений identity/max/min supply, GasCenter conversion, Checks, Bridge и произвольного Multicall. Не хватает ряда token-staking reset/transfer/hold/completion сценариев, хотя аналоги некоторых NFT-сценариев уже присутствуют. Поэтому переносить общий вывод «нет всех reset/complete» на весь Go SDK неправильно.

В [del.go](https://bitbucket.org/decimalteam/dsc-go-sdk/src/3ef4a089b6020889e60783c2026df5725fb960e2/decimalevm/methods/del.go#lines-27) `SendDEL` использует `context.Background()`, фиксированный gas limit 21000, DynamicFeeTx с расчетом через `Header.BaseFee`, отправку и ожидание receipt внутри одного метода и возвращает только `error`. Новый SDK возвращает intent, fee quote, signed transaction/hash и mined result на отдельных этапах; EIP-1559 не считается обязательным свойством сети.

Новый модуль покрывает все функции 11 импортированных ABI, а отличающиеся deployment-поколения подключаются явно через `ContractWithABI` или `LoadDeploymentContract`. Для 35 JS write-сценариев нельзя доказать ABI только по текущей ноде. Их отсутствие в pinned ABI не означает, что развернутый контракт не поддерживает вызов; наличие метода в JS/Go тоже не доказывает обратное.

## Несовпадения ABI

| Участок | Подтверждено источниками | Решение |
|---|---|---|
| Token updates | JS вызывает отдельные методы identity/max/min; [ABI DecimalToken](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/contracts/token/DecimalToken.json) содержит `updateDetails(string,uint256)` | Текущий метод сгенерирован; отдельные старые selectors требуют deployment ABI |
| NFT creation | JS `createCollection` использует `createDRC721/createDRC1155` и reserveless варианты; [NFTCenter ABI](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/contracts/nftCenter/DecimalNFTCenter.json) содержит `createERC721/createERC1155` с иной meta | Не переименовывать методы и не сокращать tuple автоматически |
| NFT reserve | JS использует `mintByDEL/addReserveByDEL`; node ABI - `mintByETH/addReserveByETH` | Сборщики node ABI и явный адаптер deployment ABI |
| Main-contract NFT staking | В JS есть `delegateNFT`, `delegateNFT1155` и соответствующие hold/withdraw/transfer; они отсутствуют в pinned Delegation ABI | Не выдавать их за DelegationNFT: это другой contract target и другой calldata |
| NFT reset | В Go/JS есть reset-сценарии, отсутствующие в pinned DelegationNFT ABI | Требовать ABI выбранного deployment |
| Reserveless mint | `mint(address,uint256)` нельзя считать методом резервного DecimalToken | Отдельный `BuildReservelessMint` только на явно предоставленном ABI |

## Сети и адреса

[contracts/utils.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/contracts/utils.go#lines-166) задает ContractCenter mainnet `0xc108715a06f76caa96fa2c943ebf05159c29a87d` и testnet `0xbC96b61F137F28F0Da47Cc4Ef06e5f984B565A2E`. Дочерние адреса берутся через `getContractAddress(string slug)` и проверяются на наличие кода, а не наследуются из настроек другой сети.

В [cmd/config/config.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/cmd/config/config.go#lines-22) присутствует константа 2020. Фактический публичный mainnet RPC во время проверки вернул 75, testnet - 202020. Нельзя по одной константе переписать EVM network config. Доказательство сетевого чтения и его пределы записаны в [VALIDATION.md](VALIDATION.md).

Адреса Multicall/Checks/Safe/factory/multiSend взяты из [endpoints.ts](https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/endpoints.ts). Это provenance конфигурации, не проверка их текущей реализации в сети.

## Ограничения и проверки до релиза

Публичные функции нового SDK не являются доказательством успешной отправки. Все Build-методы работают без подписи и сети. В этой работе выполнены тесты без реальных переводов и read-only RPC checks, без funded-wallet testnet кампании.

Legacy Cosmos protobuf, старый RLP-check из node `sdk/tx`, contract-signature Safe, IPFS и загрузка contract source не входят в EVM-модуль. ERC20 multisend через произвольный Multicall не переносится: обычный approve дает разрешение самому Multicall, а не разрешение на конкретный intent. Безопасный sender-restricted deployment должен подключаться отдельным ABI.

До публичного релиза нужны выбранный module path, приемочная матрица транзакций на разрешенном тестовом кошельке, подтверждение deployment ABI и бизнес-параметров для используемой сети. Наличие исходников и успешный offline-тест не заменяет эти проверки.
