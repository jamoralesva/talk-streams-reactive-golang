package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/yomorun/yomo"
)

// NoiseData
// Los campos deben ser 'exportados' para poder ser códificados
// Se usan Field Tags para agregar metadatos útiles para la declaración. Nota: para leerlos se usa Reflection
type noiseData struct {
	Noise float32 `json:"noise"` // Noise value
	Time  int64   `json:"time"`  // Timestamp (ms)
	From  string  `json:"from"`  // Source IP
}

func main() {
	// Se crea la conexión al Zipper
	//Recordemos que el Zipper tiene un punto de conexión definido por un host y puerto.
	source := yomo.NewSource("yomo-source", yomo.WithZipperAddr("localhost:9000"))
	err := source.Connect()
	if err != nil {
		log.Printf("❌ Emit the data to YoMo-Zipper failure with err: %v", err)
		return
	}

	// El DataTag es analago al tópico donde van a parar todos los eventos generados por este 'source'
	source.SetDataTag(0x10)

	// La sentencia defer se usa para ejecutar una llamada a función justo antes de que regrese
	// la función circundante donde está presente la sentencia defer.
	defer source.Close()

	// genera datos mock y los envía al YoMo-Zipper cada 5 segundos.
	generateAndSendData(source)
}

func generateAndSendData(source yomo.Source) {
	for {
		// generate random data.
		data := noiseData{
			Noise: rand.New(rand.NewSource(time.Now().UnixNano())).Float32() * 200, //generamos datos entre 0 y 200
			Time:  time.Now().UnixNano() / int64(time.Millisecond),                 //Convertimos a Epoch en ms
			From:  "localhost",
		}

		sendingBuf, err := json.Marshal(data)
		if err != nil {
			log.Fatalln(err)
			os.Exit(-1)
		}

		// Se envían los datos vía QUIC stream.
		_, err = source.Write(sendingBuf)
		if err != nil {
			log.Printf("❌ Emit %v to YoMo-Zipper failure with err: %v", data, err)
		} else {
			log.Printf("✅ Emit %v to YoMo-Zipper", data)
		}

		//utilizado para detener la última gorutina durante al menos la duración indicada d.
		// https://golang.org/ref/spec#Constants
		time.Sleep(5000 * time.Millisecond) // También se podría 5 * time.Seconds

		// En Go, un literal numérico es una constante sin tipo.
		// Eso significa que será coaccionado silenciosamente a cualquier tipo que sea apropiado
		// para la operación donde se está utilizando. Entonces, cuando dices:
		// var x: = 5 * time.Seconds
		// Luego, el tipo se infiere a time.Duration.
	}
}
