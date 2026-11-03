# Mainnet Token/NFTCenter: проверка источников, 2026-11-03

Обновление: официальный API реализации и RPC-привязка теперь найдены для reserve Token и NFTCenter. См. [подтвержденные профили и границы](MAINNET_PROFILES_20261103.ru.md). Ниже сохранены результаты первоначальной проверки именно Go-binding.

Сверены `web/src/assetProtocol.js`, `nftProtocol.js` и `protocol/candy_mainnet_v1.json` локального снимка Console. Профиль Console сообщает chain75 и дату reviewed 2026-09-16; это заявленная provenance снимка, а не проверка его соответствия каждому живому контракту.

Текущий master официального GitHub-репозитория go-smart-node проверен через API: commit `a46cb48d7965c1a64a495cebb22e735dfe4b9da3`. Из Go-binding извлечены ABI для Token, NFTCenter и ContractCenter. Ссылки, SHA256 исходников и перечни методов записаны в [машинном отчете](CONSOLE_MAINNET_ABI_AUDIT.json).

Подтвержденные расхождения:

- Официальный Token binding не содержит `mint`, `updateMaxTotalSupply`, `updateMinTotalSupply`, `setMintable`. Он содержит getters min/max/total supply. Наличие этих write-методов в testnet-профиле или Console ABI не доказывает их mainnet-совместимость.
- Официальный NFTCenter binding содержит `createERC721` и `createERC1155`; требуемые Console DRC/Reserveless-варианты этим источником не подтверждаются. Нельзя механически заменить имя функции: selector зависит также от полного набора типов аргументов.
- Официальный ContractCenter binding сам по себе не подтверждает отсутствующие интерфейсы Token/NFTCenter. Адрес, полученный из registry, не является доказательством ABI реализации.

Новый versioned mainnet-профиль пока не добавлен. Для его принятия нужны точный официальный ABI реализации и привязка конкретного mainnet proxy/implementation к этой схеме: chain75, фиксированный блок с исходным RPC-хешем, registry getter, код proxy и реализации, механизм proxy/beacon и хеш ABI. ABI из публичного API следует маркировать как источник интерфейса, а не криптографическое доказательство всех методов. Совпадение отдельных getters или selectors не сертифицирует write-пути.

Временный `ContractWithABI` допустим как явный выбор интерфейса приложения: сохранять источник, хеш, адрес и поколение каждого ABI, точно проверять calldata офлайн и прекращать работу при неизвестной схеме. Такой контракт не наследует автоматически подтверждение deployment или сетевые ограничения встроенных versioned-профилей. Проверки chain/address/codehash нужно выполнять отдельно. Копия фиксированного Console ABI не становится официальным mainnet-профилем только из-за использования SDK.

Chain guards существующих testnet-профилей не ослаблены, глобальные адреса и ABI не заменены. Исходники production/Console, ее копия SDK, публичные релизы и latest не менялись. Сетевые транзакции, approvals, подписи и деплой не выполнялись. Следующий шаг по профилям зависит от получения недостающих авторитетных интерфейсов и read-only привязки deployment; неизвестные методы не выдумываются.
