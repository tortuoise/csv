# csv

[![GoDoc](https:/godoc.org/github.com/tortuoise/csv?status.svg)](https://godoc.org/github.com/tortuoise/csv)

`csv` allows you unmarshal and marshal (TBD) between csv (Comma
Separated Values) text and Golang struct.

# Quick Example

```go
CSVText := `1, 2, 3, "test"
4, 5, 6, "another_test"`

type T struct {
    F1 int
    F2 int
    F3 float32
    F4 string
}
dec := csv.NewDecoder(strings.NewReader(CSVText))
dec.TrimLeadingSpace = true

t := T{}
for {
    err := dec.Decode(&t)
     if err == io.EOF {
         break
     } else if err != nil {
         fmt.Printf("error: %v\n", err)
     }
     fmt.Println(t)
}
//Output:
//{1 2 3 test}
//{4 5 6 another_test}
```
# Embedded structs & slices

```go
type Good struct {
        Id int64
        Price float64
        Vols []int `cap:"2"` // the struct field is used to determine the number of csv values to assign this field
        Deets GoodDeets
}
type GoodDeets struct {
        Stock bool
        Related []int `cap:"2"`
        Features []string `cap:"3"`
}
input := `"0", "0.0", "10", "1", "false", "1002", "1003", "Boo", "Yeah", "Banjo"
        "1", "2.1", "11", "12", "true", "1002", "1005", "Boo", "Nay", "Banjo"`
expect := Good{}
dec := csv.NewDecoder(strings.NewReader(input))
get := reflect.New(reflect.TypeOf(expect)).Interface() // or get := Good{}
for {
        err := dec.DeepUnmarshalCSV(get)
        if err == io.EOF {
                break
        } else if err != nil  {
                fmt.Printf("error:%v\n", err)
        }
        fmt.Println(get)
}
//Output:
//&{0 0 [10 1] {false [1002 1003] [Boo Yeah Banjo]}}
//&{1 2.1 [11 12] {true [1002 1005] [Boo Nay Banjo]}}
```
