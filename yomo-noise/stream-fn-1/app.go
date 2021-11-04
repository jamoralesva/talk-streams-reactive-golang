package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"time"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/pkg/logger"
)

// NoiseData
// Los campos deben ser 'exportados' para poder ser códificados
// Se usan Field Tags para agregar metadatos útiles para la declaración. Nota: para leerlos se usa Reflection
type noiseData struct {
	Noise float32 `json:"noise"` // Noise value
	Time  int64   `json:"time"`  // Timestamp (ms)
	From  string  `json:"from"`  // Source IP
}

// main observara datos con SeqID=0x10, los transforma y pone el valor de Noise en SeqID=0x14.
func main() {
	//Recordemos que el Zipper tiene un punto de conexión definido por un host y puerto.
	sfn := yomo.NewStreamFunction("Noise-1", yomo.WithZipperAddr("localhost:9000"))

	// La sentencia defer se usa para ejecutar una llamada a función justo antes de que regrese
	// la función circundante donde está presente la sentencia defer.
	defer sfn.Close()

	sfn.SetObserveDataID(0x10)
	sfn.SetHandler(handler)

	err := sfn.Connect()
	if err != nil {
		logger.Errorf("[fn1] connect err=%v", err)
		os.Exit(1)
	}

	//Cuando usamos sentencias select vacías dentro de un programa,
	//la sentencia select se bloquea para siempre ya que no hay ninguna rutina disponible para proporcionar ningún dato.
	// Entonces, simplemente bloquea la goroutine actual.

	// En CSP-speak, la selección vacía es como STOP.

	// Es idiomatico de Go, Dado que la gorutina principal no ha retornado, esto deja el proceso
	// y se puede analizar el comportamiento de las otras gorutinas.
	select {}
}

// Función que maneja los eventos, devuelve el Id del downstream y un evento.
func handler(data []byte) (byte, []byte) {
	var mold noiseData
	err := json.Unmarshal(data, &mold)
	if err != nil {
		logger.Errorf("[fn1] json.Unmarshal err=%v", err)
		return 0x0, nil
	}
	mold.Noise = mold.Noise / 10

	// Se Imprimime y Extrae el valor del
	result, err := printExtract(context.Background(), &mold)
	if err != nil {
		logger.Errorf("[fn1] to downstream err=%v", err)
		return 0x0, nil
	}

	// transferimos el resultado al downstream 0x14
	return 0x14, float32ToByte(result)
}

// Imprime todos los valores y devuelva el valor de ruido.
// Importante: Golang tiene punteros, es decir *T representa la dirección del valor T. Golang no tiene aritmetica de punteros. Su valor cero es nil.
// Al igual que C, el operador & genera punteros a un valor y * representa el valor subyacente al puntero -> "deferencing" o "indirecting".

// Similar a como se hace en JS se pueden crear funciones anonimas que pueden asigarse a variables.
var printExtract = func(_ context.Context, value *noiseData) (float32, error) {
	rightNow := time.Now().UnixNano() / int64(time.Millisecond)
	logger.Printf("✅ [%s] %d > value: %f ⚡️=%dms", value.From, value.Time, value.Noise, rightNow-value.Time)

	// A pesar de que value es un puntero a una estructura, se puede acceder a los campos de esta con el '.'
	return value.Noise, nil
}

func float32ToByte(f float32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, f)
	return buf.Bytes()
}
