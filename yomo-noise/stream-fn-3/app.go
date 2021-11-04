package main

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"sync"
	"time"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/pkg/logger"
)

// ThresholdAverageValue es el umbral del valor medio después de una ventana deslizante.
const ThresholdAverageValue = 13

// SlidingWindowInMS es el tiempo en milisegundos de la ventana deslizante. 10 s
const SlidingWindowInMS uint32 = 1e4

// SlidingTimeInMS es el intervalo en milisegundos del deslizamiento. 1 s
const SlidingTimeInMS uint32 = 1e3

// Calcula el promedio de cada ventana pasados de IoT data

// Una interfaz vacía puede contener valores de cualquier tipo. (Cada tipo implementa al menos cero métodos).
// Las interfaces vacías son utilizadas por código que maneja valores de tipo desconocido.
// Por ejemplo, fmt.Print toma cualquier número de argumentos de tipo interfaz {}.
var slidingAvg = func(i interface{}) error {
	// Una aserción de tipo proporciona acceso al valor concreto subyacente de un valor de interfaz.
	// Para probar si un valor de interfaz contiene un tipo específico,
	// una aserción de tipo puede devolver dos valores: el valor subyacente y un valor booleano que informa si la aserción se realizó correctamente.
	// basicamente se prueba que venga un arreglo.
	values, ok := i.([]interface{})
	if ok {
		var total float32 = 0
		for _, value := range values {
			// Convertimos a float32
			total += value.(float32)
		}
		avg := total / float32(len(values))
		logger.Printf("🧩 average value in last %d ms: %f!", SlidingWindowInMS, avg)
		if avg >= ThresholdAverageValue {
			logger.Printf("❗❗  average value in last %d ms: %f reaches the threshold %d!", SlidingWindowInMS, avg, ThresholdAverageValue)
		}
	}
	return nil
}

// También puede usar varias líneas para declarar e inicializar los valores de diferentes
// tipos usando una palabra clave var de la siguiente manera:

var (
	observe = make(chan float32, 1)
)

func main() {
	sfn := yomo.NewStreamFunction("Noise-3", yomo.WithZipperAddr("localhost:9000"))
	defer sfn.Close()

	sfn.SetObserveDataID(0x15)
	sfn.SetHandler(handler)

	err := sfn.Connect()
	if err != nil {
		logger.Errorf("[fn3] connect err=%v", err)
		os.Exit(1)
	}

	// creamos una gorutina llamando la función 'SlidingWindowWithTime':
	// le pasamos el canal de float32, el tamaño de la ventana, el intervalo de la ventana y pasamos una función que calcula el promedio.
	go SlidingWindowWithTime(observe, SlidingWindowInMS, SlidingTimeInMS, slidingAvg)

	select {}
}

func handler(data []byte) (byte, []byte) {
	v := Float32frombytes(data)
	logger.Printf("✅ [fn3] observe <- %v", v)
	// mandamos el dato recién recibido al canal.
	observe <- v

	return 0x16, nil // no more processing, return nil
}

// el tipo Handler define una función que maneja un valor de entrada generico.
type Handler func(interface{}) error

type slidingWithTimeItem struct {
	timestamp time.Time
	data      interface{}
}

// SlidingWindowWithTime almacena los datos en el tiempo de ventana deslizante especificado, los datos almacenados en el búfer se pueden procesar en la función del manejador.
// Devuelve los datos originales a Stream, no el segmento almacenado en búfer.

// Los canales se pueden pasar como parámetros de una función
// chan : bidirectional channel (Both read and write)
// chan <- : only writing to channel
// <- chan : only reading from the channel (input channel)
// *chan : channel pointer. Both read and write

// En este caso recibimos un canal de solo lectura
func SlidingWindowWithTime(observe <-chan float32, windowTimeInMS uint32, slideTimeInMS uint32, handler Handler) {
	f := func(ctx context.Context, next chan float32) {

		//Se crea un buffer, el cual es básicamente un slice de longitud 0.
		// https://stackoverflow.com/questions/25543520/declare-slice-or-make-slice#25543590
		// Cuando se crea con make, así sea de longitud 0, se asigna memoria,
		// a diferencia de crearlo con:
		// var buf []slidingWithTimeItem -> apuntará a nil.
		buf := make([]slidingWithTimeItem, 0)

		//https://dave.cheney.net/2014/03/25/the-empty-struct
		// Width: describe el número de bytes de almacenamiento que ocupa una instancia de un tipo.
		// Como el espacio de direcciones de un proceso es unidimensional, es más adecuado que el tamaño.
		// Width es una propiedad de un tipo. Como cada valor en un programa Go tiene un tipo, el ancho del valor se define por su tipo y siempre es un múltiplo de 8 bits.
		// https://play.golang.org/p/PyGYFmPmMt

		// Siempre que se declara un canal, se asigna memoria del tipo que sea.
		// Pero cuando se usa estructura vacía como tipo, no se asigna memoria y se usa solo como canal de señal única.
		stop := make(chan struct{})

		firstTimeSend := true
		mutex := sync.Mutex{}

		// Esta función anónima revisar el slice 'buf' obtiene los elementos que componente la ventana y calcula el promedio.
		checkBuffer := func() {
			mutex.Lock()
			// filtra los elementos por tiempo
			updatedBuf := make([]slidingWithTimeItem, 0)
			availableItems := make([]interface{}, 0)
			t := time.Now().Add(-time.Duration(windowTimeInMS) * time.Millisecond)
			for _, item := range buf {
				if item.timestamp.After(t) || item.timestamp.Equal(t) {
					updatedBuf = append(updatedBuf, item)
					availableItems = append(availableItems, item.data)
				}
			}
			buf = updatedBuf

			// aplica el handler
			if len(availableItems) != 0 {
				err := handler(availableItems)
				if err != nil {
					logger.Errorf("[fn3] SlidingWindowWithTime err=%v", err)
					return
				}
			}
			firstTimeSend = false
			mutex.Unlock()
		}

		// Dejamos corriendo una gorutina anónima que cuando recibe la señal de 'stop' o 'Done' revisa el buffer para obtener la ventana y promediar.
		go func() {
			// recordemos que next es canal de lectura y escritura que se recibe como parametro.
			defer close(next)

			// Un ciclo infino, recordemos que en Golang no hay ciclos while(true) o while(1)
			for {
				select {
				case <-stop:
					checkBuffer()
					return
				case <-ctx.Done():
					return

				// Recordemos la definición: func After(d Duration) <-chan Time
				// se usa para esperar a que pase el tiempo y luego entrega el tiempo real en el canal devuelto.
				case <-time.After(time.Duration(windowTimeInMS) * time.Millisecond):
					if firstTimeSend {
						checkBuffer()
					}
				case <-time.After(time.Duration(slideTimeInMS) * time.Millisecond):
					checkBuffer()
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				close(stop)
				return
			// En este punto recolectamos el dato del canal y lo agregamos al 'buf'
			case item, ok := <-observe:
				if !ok {
					close(stop)
					return
				}
				mutex.Lock()
				// buffer data
				buf = append(buf, slidingWithTimeItem{
					timestamp: time.Now(),
					data:      item,
				})
				mutex.Unlock()
				// immediately send the original item to downstream
				SendContext(ctx, item, next)
			}
		}
	}

	next := make(chan float32)

	// https://golangbyexample.com/using-context-in-golang-complete-guide/
	//Contexto:
	// Problema que resuelve:
	// Empezaste una goroutine que a su vez inicia más goroutines y así sucesivamente.
	// Suponga que la tarea que estaba haciendo ya no es necesaria.
	// Luego, cómo informar a todos goroutines hijas para que salgan con gracia y puedan liberar recursos.

	// context.Background ():
	// Devuelve un Contexto vacío que implementa la interfaz de Context
	// - No tiene valores
	// - Nunca se cancela
	// - No tiene fecha límite
	go f(context.Background(), next)
}

func SendContext(ctx context.Context, input float32, ch chan<- float32) bool {
	select {
	case <-ctx.Done(): // Context.Done tiene la mas alta prioridad
		return false
	default:
		select {
		case <-ctx.Done():
			return false
		case ch <- input:
			return true
		}
	}
}

func Float32frombytes(bytes []byte) float32 {
	bits := binary.BigEndian.Uint32(bytes)
	return math.Float32frombits(bits)
}
