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

## Установка

freya подтягивается в проект, как обычный го-модуль.

```shell
go get -u github.com/nenormalka/freya
```

В ***main.go*** создаётся переменная **Module** с типом *types.Module*, которая по сути является
слайсом с конструкторами, которые требуются для элементов бизнес логики. Затем создаётся движок
с дефолтными и требуемыми конструкторами, который и запускается.

```go
package main

import (
	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/types"

	"freya/example"
	exampleconfig "freya/example/config"
	grpc "freya/example/grpc"
	"freya/example/http"
	"freya/example/repo"
	"freya/example/service"
)

var releaseID = "release-id-example"

var Module = types.Module{
	{CreateFunc: func() (*types.AppInfo, error) {
		return types.GetAppInfo(example.ModInfo, releaseID, "")
	}},
	{CreateFunc: repo.NewRepo},
}.
	Append(exampleconfig.Module).
	Append(grpc.Module).
	Append(service.Module).
	Append(http.Module)

func main() {
	freya.
		NewEngine(Module).
		Run()
}
```

В первой CreateFunc создаётся экземпляр типа types.AppInfo, в которой хранится информация о приложении. Первый
аргумент - это embed переменная,
внутри которой хранится go.mod (можно глянуть в [gomod.go](example%2Fgomod.go)), вторая переменная - это айди релиза,
который проставляется в бинарнике во время сборки, а третья - это маска для прото пакета (по дефолту go-gen-proto).
Если нет желания возиться, можно просто первым аргументом передать nil, а третьим пустую строку. Эти данные нужны лишь
для
метрики прометеуса о приложении.

Пример реализации простого сервиса можно найти в папке
<a href="https://github.com/nenormalka/freya/-/blob/master/example/main.go"> example </a>.
</p>

<p>

## Структура проекта

### [apm](apm)

В проекте используется [apm эластика](https://pkg.go.dev/go.elastic.co/apm). Он собирает данные по запросам grpc, бд и
эластика.
Посмотреть можно в кибане проекта. Требуемые переменные конфига:

**ELASTIC_APM_SERVICE_NAME** - название сервиса для APM<br>
**ELASTIC_APM_SERVER_URL** - url APM<br>
**ELASTIC_APM_ENVIRONMENT** - окружение APM<br>

### [config](config)

Здесь происходит сборка конфига для фреи (не самого вашего приложения!). Задать конфигруацию можно 3 способами:

1) env
2) yaml
3) Кастомный способ. Для этого требуется экспортировать экземпляр типа *config.Configure*
   (*Configure(cfg *config.Config) error*) с тегом `group:"configurators"`. Таким способом можно, допустим, из консула
   достать все нужные данные и установить их в переменную конфига. [пример](example%2Fconfig%2Fconfig.go)

Порядок задания конфигурации такой env->yaml->custom, т.е. данные из custom перезапишут все предыдущие значения.

### [conns](conns)

В этом пакете представлены различные соединения. Экспортируется экземпляр типа Conns, через
методы которого можно получить требуемый коннект:

```go

func GetSQLConnByName(nameConn string) (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error) по названию
соединения возвращает sqlx коннект к постгре

func GetPGXConnByName(nameConn string) (connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx], error) по названию
соединения возвращает pgx коннект к постгре

func GetGoQuConn(nameConn string) (connectors.DBConnector[*goqu.Database, *goqu.TxDatabase], error) по названию
соединения возвращает goqu коннект к постгре (простите меня за это)

func GetKafka() (*kafka.Kafka, error) возвращает экземпляр для работы с кафкой

func GetElasticConn() (*elastic.ElasticConn, error) возращает коннект к эластику

func GetCouchbase() (*couchbase.Couchbase, error) возращает коннект к коучбейзу

func GetConsul() (*consul.Consul, error) возращает коннект к консулу
```

Остальные методы депрекейтнуты, и категорически не советую ими пользоваться.

Далее идёт описания коннектов:

### [connectors](conns%2Fconnectors)

Тут описываются интерфейсы соединений. Все коннекты реализуют интерфейсы:

```go
CallContextConnector[T ConnectDB] interface {
   CallContext(
   ctx context.Context,
   queryName string,
   callFunc func (ctx context.Context, db T) error,
) error
}

CallTransactionConnector[M ConnectTx] interface {
   CallTransaction(
   ctx context.Context,
   txName string,
   callFunc func (ctx context.Context, tx M) error,
) error
}

DBConnector[T ConnectDB, M ConnectTx] interface {
   CallContextConnector[T]
   CallTransactionConnector[M]
}
```

Тут же есть папочка [mocks](conns%2Fconnectors%2Fmocks) в которой есть мок для sqlx соединения.

### [consul](conns%2Fconsul)

Соединение с консулом. Требуемые переменные конфига:

**CONSUL_ADDRESS** - адрес консула <br>
**CONSUL_SCHEME** - схема. По дефолту http <br>
**CONSUL_TOKEN** - токен, может быть пустым <br>
**CONSUL_INSECURE_SKIP_VERIFY** -скип верификации, по дефолту true <br>
**CONSUL_SESSION_TTL** -время сессии, по дефолту 30 секунд <br>
**CONSUL_LEADER_TTL** -время, через которое проверяется лидерство, по дефолту 20 секунд <br>

Абстракция над консулом предоставляет несколько интерфейсов:

1) KV - работа с ключами-значениями. Реализует интерфейс DBConnector. Нужен для работы с ключами. Методы:

```go

func (kv *KV) CallContext(
	ctx context.Context,
	queryName string,
	callFunc func(ctx context.Context, db *api.KV) error,
) error

func (kv *KV) CallTransaction(
   ctx context.Context,
   txName string,
   callFunc func(ctx context.Context, tx *api.Txn) error,
) error
```

2) Session - нужен для работы с сессиями. Методы:

```go

Session interface {
   Create(ctx context.Context) (string, error)
   Destroy(ctx context.Context) error
   Renew(ctx context.Context) <-chan error
   SessionID() string
   SessionKey() string
}
```

3) Locker - нужен для работы с локами. Методы:

```go

Locker interface {
   Acquire(ctx context.Context, key, sessionID string) (bool, error)
   Release(ctx context.Context, key, sessionID string) (bool, error)
   KeyOwner(ctx context.Context, key string) (string, error)
}
```

4) Watcher - позволяет отслеживать изменения ключей. Методы:

```go

Watcher interface {
   Start(ctx context.Context) error
   Stop(ctx context.Context) error
   WatchKeys(keys watcher.WatchKeys) error
   WatchPrefixKeys(keys watcher.WatchPrefixKey) error
}
```

5) Leader - позволяет выбирать лидера. При старте создаёт сессию и с её помощью вещает лок. Методы:

```go

Leader interface {
   Start(ctx context.Context) error
   Stop(ctx context.Context) error
   IsLeader() bool
}
```
6) ServiceDiscovery - позволяет получать информацию о сервисах, регистрировать и разрегистрировать сервис. Методы:

```go

ServiceDiscovery interface {
   ServiceInfo(ctx context.Context, serviceName string, tags []string) ([]*api.ServiceEntry, error)
   ServiceList(ctx context.Context) (map[string][]string, error)
   ServiceRegister(ctx context.Context, reg *api.AgentServiceRegistration) error
   ServiceDeregister(ctx context.Context, serviceID string) error
}
```

### [couchbase](conns%2Fcouchbase)

Соединение с коучембейзом. Реализует интерфейс *DBConnector*. Требуемые переменные конфига:

**COUCHBASE_BUCKET** - название бакета <br>
**COUCHBASE_DSN** - дсн к коучу <br>
**COUCHBASE_USER** - пользователь <br>
**COUCHBASE_PWD** - пароль <br>
**COUCHBASE_ENABLE_DEBUG** - флаг включает логирование. Алярм! эта штука срёт, как не в себя,
так что включать на свой страх и риск. Использует логгер zap. <br>

Чтобы получить возможность работать с коллекциями, надо дёрнуть метод:

```go
func GetCollection(bucketName, collectionName string) (connectors.DBConnector[*gocb.Collection, *txtype.CollectionTx], error)
```

Пример можно посмотреть [здесь](example%2Fservice%2Fservice.go).

Так уже у этого пакета есть [couchlock](conns%2Fcouchbase%2Fcouchlock) (я не силён в придумывании названий, соррян).
Эта штуковина нужна, чтобы брать лок на каком-то инстансе, для выполнения какого-то
действия только один раз. При создании лока можно указать:

1) Количество ретраев
2) TTL лока
3) И вместо дефолтного логгера подсунуть zap

### [elastic](conns%2Felastic)

Тут находится коннект к эластику. Ничего криминального в нём нет. Требуемые переменные конфига:

**ELASTIC_SEARCH_DSN** - dsn для эластика, если его не указать, соединение подниматься не будет<br>
**ELASTIC_SEARCH_MAX_RETRIES** - количество ретраев, дефолтное значение 5<br>
**ELASTIC_WITH_LOGGER** - флаг логгера, если true - все реквесты/респонсы к эластику будут скидываться в консоль,
дефолтное значение false.
Опять таки, алярм. Всё это лучше использовать только в дев окружении. <br>

Чтобы получить коннект, надо дёрнуть метод:

```go
func GetElasticConn() (*elastic.ElasticConn, error)
```

### [kafka](conns%2Fkafka)

Абстракция над кафкой. Требуемые переменные окружения

**KAFKA_ADDRESSES** - адреса кафки <br>
**KAFKA_ENABLE_DEBUG** - включает логирование. По дефолту - false <br>
**KAFKA_SKIP_ERRORS** - позволяет указать топики, при обработке сообщений из которых, ошибки будут скипаться <br>

В данный момент предоставляет только два интерфейса:

```go
ConsumerGroup interface {
AddHandler(topic common.Topic, hm common.MessageHandler)
   Consume() error
   Close() error
   PauseAll()
   ResumeAll()
}

SyncProducer interface {
   Send(topic string, message []byte, opts ...syncproducer.SendOptions) error
   Close() error
}
```

1) [consumergroup](conns%2Fkafka%2Fconsumergroup) как не странно предоставляет консьюмер группу.
   Пример можно посмотреть [здесь](example%2Fservice%2Fservice.go). Суть в том, что создаётся группа,
   добавляются хендлеры к топикам (тип хендлера можно посмотреть тут [common.go](conns%2Fkafka%2Fcommon%2Fcommon.go)) и
   запускается Consume. Если хочется, чтобы хендлер работал с конкретным типом сообщения (а не с байтиками),
   то можно использовать вот этот метод отсюда [kafka.go](conns%2Fkafka%2Fkafka.go):

```go
func AddTypedHandler[T any](
   cg ConsumerGroup,
   topic common.Topic,
   f common.MessageHandlerTyped[T],
) error
```

Чтобы приостановить/продолжить чтение, есть методы PauseAll и ResumeAll соответственно.
При завершении приложения требуется дёрнуть метод Close.

2) [syncproducer](conns%2Fkafka%2Fsyncproducer) тут всё тоже соответствует названию. Предоставляется всего
   два метода Send и Close. Метод Send помимо основных параметров, принимает функциональные опции,
   с помощью которых можно указать партицию, заголовки и метаданные сообщения. При завершении работы
   приложения желательно дёрнуть метод Close. Если лень маршалить объекты, то можно дёргать функцию:

```go
func TypedSend[T any](
	sp SyncProducer,
	topic string,
	message T,
	opts ...syncproducer.SendOptions,
) error 
```

Она будет превращать в набор байтиков за вас. Пример так же можно глянуть [тут](example%2Fservice%2Fservice.go).

Чтобы получить обёртку, надо дёрнуть метод:

```go
func GetKafka() (*kafka.Kafka, error)
```

### [postgres](conns%2Fpostgres)

Предоставляет ТРИ вида соединений к постгре.

1) sqlx

```go
func GetSQLConnByName(nameConn string) (connectors.DBConnector[*sqlx.DB, *sqlx.Tx], error)
```

2) pgx

```go
func GetPGXConnByName(nameConn string) (connectors.DBConnector[dbtypes.PgxConn, dbtypes.PgxTx], error)
```

3) goqu

```go
func GetGoQuConn(nameConn string) (connectors.DBConnector[*goqu.Database, *goqu.TxDatabase], error)
```

Коннекты на любой вкус и цвет. Все реализуют интерфейс DBConnector.

Переменные окружения:

**DB_DSN** - dsn для базы, если его не указать, соединение подниматься не будет.
Тут есть некоторого рода костыль. Т.к. основным способом получения конфигурации является env, то
пришлось пожонглировать, чтобы можно было работать с несколькими коннектами к разным базам
одновременно. Поэтому, если указана переменная DB_DSN, то она считается master соединением. Если
требуется указать несколько разных коннектов, то переменные должны выглядеть так:
DB_DSN_CONNECT1,DB_DSN_CONNECT2,DB_DSN_CONNECT3 и т.д., тогда они будут доступны через вызов (к примеру)
GetSQLConnByName с передачей названия нужного коннекта (connect1, connect2, connect3 и т.д.) <br>
**DB_MAX_OPEN_CONNECTIONS** - количество открытых коннектов, дефолтное значение 25 <br>
**DB_MAX_IDLE_CONNECTIONS** - количество коннектов в режиме ожидания, дефолтное значение 25 <br>
**DB_CONN_MAX_LIFETIME** - время жизни коннекта, дефолтное значение 5m <br>
**DB_TYPE** - может иметь значения pgx или sqlx (дефолтное).

Пример можно подсмотреть [тут](example%2Frepo%2Frepo.go).

### [grpc](grpc)

Тут происходит сборка grpc сервера. Переменные окружения:

**GRPC_LISTEN_ADDR** - порт для grpc сервера<br>
**GRPC_KEEPALIVE_TIME** - время для keepalive параметра, дефолтное значение 30s<br>
**GRPC_KEEPALIVE_TIMEOUT** - время для keepalive таймаута, дефолтное значение 10s<br>
**GRPC_REGISTER_REFLECTION_SERVER** - флаг, определяющий поднимать ли рефлексию, дефолтное значение true <br>
**ENABLE_SERVER_METRICS** - флаг, определяющий включать ли сбор метрик по запросам, дефолтное значение true <br>

Теперь разберём, что за чертовщина тут происходит. Фрея позволяет указать любое количество сервисов
grpc, которые будут висеть на одном сервере. Чтобы это сделать, требуется экспортировать экземпляр
типа *grpc.Definition* с тегом `group:"grpc_impl"`. Этот типа является структурой:

```go
Definition struct {
Description    *grpc.ServiceDesc
Implementation any
}
```

где Description - это сгенеринное из proto описание вашего сервиса (находится в файле
{service_name}_grpc.pb.go), а Implementation это реализация вашего сервера, удовлетворяющая
описанию из Description. Всё довольно просто [тыц](example%2Fgrpc%2Fserver.go). Всегда, даже если нет других,
поднимается сервер Health.

Полный список интерсепторов можно глянуть [тут](grpc%2Finterceptors.go). Чтобы добавить кастомный
интерсептер, требуется экспортировать тип *[]grpc.UnaryServerInterceptor* с тегом
`group:"grpc_unary_interceptor"`. Подглядеть можно [здесь](example%2Fgrpc%2Fintercepters.go)

Так же у сервера есть опция (пока только одна). Устанавливающая экстеншен в виде сенситивной информации.
[Тут](example%2Fgrpc%2Fdig.go) можно посмотреть, как должна выглядеть экспортируемая структура для сервера.
Алярм!!! Экспортируемая структура обязательно должна содержать встроенный тип *dig.Out*

### [http](http)

Здесь создаётся http сервер. Из коробки на нём висят ручки метрик, профайлера и health.
Если требуется добавить что-то ещё, то требуется экспортировать структуру с тегом `group:"custom_http_servers"`,
которая будет реализовывать интерфейс *http.CustomServer*. Как [тут](example%2Fhttp%2Fdig.go).

### [logger](logger)

Логгер он и в Африке логгер. Тут используется zap. Требуются такие переменные окружения:

**LOG_LEVEL** - уровень логирования, дефолтное значение: info<br>
**APP_NAME** - название сервиса

### [metadata](metadata)

Пакет предназначен для работы с метаданными в grpc реквестах. В какие-то конкретные подробности
вдаваться не имеет смысла, т.к. проще посмотреть самостоятельно. Если вкратце, функционал позволяет
получать конкретные ключи из контекста запроса, сравнивать версии и типы приложения, приславшего запрос,
а так же проверять наличие включённых фичетогглов.

### [sentry](sentry)

Экспортирует сентри. Нужные переменные конфига:

**APP_NAME** - название сервиса <br>
**SENTRY_DSN** - дсн-ка сентри <br>
**ENV** - окружение, по дефолту development <br>

### [types](types)

Филиал хаоса. Тут сброшено в кучу всякое разное и растащить по нормальным папкам уже не видится возможным,
т.к. сломается обратная совместимость. Ну да ладно, поехали.

1) [errors](types%2Ferrors) Пакет позволяет создавать кастомные ошибки, которые в интерсепторе сервера
   преобразуются в определённый вид, позволяющий на стороне отправителя запроса, разобрать детали ошибки.
   Можно посмотреть [здесь](example%2Fgrpc%2Fserver.go) в методе GetErr.
2) [appinfo.go](types%2Fappinfo.go) Это по большей части внутренняя переменная, которая хранит информацию
   о запущенном приложении. Используется в метриках прометеуса.
3) [metrics.go](types%2Fmetrics.go) Тут хранятся все метрики фреи. Когда вы делаете запрос в бд, эластик,
   коуч и т.д., увеличивается определённая метрика. Вот их список:
    1) DBMetrics - гистограмная метрика бд, разбитая по названиям запросов
    2) DBErrorMetrics - каунтер ошибок к бд, разбитый по названиям запросов
    3) CouchbaseMetrics - гистограмная метрика коуча, разбитая по названиям запросов
    4) HTTPMetrics - гистограмная метрика запросов по http, разбитая по названиям. Нужна для
       сбора метрик кастомных запросов по хттп. Используется через WithHTTPMetrics
    5) ElasticMetrics - гистограмная метрика эластика, разбитая по названиям запросов
    6) GRPCPanicMetrics - каунтер паник сервиса при запросах grpc. Работает через интерсептер
    7) KafkaConsumerGroupMetrics - гистограмная метрика консьюмер группы, разбитая по названию консьюмер группы
       и топику
    8) KafkaSyncProducerMetrics - гистограмная метрика синк продюсера, разбитая по названию топику
    9) GaugeAppState - информация о сервисе (версия приложения, версия go, версия фреи, версия пакета прото,
       время запуска инстанса)
    10) ServerGRPCMetrics - метрика сервера grpc
4) [runnable.go](types%2Frunnable.go) Основной интерфейс сервисов и серверов приложения на фреи.
   Имеет вид:

```go
Runnable interface {
Start(ctx context.Context) error
Stop(ctx context.Context) error
}
```

Немножко душной информации. Сервис и сервер в контексте фреи - это две разных сущности,
реализующих интерфейс *Runnable*. Конкретно про каждый отпишу, когда до них дойдём, но в целом
для понимания хватает того, что сервисы запускаются ДО серверов и вырубаются ПОСЛЕ. Т.е., допустим
у вас есть какой-то кэш, который должен:

* перед запуском инстанса прогреться, чтобы, как только запустится сервер, отдавать данные на
  запросы уже из кэша
* актуализировать себя в фоне всё время работы приложения
* после сигнала о выключении инстанса, сбросить кэш в какое-нибудь хранилище

Вот это идеальный кандидат на становление сервисом.

5) [server.go](types%2Fserver.go) Как следует из названия, это сущность сервера. Из коробки их два:

* http
* grpc

Если зачем-то потребуется свой, то достаточно экспортировать структуру с тегом `group:"servers"`,
которая реализует интерфейс *types.Runnable*. Можно глянуть на примере [grpc](grpc%2Fdig.go) сервера.

6) [service.go](types%2Fservice.go) Это сущность сервиса. Ничего страшного в ней нет. Если требуется
   создать свой сервис, надо экспортировать структуру с тегом `group:"services"`, которая реализует
   интерфейс *types.Runnable*. Пример [тут](example%2Fservice%2Fdig.go).

7) [types.go](types%2Ftypes.go) Здесь находятся основные типы: Provider (конструктор какого-то нашего функционала) и
   Module (слайс Provider-ов)

### [app.go](app.go)

Это и есть наше приложение. Оно знает о всех зарегистрированных сервисах и серверах. Есть единственный
метод Run, который и запускает сначала сервисы, потому сервера и ожидает сигналов в контексте. При
получении сигнала на выключение, сначала стопает сервера, потом сервисы.

### [engine.go](engine.go)

Сердце тьмы и сосредоточие хаоса. Вся магия творится именно тут. Тут находятся дефолтные модули
фреи. Здесь происходит сбор всех сервисов и серверов. При создании движка, создаётся di и
провайдятся все дефолтные модули фреи + модули приложения. При вызове метода Run происходит инвок
мейновой функции. Во время инвока di проверяет все зависимости в конструкторах и затем запускает mainFunc
движка, в которой вызывается метод Run из [app.go](app.go). Когда приложение останавливается, срабатывает
defer в mainFunc. В нём происходит закрытие всех соединений, синхронизация логгера и сброс апм
и сентри.

### [mockengine.go](mockengine.go)

Это мок движка. Он нужен для тестов. В нём происходит создание di и провайдятся все дефолтные модули фреи, если не
указано обратное. Имеет три метода:

1) Run - запускает какую-то кастомную вашу функцию в методе Invoke di
2) RunTest - запускает тестовую функцию, которая принимает *testing.T, название и функцию для инвока
3) RunBenchmark - запускает бенчмарк функцию, которая принимает *testing.B, название и функцию для инвока

Пример можно глянуть [тут](example%2Fservice%2Fservice_test.go)
</p>

<p>

### Общая инфа

Сервис gracefully, ожидает сигналов `syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP`. <br>
З.Ы. Запилено всё это добро было от безысходности и во славу безумия, работает на честном
слове и чёрной магии.<br>
З.Ы.Ы. Не надо писать свой код точь-в-точь, как это сделано в [example](example). Пример я сделал
только для того, чтобы запускать приложение и проверять, что оно работает так, как задумывалось.<br>

</p>
