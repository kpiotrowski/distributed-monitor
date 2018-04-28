# distributed-monitor

## Opis
Celem projektu jest implementacja rozproszonego monitora.
W celu zapewnienia wzajemnego wykluczenia zastosowano algorytm Suzuki–Kasami.
W projekcie założono także że wszelkie operacje na synchronizowanych danych (read/write) mogą się
odbywać tylko w sekcji krytycznej.

### Cechy implementacji:
 - algorytm Suzuki-Kasami oparty na wymianie tokenu został użyty do zapewnienia wzajemnego
     wykluczenia.
 - Komunikacja pomiędzy węzłami odbywa się przy użyciu zeroMQ (PUSH/PULL)
 - Możliwość zdefiniowania wielu monitorów dla tego samego klastra węzłów
 - Możliwość powiązania wielu zmiennych warunkowych z jednym monitorem
 - Mozliwość powiązania danych z monitorem (np buforu)
 - Transparentne operacje na danych (warunkiem jest zbindowanie wskaźnika na zmienną!)
 - Zastosowano podejcie lazy przy wymianie współdzielonych danych (Są one przesyłane wraz z tokenem
     przy przyznawaniu sekcji krytycznej danemu węzłowi)


### Wymagania:
  - Biblioteka ZeroMQ w wersji conajmniej 4.0.1
  - Golang w wersji >= 1.8

### Instalacja:
 `go get "github.com/kpiotrowski/distributed-monitor"`
 Komenda ta pobierze projekt do odpowiedniego katalogu wymaganego przez Golang ($GOPATH/src/).

### Przykład
  W celu łatwego zaprezentowania projektu zaimplementowano rozwiązanie problemu
  producenta-konsumenta opartego na 3 węzłach. (1 producent, 2 konsumentów)
  Przykład znajduje się w katalogu examples/prodCons
  Przy uruchamianiu jako pierwszy argunemt należy podać adres danego węzłą a w kolejnych adresy
  innych węzłów. Przykład:
  - node1: 127.0.0.1:5551 127.0.0.1:5552
  - node2: 127.0.0.2:5552 127.0.0.2:5551

  Skrypty ./startNode1.sh, startNode2.sh, ./startNode3.sh

  W celu wyświetlenia logów typu DEBUG należy zmienić wartość zmiennej logLevel na log.DebugLevel
  (13 linia w main.go)


