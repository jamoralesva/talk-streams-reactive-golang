# YoMo Noise

Basado en el ejemplo: https://github.com/yomorun/yomo/tree/master/example/multi-stream-fn

## Estructura

+ `source`: Genera datos ficticios de un detector de ruido. [yomo.run/source](https://docs.yomo.run/source)
+ `stream-fn-1`: Calcula el valor real del ruido en tiempo real. [yomo.run/stream-function](https://docs.yomo.run/stream-function)
+ `stream-fn-2`: Imprime un mensaje de advertencia cuando se supera cierto umbral. [yomo.run/stream-function](https://docs.yomo.run/stream-function)
+ `stream-fn-3`: El flujo sin procesar es inmutable, stream-fn-3 aún puede observar los datos sin procesar y calcular el valor promedio en una ventana deslizante. [yomo.run/stream-function](https://docs.yomo.run/stream-function)
+ `zipper`: Orchestrate a workflow that receives the data from `source`, stream computing in `stream-fn` [yomo.run/zipper](https://docs.yomo.run/zipper)

## Cómo correr este ejemplo

### 1. Instalar el cliente

Ir a [YoMo Getting Started](https://github.com/yomorun/yomo#1-install-cli) para mas detalles.

### 2. Ejecute YoMo-Zipper

```
yomo serve -c ./zipper/workflow.yml
```

### 3. Ejecute las funciones

```
go run ./stream-fn-1/app.go -n Noise-1
```

```
go run ./stream-fn-2/app.go -n Noise-2
```

```
go run ./stream-fn-3/app.go -n Noise-3
```

### 4. Ejecute la fuente de datos (_source_)

```
go run ./source/app.go 
```
