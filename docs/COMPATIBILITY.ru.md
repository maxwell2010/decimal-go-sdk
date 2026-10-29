# Матрица совместимости без отправки транзакций

Это снимок наблюдений, а не процент готовности. Наличие кода, ответ getter или успешный eth_call не доказывают ABI целиком, полномочия пользователя, приём подписанной транзакции и settlement. Live-инспекторы не загружали ключи, не подписывали и не отправляли публичные транзакции. Отдельные локальные тесты используют синтетические векторы подписи.

[JSON matrix](compatibility/matrix.json) | [Deployments](DEPLOYMENT_COMPATIBILITY.md) | [Simulations](SIMULATION_COMPATIBILITY.md) | [Checks history](CHECKS_HISTORY.md)

524 ABI methods / 260 builders / 264 readers / 20 profiles. 93 reference JS operations. Локальный тест ABI: `TestAllGeneratedABIWrappers`, результат и fingerprint исходников: [local-validation.json](compatibility/local-validation.json). [Validation](VALIDATION.md).

| Network | Block | Code addresses | Getter responses / failures | Successful limited eth_call | Checks events in bounded window |
|---|---|---|---|---|---|
| testnet | 22477428 | 14 | 26 / 6 | 2 | 6 |
| mainnet | 34336444 | 11 | 19 / 1 | 1 | 0 |

**Mainnet Checks требует отдельной проверки deployment:** настроенный адрес имеет runtime testnet-implementation, нулевые slots/getImpl/owner. Это не подтверждённый initialized proxy. Адреса и конфигурация не изменялись.

Успешные симуляции ограничены self-transfer: testnet reserve/reserveless token по 1 raw unit и mainnet DEL 1 wei. Они не проверяют внешнего получателя, burn, allowance, permit, права или bridge. Testnet native self-transfer не проверен: в выбранных 32 блоках не найден подходящий EOA. Контроль над публичными From-адресами не заявлен.

Получены 6 testnet CreatedCheck и один отдельно проверенный receipt. Ноль mainnet-событий относится только к выбранному окну 2000 блоков. Историческое событие не доказывает текущую погашаемость. Время снимка истории отличается от снимка deployments; диапазоны и RPC-хеши сохранены раздельно.

| Profile | Builders / readers | Testnet targets / getter responses / simulated signatures | Mainnet targets / getter responses / simulated signatures |
|---|---|---|---|
| Checks | 7 / 3 | 1 / 3 / 0 | 1 / 3 / 0 |
| ContractCenter | 10 / 9 | 1 / 2 / 0 | 1 / 2 / 0 |
| Delegation | 34 / 18 | 1 / 2 / 0 | 1 / 2 / 0 |
| DelegationNFT | 23 / 14 | 1 / 0 / 0 | 1 / 2 / 0 |
| ERC1155 | 14 / 20 | 0 / 0 / 0 | 0 / 0 / 0 |
| ERC721 | 16 / 17 | 0 / 0 / 0 | 0 / 0 / 0 |
| MasterValidator | 18 / 8 | 1 / 2 / 0 | 1 / 2 / 0 |
| Multicall | 1 / 0 | 1 / 0 / 0 | 1 / 0 / 0 |
| NFTCenter | 8 / 8 | 1 / 1 / 0 | 1 / 1 / 0 |
| TestnetContractCenter20261029 | 11 / 13 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetMasterValidator20261029 | 20 / 9 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetNFTCenter20261029 | 15 / 14 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetReserveToken20261029 | 15 / 34 | 1 / 4 / 1 | 0 / 0 / 0 |
| TestnetReservelessToken20261029 | 10 / 17 | 1 / 5 / 1 | 0 / 0 / 0 |
| TestnetSafe20261029 | 15 / 18 | 1 / 2 / 0 | 0 / 0 / 0 |
| TestnetSafeFactory20261029 | 3 / 2 | 1 / 0 / 0 | 0 / 0 / 0 |
| TestnetSafeMultiSend20261029 | 1 / 0 | 1 / 0 / 0 | 0 / 0 / 0 |
| TestnetTokenCenter20261029 | 13 / 14 | 1 / 4 / 0 | 0 / 0 / 0 |
| Token | 13 / 32 | 2 / 4 / 0 | 0 / 0 / 0 |
| TokenCenter | 13 / 14 | 1 / 4 / 0 | 1 / 4 / 0 |

Targets означает наблюдённые адреса, а не доказанную привязку всего ABI. Versioned testnet-профили не переносятся на mainnet. Отсутствие выборки ERC721/1155 instances не заменяется чтением NFTCenter. Ответы getter сохранены как raw RPC bytes; нулевой owner не объявляется инициализацией. Methods и все 93 операции с независимыми статусами находятся в JSON.

Осталось: авторитетные ABI недостающих поколений; state/proof/role inputs для других симуляций; отдельно разрешённая тестнет-приёмка. Для raw идентичности блока используйте Client.BlockIdentity, а не types.Header.Hash(). Before/after equality не является криптографическим state proof или гарантией финальности.


В низкоуровневых getter-пробах инспектора поле From не задано. Такие результаты не подтверждают глобальность nonce или состояние конкретного создателя. Для caller-зависимых чтений используйте явно выбранный адрес и ReadFromAt после проверки ABI. Граница не меняется от наличия уже созданных чеков: без разрешённого proof погашение не симулируется.
