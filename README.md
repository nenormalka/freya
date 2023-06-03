# freya

<p> 
freya - это ioc-контейнер, основанный на <a href="https://github.com/uber-go/dig"> di от uber</a>.
Немного теории: 
<a href="https://habr.com/ru/post/321344/"> тык </a> и
<a href="https://habr.com/ru/post/131993/"> тык </a>
<br>
Задача этого пакета упростить разработку новых сервисов и облегчить поддержку старых.

</p>

<p>

### Установка

freya подтягивается в проект, как обычный го-модуль.

```shell
go get -u github.com/nenormalka/freya
```

В ***main.go*** создаётся переменная **Module** с типом *types.Module*, которая по сути является
слайсом с конструкторами, которые требуются для элементов бизнес логики. Затем создаётся бокс
с дефолтными и требуемыми конструкторами, который отдаётся в движок для создания
приложения

```go
package main

import (
    "github.com/nenormalka/freya"
    "github.com/nenormalka/freya/config"
    example_service "github.com/nenormalka/freya/example/grpc"
    "github.com/nenormalka/freya/example/repo"
    "github.com/nenormalka/freya/example/service"
    "github.com/nenormalka/freya/types"
)

var releaseID = "release-id-example"

var Module = types.Module{
    {CreateFunc: func() config.ReleaseID {
        return config.ReleaseID(releaseID)
    }},
    {CreateFunc: repo.NewRepo},
}.
    Append(example_service.Module).
    Append(service.Module)

func main() {
	freya.
		NewEngine(Module).
		Run()
}
```

Пример реализации простого сервиса можно найти в папке <a href="https://github.com/nenormalka/freya/-/blob/master/example/main.go"> example </a>.
</p>

<p>

### Конфигурация

Переменные окружения, которые требуются для поднятия сервиса на freya:
<br>
<br>

***Основные***:
<br>
**ENV** - отвечает за определение окружения, дефолтное значение: development<br>
**LOG_LEVEL** - уровень логирования, дефолтное значение: info<br>
**APP_NAME** - название сервиса
<br>
<br>

***HTTP сервер***:
<br>
**HTTP_LISTEN_ADDR** - порт для http сервера<br>
**HTTP_KEEPALIVE_TIME** - время для keepalive параметра, дефолтное значение 30s<br>
**HTTP_KEEPALIVE_TIMEOUT** - время для keepalive таймаута, дефолтное значение 10s <br>
(freya под капотом всегда поднимает http сервер, на котором висят ручки профайлера, метрик прометеуса и health [тык](https://github.com/nenormalka/freya/-/blob/master/http/server.go#L31))
<br>
<br>

***GRPC сервер***:
<br>
**GRPC_LISTEN_ADDR** - порт для grpc сервера<br>
**GRPC_KEEPALIVE_TIME** - время для keepalive параметра, дефолтное значение 30s<br>
**GRPC_KEEPALIVE_TIMEOUT** - время для keepalive таймаута, дефолтное значение 10s<br>
**GRPC_REGISTER_REFLECTION_SERVER** - флаг, определяющий поднимать ли рефлексию, дефолтное значение true
<br>
<br>

***Elastic APM***:
<br>
**ELASTIC_APM_SERVICE_NAME** - название сервиса для APM<br>
**ELASTIC_APM_SERVER_URL** - url APM<br>
**ELASTIC_APM_ENVIRONMENT** - окружение APM<br>
(Конфиг обязательный, т.к. apm используется в интерсепторе grpc)
<br>
<br>

***Конфиг постгреса***:
<br>
**DB_DSN** - dsn для базы, если его не указать, соединение подниматься не будет<br>
**DB_MAX_OPEN_CONNECTIONS** - количество открытых коннектов, дефолтное значение 25<br>
**DB_MAX_IDLE_CONNECTIONS** - количество коннектов в режиме ожидания, дефолтное значение 25<br>
**DB_CONN_MAX_LIFETIME** - время жизни коннекта, дефолтное значение 5m
<br>
<br>

***Конфиг эластика***:
<br>
**ELASTIC_SEARCH_DSN** - dsn для эластика, если его не указать, соединение подниматься не будет<br>
**ELASTIC_SEARCH_MAX_RETRIES** - количество ретраев, дефолтное значение 5<br>
**ELASTIC_WITH_LOGGER** - флаг логгера, если true - все реквесты/респонсы к эластику будут скидываться в консоль, дефолтное значение true<br>
</p>

<p>

### Создание grpc сервера

Чтобы создать grpc сервер, требуется указать требуемую конфигурацию для окружения и экспортировать в di 
переменную типа *grpc.Definition* с тегом `group:"grpc_impl"`. *grpc.Definition* является структурой 
с сервером и описанием сервера из pb.go файла. Пример: 
<a href="https://github.com/nenormalka/freya/-/blob/master/example/grpc/server.go#L22"> тык </a>.
<br>
Алярм!!! Экспортируемая структура обязательно должна содержать встроенный тип *dig.Out*

</p>

<p>

### Получение коннектов

Для соединений к постгресу и эластику существует сущность *conns.Conns, которая содержит геттеры
для получения нужного коннекта (GetDB, GetElastic). Для этого в конструкторе репозитория надо указать получаемый параметр
conns *conns.Conns и вызвать требуемый геттер. Пример для постгреса:

```go
package repo

import (
    "fmt"

    "github.com/nenormalka/freya/conns"
	
    "go.uber.org/zap"
)

type (
    Repo struct {
        db     connectors.SQLConnector
        logger *zap.Logger
    }
)

func NewRepo(c *conns.Conns, logger *zap.Logger) (*Repo, error) {
	db, err := c.GetSQLConnByName("db")
	if err != nil {
		return nil, fmt.Errorf("create repo err: %w", err)
	}

	return &Repo{
		db:     db,
		logger: logger,
	}, nil
}
```

</p>

<p>

### Создание gracefully сервисов/воркеров

Под сервисом/воркером в данном случае понимается сущность, которая при старте приложения будет запущена
ДО запуска серверов, а при выключении приложения будет остановлена ПОСЛЕ серверов. Чтобы запустить 
такой сервис, требуется экспортировать реализацию интерфейса *types.Runnable* с тегом `group:"services"`. 
Пример: <a href="https://github.com/nenormalka/freya/-/blob/master/example/service/service.go"> тык </a>.
<br>
Алярм!!! Экспортируемая структура обязательно должна содержать встроенный тип *dig.Out*

</p>

<p>

### Общая инфа

Сервис gracefully, ожидает сигналов `syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP`. Для логов 
используется zap.
З.Ы. Запилено всё это добро было от безысходности и во славу безумия, работает на честном
слове и чёрной магии.

</p>
