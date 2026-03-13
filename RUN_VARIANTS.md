# Варианты запуска `git-private`

Ниже перечислены все поддерживаемые формы запуска по текущему коду `git-private`.

## Базовые команды

```bash
git-private init
git-private add <FILE...>
git-private remove <FILE...>
git-private hide [-keyfile FILE] [-clean] [FILE...]
git-private reveal [-keyfile FILE] [-force] [-clean] [FILE...]
git-private clean [-force]
git-private status
git-private help
```

Примеры:

```bash
git-private init
git-private add apikeys.json secrets/prod.env
git-private hide -keyfile ~/.ssh/id_rsa
git-private reveal -force -clean
```

## Команда `keys`

```bash
git-private keys list [-keyfile FILE]
git-private keys add [-keyfile FILE] [-id ID] [-readonly] <-pubfile FILE | "PUBLIC_KEY">
git-private keys remove [-keyfile FILE] <-id ID | ID>
git-private keys generate -keyfile FILE [-pubfile FILE]
```

Примеры:

```bash
git-private keys list -keyfile ~/.ssh/id_rsa
git-private keys add -keyfile ~/.ssh/id_rsa -pubfile ~/.ssh/id_rsa.pub
git-private keys add -keyfile ~/.ssh/id_rsa -id ci-bot -readonly "ssh-ed25519 AAAA... ci@host"
git-private keys remove -keyfile ~/.ssh/id_rsa -id ci-bot
git-private keys remove -keyfile ~/.ssh/id_rsa ci-bot
git-private keys generate -keyfile ./team.agekey -pubfile ./team.age.pub
```

## Варианты передачи приватного ключа

Для команд, где нужен приватный ключ (`hide`, `reveal`, `keys list/add/remove`), есть 3 варианта:

1. Флаг:

```bash
git-private hide -keyfile ~/.ssh/id_rsa
```

2. Переменная `GIT_PRIVATE_KEY` (содержимое ключа):

```bash
export GIT_PRIVATE_KEY="$(cat ~/.ssh/id_rsa)"
git-private reveal
```

3. Переменная `GIT_PRIVATE_KEYFILE` (путь к файлу):

```bash
export GIT_PRIVATE_KEYFILE=~/.ssh/id_rsa
git-private keys list
```

Приоритет: `-keyfile` -> `GIT_PRIVATE_KEY` -> `GIT_PRIVATE_KEYFILE`.

## Дополнительно

- `-clean` в `hide`: удаляет исходные (расшифрованные) файлы после шифрования.
- `-force` в `reveal`: перезаписывает существующие файлы.
- `-clean` в `reveal`: удаляет `*.private` после расшифровки.
- `-force` в `clean`: удаляет файлы даже если они не синхронизированы.
- В `keys add` для AGE-ключа `-id` обязателен; для SSH обычно берется comment из публичного ключа.
