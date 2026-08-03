

# go_memoize

![Estado del flujo de trabajo](https://github.com/agkloop/go_memoize/actions/workflows/ci.yml/badge.svg)

`go_memoize` es un paquete de memoización y caché para Go con un único módulo raíz público para la memoización directa de funciones, flujos de trabajo explícitos del motor de caché, almacenes en memoria, instantáneas en segundo plano, cargadores, métricas y adaptadores.

## Instalación

```sh
go get github.com/agkloop/go_memoize
```

## Elija la API Adecuada

| Necesidad | Uso | ¿Por qué? |
|---|---|---|
| Memoizar una función con argumentos comparables | `memoize.Memoize1(fn, memoize.Opts().WithTTL(ttl))` hasta `Memoize7` | API directa y ligera con opciones del motor de caché. |
| Memoizar una función que puede fallar | `memoize.Memoize1E` o `memoize.MemoizeCtx1E` | Se devuelven los errores y se almacenan en caché los valores exitosos. |
| Almacenar en caché claves comerciales explícitas | `memoize.New[K,V]` + `Cache.GetOrCompute` | Control total sobre el tipo de clave, almacén, TTL, comportamiento de datos caducados, métricas y apagado. |
| Muchas claves en memoria con límite | `memory.New[K,V](capacity)` | LRU exacto de forma predeterminada. |
| Un único valor lógico o una clave de alto tráfico | `memory.NewSingle[K,V]()` o `background.Keep` | Evita la sobrecarga innecesaria de LRU/hash. |
| Muchas claves de alto tráfico simultáneamente | `memory.NewSharded[K,V](capacity)` | Distribuye las claves primitivas compatibles en diferentes fragmentos (shards). |
| Caché entre procesos o hosts | Adaptador de Redis o un almacén compartido personalizado | Los almacenes en memoria dentro del proceso no se comparten entre procesos. |
| Una instantánea programada con lecturas instantáneas | `background.Keep` o `background.Mirror` | Se actualiza de forma independiente al tráfico de solicitudes. |

## Ejemplos Rápidos

Memoizador con TTL directo:

```go
cached, err := memoize.Memoize1(loadUser, memoize.Opts().WithTTL(time.Minute))
if err != nil {
    return err
}
user := cached(42)
```

Memoizador directo con datos caducados en caso de error:

```go
cached, err := memoize.MemoizeCtx1E(repo.LoadProfile,
    memoize.Opts().
        WithTTL(time.Minute).
        WithStaleTTL(5*time.Minute).
        KeepStaleOnError(),
)
if err != nil {
    return err
}
profile, err := cached(ctx, 42)
```

Caché explícita con claves comerciales:

```go
cache, err := memoize.New[string, User](
    memoize.Opts().
        WithStore(memory.New[string, User](10_000)).
        WithTTL(time.Minute),
)
if err != nil {
    return err
}
defer cache.Stop()

user, err := cache.GetOrCompute(ctx, "user:42", func(ctx context.Context) (User, error) {
    return repo.LoadUser(ctx, 42)
})
```

## Valores predeterminados

`memoize.New[K,V]` intencionalmente no elige un almacén ni una política de expiración por usted:

| Configuración | Predeterminado |
|---|---|
| Almacén | Ninguno. Use `Opts().WithStore(store)` a menos que utilice `Bypass()`. |
| Política de expiración | Ninguna. Elija `WithTTL`, `NoExpiration` o `Bypass`. |
| Métricas | Desactivadas con una implementación noop (sin operación). Habilítelas con `WithMetrics`. |
| Reloj | Reloj basado en ticks con un intervalo de 1 ms. Llame a `cache.Stop()` para liberarlo. |
| Tiempo de espera de actualización | 30 segundos para la actualización en segundo plano de datos caducados. Anúlelo con `WithRefreshTimeout`. |
| Fusión de fallos de la misma clave | Habilitado internamente para fallos concurrentes de `GetOrCompute` en la misma clave. |

Las funciones directas `Memoize*` utilizan las mismas opciones. Si se omite `WithStore`, crean un almacén interno sin límite `Store[uint64,V]` para claves de argumentos hash. Los memoizadores directos aún requieren `WithTTL`, `NoExpiration` o `Bypass`; no eligen silenciosamente una política de expiración.

## Arquitectura y Rendimiento

`go_memoize` utiliza un único motor de caché raíz tanto para memoizadores directos como para cachés explícitas.

- Los memoizadores directos convierten en hash los argumentos comparables a `uint64` y utilizan el mismo motor de caché que `memoize.New`.
- Los almacenes persisten envolturas crudas `memoize.Stored[V]`; el motor de caché se encarga de las decisiones sobre frescura, datos caducados y expirados.
- `GetOrCompute` fusiona fallos concurrentes de la misma clave a través de un mapa interno `singleflight`.
- Los almacenes en memoria proporcionan un comportamiento LRU exacto, límites de bytes opcionales y rutas rápidas privadas utilizadas por el motor de caché.
- La actualización de datos caducados de la caché utiliza la maquinaria de concurrencia del motor de caché; `background.Keep` y `loader.New` comparten internamente la infraestructura del bucle de actualización periódica.
- Las métricas utilizan un único método de eventos, `RecordMetric(MetricEvent)`, para mantener la instrumentación de la ruta crítica pequeña y neutra al backend.

La instantánea de referencia más reciente de este clon utilizó:

```sh
go test ./benchmarks/ -bench=. -benchmem -benchtime=1s -count=1
```

Resultados principales en esta máquina: `BenchmarkMemoryHotHit` fue `29.80 ns/op` con `0 B/op` y `0 allocs/op`; `BenchmarkSingleHotHit` fue `14.74 ns/op`; `BenchmarkGetOrComputeStampede` fue `1747 ns/op` con `0 allocs/op`; la estampida de datos caducados fue `399.8 ns/op` con `0 allocs/op`. La fragmentación no mejoró una clave única de alto tráfico (`BenchmarkMemoryHotHitParallel` `150.9 ns/op`, fragmentado `150.0 ns/op`). Consulte `docs/PERFORMANCE.md` para obtener la salida completa y su interpretación.

## Recomendaciones para Producción

- Utilice funciones directas `Memoize*` para la memoización simple de funciones dentro del proceso.
- Utilice `Cache.GetOrCompute` cuando las claves sean identificadores comerciales, cuando necesite almacenes personalizados o cuando la actualización de datos caducados sea importante.
- Utilice `background.Keep` con un almacén compartido para un actualizador de instantáneas de un solo escritor, y `background.Mirror` en los procesos de lectura para lecturas atómicas locales.
- Los almacenes personalizados implementan `memoize.Store[K,V]` y deben almacenar envolturas crudas `memoize.Stored[V]`; el motor de caché se encarga de las decisiones de frescura.
- S3 y otros almacenes de objetos pueden ser almacenes L2 duraderos personalizados detrás de `chain.New`, pero no deben ser la primera capa de caché para rutas críticas de baja latencia.
- Utilice `memory.NewSingle` o `background.Keep` para un único valor lógico como configuraciones, banderas de características o tipos de cambio.
- Utilice `memory.NewSharded` solo cuando muchas claves primitivas compatibles diferentes sean de alto tráfico simultáneamente; una clave aún se asigna a un fragmento.
- Utilice el adaptador de Redis u otro almacén compartido cuando múltiples procesos o hosts necesiten la misma caché de respaldo.
- Siempre use `defer cache.Stop()` para cachés explícitas que utilicen el reloj de tick predeterminado.
- Ejecute `go test ./... -count=1` y `go test ./... -race -count=1` antes de la publicación.

## Documentación

- `docs/GETTING_STARTED.md` - instalación y primeros ejemplos funcionales.
- `docs/CONCEPTS.md` - caché directa vs explícita, claves, almacenes, TTL/caducidad y valores predeterminados.
- `docs/RECIPES.md` - patrones de uso listos para copiar y pegar.
- `docs/PERFORMANCE.md` - arquitectura interna, optimizaciones de rendimiento y resultados de referencia.
- `docs/API.md` - referencia completa de la API pública.
- `docs/PRODUCTION.md` - selección de almacenes para producción, observabilidad y comportamiento ante fallos.
- `examples/direct_stale_profile_cache` - ejemplo de memoizador directo con datos caducados en caso de error.
- `examples/http_user_cache` - caché explícita con claves comerciales.
- `adapters/redis/examples/hybrid_profile_cache` - memoizador directo con L1 en memoria y L2 en Redis.
