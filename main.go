package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"centrobank/internal/client"
	"centrobank/internal/pkg/model"
)

//Для красоты надо бы всё это в сервисный слой определить
type ErrFetch struct {
	err error
	date time.Time
}

type Service struct {
	currencies map[time.Time]float64
	currencyName string
	startDate time.Time
	endDate time.Time
	mu *sync.RWMutex
}

func NewService (currencyName string, startDate, endDate time.Time) *Service{
	return &Service{
		currencies: make(map[time.Time]float64),
		currencyName: currencyName,
		startDate: startDate,
		endDate: endDate,
		mu : &sync.RWMutex{},
	}
}

type ResultAnswer struct {
	currencyName string
	max float64
	dateMin time.Time
	min float64
	dateMax time.Time
	avg float64
}

//TODO: MOCK
func main() {
	ctx := context.Background()
	//можно протестить несуществующую дату
	//endDate := time.Date(1800, 1, 1, 0, 0, 0, 0, time.UTC)
	
	client := client.NewClient()

	//Накручивать удобность можно бесконечно. Можно всё это завернуть в какую-то функцию и сервисы передавать массивом
	//Подходов масса
	endDateRUB := time.Now()
	startDateRUB := endDateRUB.AddDate(0, 0, -90)

	serviceRUB := NewService(model.CurrencyUSD, startDateRUB, endDateRUB)
	errCh := serviceRUB.FetchCurrencyRatesLastNDays(ctx, client)

	for err := range errCh {
		fmt.Println("error fetching data!", err.date, err.err)
	}

	if len(serviceRUB.currencies) == 0 {
		fmt.Println("No data found")

	}

	serviceRUB.GetAnswer()

	fmt.Println()
	fmt.Println()

	
	endDateCNY := time.Now()
	startDateCNY := endDateRUB.AddDate(0, 0, -90)

	serviceCNY := NewService(model.CurrencyCNY, startDateCNY, endDateCNY)
	errCh = serviceCNY.FetchCurrencyRatesLastNDays(ctx, client)

	for err := range errCh {
		fmt.Println("error fetching data!", err.date, err.err)
	}

	if len(serviceRUB.currencies) == 0 {
		fmt.Println("No data found")

	}

	serviceCNY.GetAnswer()

	//Можно добавить для остальных валют
	//Круче было бы спарсить вообще все валюты из цб, это-то понятно

	// for key, val := range serviceRUB.currencies {
	// 	fmt.Println(key, val)
	// }

	// fmt.Println()
	// fmt.Println()

	// for key, val := range serviceCNY.currencies {
	// 	fmt.Println(key, val)
	// }


}

//Надо бы это в сервисный слой определить
func (s * Service) FetchCurrencyRatesLastNDays (ctx context.Context, client client.IClient) <- chan ErrFetch {
	wg := sync.WaitGroup{}

	errCh := make(chan ErrFetch)

	go func () {
		defer close(errCh)
		for date := s.startDate; !date.After(s.endDate); date = date.AddDate(0, 0, 1) {
			wg.Add(1)
			go func (d time.Time) {
				defer wg.Done()
				exchangeRate, err := client.GetExchangeRate(ctx, s.currencyName, d)
				if err != nil {
					log.Printf("Error fetching rate for %s on %v: %v", s.currencyName, d, err)
					errCh <- ErrFetch{err: err, date: d}
	
				} else {
					s.mu.Lock()
					s.currencies[d] = exchangeRate
					s.mu.Unlock()
				}
			} (date)
		}
		wg.Wait()

	} ()

	return errCh
}


func (s * Service) GetAnswer () {
	s.mu.RLock()
	defer s.mu.RUnlock()

    if len(s.currencies) == 0 {
        fmt.Println("No data found")
        return
    }


	resAnswer := ResultAnswer{currencyName: s.currencyName, max: -1, min: math.MaxFloat64}
	
	for date, val := range s.currencies {
		resAnswer.avg+=val
		if val < resAnswer.min {
			resAnswer.min = val
			resAnswer.dateMin = date
		}
		if val > resAnswer.max {
			resAnswer.max = val
			resAnswer.dateMax = date
		}
	}

	resAnswer.avg = resAnswer.avg / float64(len(s.currencies))


    // Вывод в требуемом формате
    fmt.Printf("Валюта: %s, максимум %f, минимум %f, среднее %f\n", resAnswer.currencyName, resAnswer.max, resAnswer.min, resAnswer.avg)
    fmt.Printf("Дата максимума: %s\n", resAnswer.dateMax)
    fmt.Printf("Дата минимума: %s\n", resAnswer.dateMin)

}

//unused
func Convert(ctx context.Context, from string, to string, value float64, date time.Time, client client.IClient) (float64, error) {
	if from == to {
		return value, nil
	}
	if value == 0 {
		return 0, nil
	}

	res := value

	if from != model.CurrencyRUB {
		exchangeRate, err := client.GetExchangeRate(ctx, from, date)
		if err != nil {
			return 0, err
		}

		res = res * exchangeRate
	}

	if to != model.CurrencyRUB {
		exchangeRate, err := client.GetExchangeRate(ctx, to, date)
		if err != nil {
			return 0, err
		}

		res = res / exchangeRate
	}

	return (math.Floor(res*100) / 100), nil
}
