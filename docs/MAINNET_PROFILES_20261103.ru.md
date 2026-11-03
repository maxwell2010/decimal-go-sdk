# Mainnet Token и NFTCenter: подтвержденные профили, 2026-11-03

Добавлены явно выбираемые `MainnetReserveToken20261103` и `MainnetNFTCenter20261103`: 78 методов, включая 30 write builders. Общий каталог теперь содержит 602 метода, 290 builders и 22 профиля. Старые ABI и адреса по умолчанию не заменены. Mainnet-профили разрешены только для chain75; testnet-профили сохраняют собственные guards. Офлайн-конструктор может собрать calldata без сети, но `Prepare` отклоняет вызов чужой сети до RPC.

Источник новых ABI - официальный `https://api.decimalchain.com/api/v1/contracts/compact/{implementation}`. Схемы относятся к реализации, а не к proxy shell. Это атрибуция официального API, не криптографическое доказательство всех методов или прав пользователя. Старые Go-binding ноды остаются другим поколением интерфейса.

| Профиль | Блок и привязка |
|---|---|
| MainnetReserveToken20261103 | Блок 34412425, RPC hash `0x86bdf8ee5ad681040fb83879d2e1a877b3d3ae01025eb6f2451b6bd54b6f5a2f`. Registry возвращает TokenCenter `0x9113ba675aa8f2ef0c068cee2cdabab95b6437fb`; tokens("mintcandy") возвращает proxy `0x4e8118e97586a60e5d71e45811e512546bcd52ce`. EIP1967 beacon slot указывает на TokenCenter; implementation() возвращает `0x636ea0b87d85195dedf83362d8a390780fe837dd` |
| MainnetNFTCenter20261103 | Блок 34412429, RPC hash `0x72d5d1b90b8d7be3f77a23b2e42a3327e885699e75c1fc2c1ad4feb4c255505e`. Registry возвращает proxy `0x443cc8ac24630be9257483a09c34ffa81608eace`. EIP1967 implementation slot и getImpl() возвращают `0xced55da70f42ab101b4ce10aa9e767607cef5146`; getContractCenter() возвращает проверенный registry |

Номера и исходные RPC-хеши блоков повторно проверены после чтений. Сырые запросы/ответы, код, API-ответы и SHA256 сохранены в [manifest](mainnet-deployment-sources.json), [Token evidence](deployments/mainnet-2026-11-03-token/metadata.json) и [NFTCenter evidence](deployments/mainnet-2026-11-03-nft/metadata.json). Офлайн-тесты проверяют целостность файлов, Keccak кода proxy/реализации, chain/block и цепочку slot/getter. Импортер `python -B tools/import_mainnet_deployments.py` не использует сеть; после импорта форматировать два generated Go-файла через gofmt и регенерировать примеры.

## Точные возможности

- Reserve Token: `BuildUpdateMinTotalSupply`, `BuildUpdateMaxTotalSupply` и соответствующие getters. В этом ABI нет `mint` и `setMintable`; они не добавлены по аналогии с Console.
- NFTCenter: `BuildCreateDRC721`, `BuildCreateDRC1155`, `BuildCreateDRC721Reserveless`, `BuildCreateDRC1155Reserveless`, каждый с собственным типом metadata. Все четыре метода проверены с различными значениями и точным порядком полей.
- Reserveless: TokenCenter.implementationReserveless() указывает на `0xc4b3c275749c422a2bdb750fa12875975be8fb79`. Его официальный ABI содержит mint(address,uint256), cap, mintable, withCap, но не setMintable. Код и ABI сохранены в `unbound_implementations` Token evidence. Sample token proxy не найден ограниченными проверками; deployment-профиль для него не зарегистрирован. Это не утверждение, что таких токенов в сети нет.

```go
token, err := client.MainnetReserveToken20261103(tokenProxy)
if err != nil {
    return err
}
call, err := token.BuildUpdateMinTotalSupply(minimumInRawUnits)
if err != nil {
    return err
}
fmt.Println(call.To, call.Signature) // только calldata; без подписи и отправки
```

Конструктор выбирает интерфейс и проверяет сеть, но не доказывает соответствие произвольного переданного адреса snapshot. Для использования после обновлений proxy нужно заново прочитать его привязку. Существующий `BindProxyABI(ctx, proxy, block, rawABI)` выполняет RPC-проверку и возвращает Contract с защитой deployment identity; выбирайте ABI через `ABIJSON("MainnetNFTCenter20261103")` или reserve-профиль и при необходимости оборачивайте bound.Contract соответствующей generated-структурой. Транзакции адресуются proxy, никогда implementation.

Все 602 wrappers покрыты ABI wire-тестами, все 290 write builders - [компилируемыми офлайн-примерами](../examples/README.md). Реальная write-приемка, права на mint/эмиссию, успешное создание NFT и settlement не проверялись. В рамках RPC-проверки не было подписи, approval или отправки транзакций. Mainnet snapshots новее старой матрицы октября: старые RPC-наблюдения не выдаются за проверку новых профилей в том же блоке.


Проверки после интеграции: `pwsh -NoProfile -File tools/check.ps1 -Race -Vulnerabilities` завершился с кодом 0 для обоих модулей; tests/vet/build/module verification/race прошли. Все 36 Python tooling-тестов прошли. Capture: 1170 успешных событий тестов/примеров/subtests в корневом модуле (181 верхнеуровневая запись), 190 в Cosmos (36 записей); это не количество типов транзакций. Матрица и генерация примеров проходят --check. Сканирование: ноль достижимых и импортируемых package-level уязвимостей, одна прежняя module-level запись для неиспользуемого пакета. Эти изменения включены в preview.13; результаты удаленного CI доступны на странице релиза.
