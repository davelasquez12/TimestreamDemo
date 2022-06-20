package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/timestreamwrite"
	"github.com/google/uuid"

	"golang.org/x/net/http2"
)

func main() {
	tr := &http.Transport{
		ResponseHeaderTimeout: 20 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			KeepAlive: 30 * time.Second,
			DualStack: true,
			Timeout:   30 * time.Second,
		}).DialContext,
		MaxIdleConns:          5000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// So client makes HTTP/2 requests
	http2.ConfigureTransport(tr)

	sess, err := session.NewSession(&aws.Config{
		Region:     aws.String("us-east-2"),
		MaxRetries: aws.Int(10),
		HTTPClient: &http.Client{
			Transport: tr,
		},
		Credentials: credentials.NewStaticCredentials(AccessKeyID, SecretKey, ""), //todo: substitute aws credentials here
	})

	if err != nil {
		log.Fatal(err)
	}

	writeSvc := timestreamwrite.New(sess)

	n := 0
	for n < 15 {
		order := Order{
			Id:               uuid.NewString(),
			LocationNum:      strconv.Itoa(n % 3), //3 unique locations
			CheckInTimestamp: time.Now(),
			OrderSubTotal:    float64(n) + 0.32, //simulate subtotal prices
		}

		fmt.Println(order)
		err = insertOrder(writeSvc, order)
		if err != nil {
			log.Fatal(err)
		}
		n++
	}

	if err != nil {
		log.Fatal(err)
	}

}

func insertOrder(writer *timestreamwrite.TimestreamWrite, order Order) error {
	orderTimestampMilli := strconv.FormatInt(order.CheckInTimestamp.UnixMilli(), 10)

	writeRecords := &timestreamwrite.WriteRecordsInput{
		DatabaseName: aws.String("orders"),
		TableName:    aws.String("checkins"),
		Records: []*timestreamwrite.Record{
			{
				Dimensions:       buildDimensions(order.Id, order.LocationNum),
				MeasureName:      aws.String("OrderMeasures"),
				MeasureValues:    buildMeasures(order),
				MeasureValueType: aws.String(timestreamwrite.MeasureValueTypeMulti),
				Time:             aws.String(orderTimestampMilli),
			},
		},
	}

	_, err := writer.WriteRecords(writeRecords)
	if err != nil {
		return err
	}
	return nil
}

func buildDimensions(orderId string, dim2 string) []*timestreamwrite.Dimension {
	return []*timestreamwrite.Dimension{
		{
			Name:  aws.String("OrderId"),
			Value: aws.String(orderId),
		},
		{
			Name:  aws.String("Dim2"),
			Value: aws.String(dim2),
		},
	}
}

func buildMeasures(order Order) []*timestreamwrite.MeasureValue {
	return []*timestreamwrite.MeasureValue{
		{
			Name:  aws.String("locationNum"),
			Value: aws.String(order.LocationNum),
			Type:  aws.String(timestreamwrite.MeasureValueTypeVarchar),
		},
		{
			Name:  aws.String("orderSubTotal"),
			Value: aws.String(fmt.Sprintf("%f", order.OrderSubTotal)),
			Type:  aws.String(timestreamwrite.MeasureValueTypeDouble),
		},
	}
}
