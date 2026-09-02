Сейчас установка hook ждет переменных GIT_SAFE_KEY/GIT_SAFE_KEYFILE или legacy-переменных GIT_PRIVATE_KEY/GIT_PRIVATE_KEYFILE
Если они не указаны то ожидает флаг -keyfile. Нужно так же добавить флаг -key после которого идет содержимое приватного ключа.
Если указан флаг -key то содержимое добавляется в хуку. То есть git-safe reveal -key [ключ]
