# Установка и обновление

[English](INSTALL.md) | [Начало работы](GETTING_STARTED.ru.md) | [Все примеры](TRANSACTIONS.ru.md)

Нужны Go 1.26.10 или новее и интернет для загрузки модулей. Проверьте `go version`. В обычном режиме Go module proxy вход в GitHub не требуется; для клонирования исходников или прямого Git-доступа нужен Git. SDK устанавливается как библиотека внутри Go-проекта.

## Одна постоянная команда для установки и обновления

В существующем проекте, где уже есть `go.mod`, всегда запускайте одну и ту же команду:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

Она читает [постоянный latest.json](https://raw.githubusercontent.com/maxwell2010/decimal-go-sdk/main/latest.json) и одной командой `go get` устанавливает либо обновляет оба модуля: EVM и Cosmos. Номер версии в вашей команде менять не нужно. Установщик выбирает точную опубликованную версию из manifest, включая preview; зависимости загружает Go. В проекте изменяются `go.mod` и `go.sum`. Обновление происходит только при явном запуске команды, внутри SDK автоматического обновления нет.

Постоянная ссылка на эту инструкцию: https://github.com/maxwell2010/decimal-go-sdk/blob/main/docs/INSTALL.ru.md .

Для нового проекта сначала выполните:

```sh
mkdir decimal-demo
cd decimal-demo
go mod init example.com/decimal-demo
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

Посмотреть выбранную команду без изменения проекта:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -print
```

Установить только нужную часть:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -evm-only
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -cosmos-only
```

Установщик сам не создаёт проект. Без `go.mod` выполните `go mod init` или используйте `-print`. Ошибка загрузки или проверки manifest останавливает установку, fallback на произвольную версию отсутствует. Если Go сообщает, что пакет установщика ещё не найден сразу после релиза, дождитесь обновления module proxy или используйте фиксированную версию ниже.

## Фиксированная версия и откат

Для воспроизводимой установки текущего релиза:

```sh
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.4 github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.4
```

Для отката укажите прежнюю опубликованную версию обоих модулей, например `v0.1.0-preview.1`, вместо текущей. Публичные теги и архивы не перезаписываются. После изменения версии проверьте сборку своего приложения:

```sh
go list -m github.com/maxwell2010/decimal-go-sdk github.com/maxwell2010/decimal-go-sdk/cosmos
go mod verify
go test ./...
```

Импорты: `decimal "github.com/maxwell2010/decimal-go-sdk"` и `cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"`. Добавляйте только используемые импорты в Go-файлы; `go mod tidy` удаляет неиспользуемые зависимости.

## Обычный Go latest и наш канал

Можно использовать обычный `go get github.com/maxwell2010/decimal-go-sdk@latest`, но Go предпочитает стабильные версии и выбирает preview только при отсутствии стабильных. Поэтому общий канал из manifest выбирает явно опубликованный latest SDK независимо от этой политики. [Правила Go version queries](https://go.dev/ref/mod#version-queries). Команда `go run .../cmd/decimal-sdk-install@latest` выбирает доступную версию самой утилиты, а версия устанавливаемого SDK берётся из актуального manifest.

## Архивы и запуск всех примеров

В latest.json также находятся ссылки на текущий GitHub release, ZIP/TAR.GZ и SHA256SUMS. Можно скачать исходники обоих модулей и проверить SHA-256 по опубликованному файлу. Архив содержит тесты, примеры и документацию. Ссылки GitHub `releases/latest` и Go `@latest` имеют собственные правила выбора; адрес latest.json нашего канала остаётся постоянным. [Ссылки GitHub на релизы](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases).

Из каталога скачанного SDK или его клона:

```sh
go run ./examples/offline_transfer
go run ./examples/token_transfer
go run ./examples/envelopes
go run ./examples/transactions
go run ./examples/checks_signing
go -C cosmos run ./examples/messages
```

Это только локальная сборка и кодирование. Пример `read_balance` отдельно требует публичный адрес и выполняет чтение RPC. Ни один из перечисленных примеров не загружает пользовательский ключ и не отправляет транзакции. Полный список: [260 ABI builders](TRANSACTIONS.ru.md), [форматы 0/1/2 и ручные сценарии](TRANSACTION_ENVELOPES.ru.md), [43 Cosmos-сообщения](../cosmos/TRANSACTIONS.ru.md). Доступность схемы или builder не подтверждает права и исполнение в конкретной сети.
