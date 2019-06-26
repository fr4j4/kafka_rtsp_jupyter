package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/adaickalavan/kafkapc"
	"gocv.io/x/gocv"
)
// https://godoc.org/github.com/Shopify/sarama
// https://github.com/Shopify/sarama/tree/master/examples
//Hooks that may be overridden for testing
var inputReader io.Reader = os.Stdin
var outputWriter io.Writer = os.Stdout

// Instantiate a producer
var producer sarama.AsyncProducer

func main() {
	//Sarama logger
	sarama.Logger = log.New(outputWriter, "[saramaLog]", log.Ltime)

	var polygons = []string{os.Getenv("POLYGONS")}
	var camera_id = os.Getenv("CAMERA_ID")

	//Create a Kafka producer
	var brokers = []string{os.Getenv("KAFKAPORT")}
	var err error
	producer, err = kafkapc.CreateKafkaProducer(brokers)
	if err != nil {
		panic("Failed to connect to Kafka. Errimg.Pixor: " + err.Error())
	}
	//Close producer to flush(i.e., push) all batched messages into Kafka queue
	defer func() { producer.Close() }()

	// Capture video
	webcam, err := gocv.OpenVideoCapture(os.Getenv("RTSPLINK"))
	if err != nil {
		panic("Error in opening webcam: " + err.Error())
	}

	// Stream images from RTSP to Kafka message queue
	frame := gocv.NewMat()
	for {
		if !webcam.Read(&frame) {
			continue
		}
		now := time.Now()
		timestamp := now.UnixNano() / 1000000
		// Type assert frame into RGBA image
		imgInterface, err := frame.ToImage()
		if err != nil {
			panic(err.Error())
		}
		img, ok := imgInterface.(*image.RGBA)
		if !ok {
			panic("Type assertion of pic (type image.Image interface) to type image.RGBA failed")
		}

		//Form the struct to be sent to Kafka message queue
		doc := Result{
			Pix:      img.Pix,
			Channels: frame.Channels(),
			Rows:     frame.Rows(),
			Cols:     frame.Cols(),
			Timestamp: timestamp,
			Fps: 30,
		}

		//Prepare message to be sent to Kafka
		docBytes, err := json.Marshal(doc)
		if err != nil {
			log.Fatal("Json marshalling error. Error:", err.Error())
		}
		msg := &sarama.ProducerMessage{
			Topic:     camera_id,
			Value:     sarama.ByteEncoder(docBytes),
			Timestamp: time.Now(),
			Polygons: polygons
		}
		//Send message into Kafka queue
		producer.Input() <- msg

		//Print time of receiving each image to show the code is running
		fmt.Fprintf(outputWriter, "---->>>> %v\n", time.Now())
	}
}

//Result represents the Kafka queue message format
type Result struct {
	Pix      []byte `json:"pix"`
	Channels int    `json:"channels"`
	Rows     int    `json:"rows"`
	Cols     int    `json:"cols"`
	CameraId  string `json:camera_id`
	Fps int `json:fps`
	Timestamp int64 `json:timestamp`
	Polygons string `json:polygons`

}
